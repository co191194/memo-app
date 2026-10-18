package main

import (
	"bufio"
	"bytes"
	"errors"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"syscall"
	"testing"
	"time"
)

// serveProcessTimeout は、起動メッセージや終了を待つ上限。
const serveProcessTimeout = 10 * time.Second

var listeningPattern = regexp.MustCompile(`^Listening on (http://127\.0\.0\.1:(\d+))\n$`)

type serveProcess struct {
	cmd    *exec.Cmd
	stderr *bytes.Buffer
	url    string
	port   string
	exited chan struct{} // プロセスが終了したら閉じる
	err    error         // Wait の結果。exited が閉じた後に読む
}

// startServe は memo serve をプロセスとして起動し、標準出力の起動メッセージを待つ。
func startServe(t *testing.T, binary, cwd, home, memoPath string, args ...string) *serveProcess {
	t.Helper()

	cmd := exec.Command(binary, append([]string{"serve"}, args...)...)
	cmd.Dir = cwd
	cmd.Env = append(os.Environ(), "HOME="+home, "MEMO_PATH="+memoPath, "TZ=UTC")
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatal(err)
	}
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Start(); err != nil {
		t.Fatalf("start memo serve: %v", err)
	}

	p := &serveProcess{cmd: cmd, stderr: &stderr, exited: make(chan struct{})}

	// 起動メッセージの1行目だけを読む。読み終えてから Wait を始める。
	lines := make(chan string, 1)
	go func() {
		line, _ := bufio.NewReader(stdout).ReadString('\n')
		lines <- line
	}()

	var line string
	select {
	case line = <-lines:
	case <-time.After(serveProcessTimeout):
		cmd.Process.Kill()
		t.Fatalf("timed out waiting for the listening message")
	}

	go func() {
		p.err = cmd.Wait()
		close(p.exited)
	}()
	t.Cleanup(p.kill)

	matches := listeningPattern.FindStringSubmatch(line)
	if matches == nil {
		p.kill()
		t.Fatalf("stdout first line = %q, expected %q; stderr = %q",
			line, "Listening on http://127.0.0.1:<port>\n", stderr.String())
	}
	p.url = matches[1]
	p.port = matches[2]
	return p
}

// kill は、まだ動いているプロセスを強制終了し、終了を待つ。
func (p *serveProcess) kill() {
	select {
	case <-p.exited:
	default:
		p.cmd.Process.Kill()
		<-p.exited
	}
}

// stop はシグナルを送り、プロセスの終了コードを返す。
// シグナルで強制終了した場合（シグナルを処理していない場合）は -1 になる。
func (p *serveProcess) stop(t *testing.T, sig os.Signal) int {
	t.Helper()

	if err := p.cmd.Process.Signal(sig); err != nil {
		t.Fatalf("send %v: %v", sig, err)
	}

	select {
	case <-p.exited:
		if p.err == nil {
			return 0
		}
		var exitErr *exec.ExitError
		if !errors.As(p.err, &exitErr) {
			t.Fatalf("wait memo serve: %v", p.err)
		}
		return exitErr.ExitCode()
	case <-time.After(serveProcessTimeout):
		t.Fatalf("memo serve did not exit after %v", sig)
		return 0
	}
}

// getWithoutRedirect は、リダイレクトをたどらずにGETを送る。
func getWithoutRedirect(t *testing.T, url string) *http.Response {
	t.Helper()

	client := &http.Client{
		Timeout: serveProcessTimeout,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
	resp, err := client.Get(url)
	if err != nil {
		t.Fatalf("GET %s: %v", url, err)
	}
	t.Cleanup(func() { resp.Body.Close() })
	return resp
}

func TestServeProcess(t *testing.T) {
	binary := buildMemo(t)
	cwd := t.TempDir()
	home := t.TempDir()
	memoPath := filepath.Join(cwd, "memos.json")

	// AC-01
	t.Run("既定ではポート8080で待ち受ける", func(t *testing.T) {
		// 8080がほかのプロセスに使われている環境では確認できないため、スキップする
		ln, err := net.Listen("tcp", "127.0.0.1:8080")
		if err != nil {
			t.Skipf("127.0.0.1:8080 is not available: %v", err)
		}
		ln.Close()

		p := startServe(t, binary, cwd, home, memoPath)
		if p.url != "http://127.0.0.1:8080" {
			t.Errorf("url = %q, expected = %q", p.url, "http://127.0.0.1:8080")
		}
		p.stop(t, syscall.SIGINT)
	})

	// AC-02
	t.Run("--port 0ではOSが選んだポートで待ち受ける", func(t *testing.T) {
		p := startServe(t, binary, cwd, home, memoPath, "--port", "0")
		if p.port == "0" {
			t.Errorf("port = 0, expected the actual listening port")
		}

		resp := getWithoutRedirect(t, p.url+"/")
		if resp.StatusCode != http.StatusFound {
			t.Errorf("GET / status = %d, expected = %d", resp.StatusCode, http.StatusFound)
		}
		p.stop(t, syscall.SIGINT)
	})

	// AC-03
	for _, tc := range []struct {
		name string
		sig  os.Signal
	}{
		{"SIGINT", syscall.SIGINT},
		{"SIGTERM", syscall.SIGTERM},
	} {
		t.Run(tc.name+"で終了コード0で終了する", func(t *testing.T) {
			p := startServe(t, binary, cwd, home, memoPath, "--port", "0")

			if code := p.stop(t, tc.sig); code != 0 {
				t.Errorf("exit code = %d, expected = 0; stderr = %q", code, p.stderr.String())
			}
		})
	}
}

// AC-08
func TestServeProcessMemoList(t *testing.T) {
	binary := buildMemo(t)
	cwd := t.TempDir()
	home := t.TempDir()
	memoPath := filepath.Join(cwd, "memos.json")

	// CLIで追加したメモを、同じ保存先で起動したサーバーが表示する
	expectProcess(t, runMemo(t, binary, cwd, home, memoPath, "add", "Added from CLI"), 0, "", "")

	p := startServe(t, binary, cwd, home, memoPath, "--port", "0")
	status, body := getMemoList(t, p.url)

	if status != http.StatusOK {
		t.Errorf("GET /memos status = %d, expected = %d", status, http.StatusOK)
	}
	if !strings.Contains(body, "Added from CLI") {
		t.Errorf("GET /memos body = %q, expected to contain %q", body, "Added from CLI")
	}
	p.stop(t, syscall.SIGINT)
}

// getMemoList は GET /memos を送り、ステータスと本文を返す。
func getMemoList(t *testing.T, baseURL string) (int, string) {
	t.Helper()

	resp := getWithoutRedirect(t, baseURL+"/memos")
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read GET /memos body: %v", err)
	}
	return resp.StatusCode, string(body)
}

// AC-17
func TestServeProcessReadsMemosPerRequest(t *testing.T) {
	binary := buildMemo(t)
	cwd := t.TempDir()
	home := t.TempDir()
	memoPath := filepath.Join(cwd, "memos.json")

	p := startServe(t, binary, cwd, home, memoPath, "--port", "0")

	// 起動時点ではまだ表示されない
	if _, body := getMemoList(t, p.url); strings.Contains(body, "Added while serving") {
		t.Fatalf("GET /memos body = %q, expected not to contain the memo before it is added", body)
	}

	// サーバーを動かしたままCLIで追加すると、再起動せずに表示される
	expectProcess(t, runMemo(t, binary, cwd, home, memoPath, "add", "Added while serving"), 0, "", "")

	status, body := getMemoList(t, p.url)
	if status != http.StatusOK {
		t.Errorf("GET /memos status = %d, expected = %d", status, http.StatusOK)
	}
	if !strings.Contains(body, "Added while serving") {
		t.Errorf("GET /memos body = %q, expected to contain %q", body, "Added while serving")
	}
	p.stop(t, syscall.SIGINT)
}

// AC-24
func TestServeProcessRecoversFromLoadFailure(t *testing.T) {
	binary := buildMemo(t)
	cwd := t.TempDir()
	home := t.TempDir()
	memoPath := filepath.Join(cwd, "memos.json")
	if err := os.WriteFile(memoPath, []byte("{invalid"), 0o600); err != nil {
		t.Fatal(err)
	}

	// 起動時には保存先を読み込まないので、不正なJSONでも起動に成功する
	p := startServe(t, binary, cwd, home, memoPath, "--port", "0")

	if status, _ := getMemoList(t, p.url); status != http.StatusInternalServerError {
		t.Errorf("1st GET /memos status = %d, expected = %d", status, http.StatusInternalServerError)
	}

	// JSONを有効な内容に直すと、同じサーバーが次のリクエストで表示できる
	validJSON := `[{"id":1,"title":"Recovered memo","body":"","created_at":"2026-10-18T09:00:00Z","updated_at":"2026-10-18T09:00:00Z"}]`
	if err := os.WriteFile(memoPath, []byte(validJSON), 0o600); err != nil {
		t.Fatal(err)
	}

	status, body := getMemoList(t, p.url)
	if status != http.StatusOK {
		t.Errorf("2nd GET /memos status = %d, expected = %d", status, http.StatusOK)
	}
	if !strings.Contains(body, "Recovered memo") {
		t.Errorf("2nd GET /memos body = %q, expected to contain %q", body, "Recovered memo")
	}
	p.stop(t, syscall.SIGINT)
}
