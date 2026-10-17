package cli_test

import (
	"bytes"
	"net"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	"github.com/co191194/memo-app/internal/storage"
)

// serveReturnTimeout は、起動に失敗するはずのserveが戻るまでの上限。
// 誤ってサーバーが起動した場合に、テストが止まり続けないようにする。
const serveReturnTimeout = 5 * time.Second

// runServe は memo serve をApp経由で実行し、標準出力・標準エラー・終了コードを返す。
func runServe(t *testing.T, args []string) (string, string, int) {
	t.Helper()

	app := App{Command: &MemoCommandImpl{
		MemoPath:        filepath.Join(t.TempDir(), "memos.json"),
		TimeProvider:    &RealTimeProvider{},
		StorageOperator: &storage.StorageOperatorImpl{},
	}}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	exitCodes := make(chan int, 1)
	go func() {
		exitCodes <- app.Run(&stdout, &stderr, append([]string{"serve"}, args...))
	}()

	select {
	case exitCode := <-exitCodes:
		return stdout.String(), stderr.String(), exitCode
	case <-time.After(serveReturnTimeout):
		t.Fatalf("memo serve %q did not return; the server seems to have started", args)
		return "", "", 0
	}
}

// AC-15
func TestServe_InvalidArgs(t *testing.T) {
	// 使用中のポートを用意する。ポート番号はOSに選ばせる
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { ln.Close() })
	usedPort := strconv.Itoa(ln.Addr().(*net.TCPAddr).Port)

	testCases := []struct {
		testName string
		args     []string
	}{
		{"ポートが数値でない場合", []string{"--port", "abc"}},
		{"ポートが負の数の場合", []string{"--port", "-1"}},
		{"ポートが65535より大きい場合", []string{"--port", "65536"}},
		{"未定義のオプションの場合", []string{"--unknown"}},
		{"位置引数がある場合", []string{"extra"}},
		{"ポートが使用中の場合", []string{"--port", usedPort}},
	}

	for _, tc := range testCases {
		t.Run(tc.testName, func(t *testing.T) {
			stdout, stderr, exitCode := runServe(t, tc.args)

			if exitCode != 1 {
				t.Errorf("exitCode = %d, expected = 1", exitCode)
			}
			if stderr == "" {
				t.Errorf("stderr is empty, expected an error message")
			}
			// 起動メッセージを出さないこと
			if stdout != "" {
				t.Errorf("stdout = %q, expected empty", stdout)
			}
		})
	}
}

// AC-16
func TestServe_Help(t *testing.T) {
	usage := "Usage:\n" +
		"  memo serve [--port <port>]\n" +
		"  -port int\n" +
		"    \tポート番号 (default 8080)\n"

	for _, option := range []string{"--help", "-h"} {
		t.Run(option, func(t *testing.T) {
			stdout, stderr, exitCode := runServe(t, []string{option})

			if exitCode != 0 {
				t.Errorf("exitCode = %d, expected = 0", exitCode)
			}
			if stdout != "" || stderr != usage {
				t.Errorf("stdout = %q, stderr = %q, expected stderr = %q", stdout, stderr, usage)
			}
		})
	}
}
