package web_test

import (
	"context"
	"io"
	"net"
	"net/http"
	"testing"
	"time"

	"github.com/co191194/memo-app/internal/web"
)

// testTimeout は、テストで何かを待つときの上限。
const testTimeout = 5 * time.Second

// blockingHandler は、release が閉じられるまで応答を返さないハンドラー。
// 「処理中のリクエストがある状態」を作るために使う。
type blockingHandler struct {
	started chan struct{} // ハンドラーが呼ばれたら閉じる
	release chan struct{} // 閉じるとハンドラーが応答を返す
}

func newBlockingHandler() *blockingHandler {
	return &blockingHandler{
		started: make(chan struct{}),
		release: make(chan struct{}),
	}
}

// ServeHTTP は1回だけ呼ばれる前提で started を閉じる。
func (h *blockingHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	close(h.started)
	select {
	case <-h.release:
		io.WriteString(w, "done")
	case <-r.Context().Done(): // サーバーが接続を切った
	}
}

// startServe は127.0.0.1の空いているポートで web.Serve を goroutine で動かす。
// 待ち受けアドレスと、Serve の戻り値が届く channel を返す。
func startServe(t *testing.T, ctx context.Context, handler http.Handler, shutdownTimeout time.Duration) (string, <-chan error) {
	t.Helper()

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}

	done := make(chan error, 1)
	go func() {
		done <- web.Serve(ctx, ln, handler, shutdownTimeout)
	}()
	return ln.Addr().String(), done
}

type response struct {
	status int
	body   string
	err    error
}

// get は goroutine でGETを送り、結果が届く channel を返す。
// 応答を待っている間に、テスト側で終了の操作をするために使う。
func get(url string) <-chan response {
	responses := make(chan response, 1)
	go func() {
		client := &http.Client{Timeout: testTimeout}
		resp, err := client.Get(url)
		if err != nil {
			responses <- response{err: err}
			return
		}
		defer resp.Body.Close()
		body, err := io.ReadAll(resp.Body)
		responses <- response{status: resp.StatusCode, body: string(body), err: err}
	}()
	return responses
}

// receive は channel から1件受け取る。testTimeout を過ぎたらテストを失敗させる。
func receive[T any](t *testing.T, ch <-chan T, what string) T {
	t.Helper()

	select {
	case v := <-ch:
		return v
	case <-time.After(testTimeout):
		t.Fatalf("timed out waiting for %s", what)
		var zero T
		return zero
	}
}

// waitUntilRefused は、addr への新しい接続が拒否されるようになるまで待つ。
func waitUntilRefused(t *testing.T, addr string) {
	t.Helper()

	deadline := time.Now().Add(testTimeout)
	for time.Now().Before(deadline) {
		conn, err := net.DialTimeout("tcp", addr, 100*time.Millisecond)
		if err != nil {
			return
		}
		conn.Close()
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("%s still accepts new connections", addr)
}

// AC-22
func TestServe_WaitsForInFlightRequest(t *testing.T) {
	handler := newBlockingHandler()
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	addr, done := startServe(t, ctx, handler, testTimeout)
	responses := get("http://" + addr + "/")
	receive(t, handler.started, "the handler to be called")

	// 終了シグナルを受けたのと同じ状態にする（CLIでは signal.NotifyContext がこの ctx を終了させる）
	cancel()

	// シグナルの後の新しい接続は受け付けない
	waitUntilRefused(t, addr)

	// 処理中のリクエストがあるので、Serve はまだ戻らない
	select {
	case err := <-done:
		t.Fatalf("Serve returned %v before the in-flight request completed", err)
	default:
	}

	// 処理中のリクエストを完了させると、応答が届き、Serve が nil で戻る
	close(handler.release)

	resp := receive(t, responses, "the response")
	if resp.err != nil {
		t.Fatalf("GET err = %v, expected the in-flight request to complete", resp.err)
	}
	if resp.status != http.StatusOK || resp.body != "done" {
		t.Errorf("status = %d, body = %q, expected = %d, %q", resp.status, resp.body, http.StatusOK, "done")
	}
	if err := receive(t, done, "Serve to return"); err != nil {
		t.Errorf("Serve() err = %v, expected nil", err)
	}
}

// AC-23
func TestServe_ClosesConnectionsAfterTimeout(t *testing.T) {
	handler := newBlockingHandler() // release を閉じないので、自分からは完了しない
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	// 既定の5秒を待たずに済むよう、待機時間を短く差し替える
	shutdownTimeout := 100 * time.Millisecond
	addr, done := startServe(t, ctx, handler, shutdownTimeout)
	responses := get("http://" + addr + "/")
	receive(t, handler.started, "the handler to be called")

	start := time.Now()
	cancel()

	// 待機時間を過ぎたら、完了していないリクエストがあっても nil で戻る
	if err := receive(t, done, "Serve to return"); err != nil {
		t.Errorf("Serve() err = %v, expected nil", err)
	}
	if elapsed := time.Since(start); elapsed < shutdownTimeout {
		t.Errorf("Serve returned after %v, expected to wait at least %v", elapsed, shutdownTimeout)
	}

	// 接続が切られたので、クライアントには応答が届かずエラーになる
	resp := receive(t, responses, "the client to finish")
	if resp.err == nil {
		t.Errorf("GET status = %d, expected the connection to be closed", resp.status)
	}
}
