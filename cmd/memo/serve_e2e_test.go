package main

import (
	"bufio"
	"bytes"
	"errors"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
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
