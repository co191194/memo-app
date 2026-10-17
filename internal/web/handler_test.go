package web_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/co191194/memo-app/internal/memo"
	"github.com/co191194/memo-app/internal/web"
)

type Memo = memo.Memo

// fakeMemoReader は、保存先を使わずに決まったメモを返すMemoReader。
type fakeMemoReader struct {
	memos []Memo
}

func (f *fakeMemoReader) ListMemos() ([]Memo, error) {
	return f.memos, nil
}

func (f *fakeMemoReader) GetMemo(id int) (Memo, error) {
	return memo.FindById(f.memos, id)
}

// serve はハンドラーへリクエストを1件送り、記録した応答を返す。
// httptest.NewRequest はサーバーを起動せずにリクエストを作り、
// httptest.NewRecorder はステータス・ヘッダー・本文を記録するResponseWriterとして働く。
func serve(t *testing.T, handler http.Handler, method string, target string) *httptest.ResponseRecorder {
	t.Helper()

	req := httptest.NewRequest(method, target, nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	return rec
}

// AC-06
func TestHandler_RootRedirectsToMemos(t *testing.T) {
	handler := web.NewHandler(&fakeMemoReader{})

	rec := serve(t, handler, http.MethodGet, "/")

	if rec.Code != http.StatusFound {
		t.Errorf("status = %d, expected = %d", rec.Code, http.StatusFound)
	}
	if location := rec.Header().Get("Location"); location != "/memos" {
		t.Errorf("Location = %q, expected = %q", location, "/memos")
	}
}

// AC-07
func TestHandler_StyleSheet(t *testing.T) {
	handler := web.NewHandler(&fakeMemoReader{})

	rec := serve(t, handler, http.MethodGet, "/static/style.css")

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, expected = %d", rec.Code, http.StatusOK)
	}
	if contentType := rec.Header().Get("Content-Type"); !strings.HasPrefix(contentType, "text/css") {
		t.Errorf("Content-Type = %q, expected to start with %q", contentType, "text/css")
	}
	// バイナリに同梱したファイルの中身が返っていること
	if rec.Body.Len() == 0 {
		t.Errorf("body is empty")
	}
}

// AC-25（/ のHEAD）
func TestHandler_HeadRoot(t *testing.T) {
	handler := web.NewHandler(&fakeMemoReader{})

	rec := serve(t, handler, http.MethodHead, "/")

	if rec.Code != http.StatusFound {
		t.Errorf("status = %d, expected = %d", rec.Code, http.StatusFound)
	}
	if location := rec.Header().Get("Location"); location != "/memos" {
		t.Errorf("Location = %q, expected = %q", location, "/memos")
	}
	// ResponseRecorder はハンドラーが書いた本文をそのまま記録するため、
	// ハンドラー自身がHEADに本文を書いていないことを確認できる
	if rec.Body.Len() != 0 {
		t.Errorf("body = %q, expected empty", rec.Body.String())
	}
}

// 未定義のパスの404画面（Spec 4章・6章）
func TestHandler_UndefinedPath(t *testing.T) {
	handler := web.NewHandler(&fakeMemoReader{})

	testCases := []struct {
		testName string
		method   string
		target   string
	}{
		{"未定義のパスへのGET", http.MethodGet, "/undefined"},
		// メソッドが違っても、パスが未定義なら405ではなく404
		{"未定義のパスへのPOST", http.MethodPost, "/undefined"},
		// /static/ の下でも、定義したstyle.css以外は同じ404画面
		{"存在しない静的ファイル", http.MethodGet, "/static/missing.css"},
	}

	for _, tc := range testCases {
		t.Run(tc.testName, func(t *testing.T) {
			rec := serve(t, handler, tc.method, tc.target)

			if rec.Code != http.StatusNotFound {
				t.Errorf("status = %d, expected = %d", rec.Code, http.StatusNotFound)
			}
			if contentType := rec.Header().Get("Content-Type"); contentType != "text/html; charset=utf-8" {
				t.Errorf("Content-Type = %q, expected = %q", contentType, "text/html; charset=utf-8")
			}
			body := rec.Body.String()
			if !strings.Contains(body, "ページが見つかりません") {
				t.Errorf("body = %q, expected to contain %q", body, "ページが見つかりません")
			}
			if !strings.Contains(body, `href="/memos"`) {
				t.Errorf("body = %q, expected to contain a link to /memos", body)
			}
		})
	}
}

// GET・HEAD以外のメソッドは405（Spec 4章）
func TestHandler_MethodNotAllowed(t *testing.T) {
	handler := web.NewHandler(&fakeMemoReader{})

	for _, target := range []string{"/", "/static/style.css"} {
		t.Run(target, func(t *testing.T) {
			rec := serve(t, handler, http.MethodPost, target)

			if rec.Code != http.StatusMethodNotAllowed {
				t.Errorf("status = %d, expected = %d", rec.Code, http.StatusMethodNotAllowed)
			}
		})
	}
}
