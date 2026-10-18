package web_test

import (
	"bytes"
	"html/template"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/co191194/memo-app/internal/memo"
	"github.com/co191194/memo-app/internal/service"
	"github.com/co191194/memo-app/internal/storage"
	"github.com/co191194/memo-app/internal/web"
)

type Memo = memo.Memo

// fakeMemoReader は、保存先を使わずに決まったメモを返すMemoReader。
type fakeMemoReader struct {
	memos []Memo
}

// ListMemos は保存先から読み込むたびに新しいスライスを返す本物に合わせて、コピーを返す。
// テスト側の期待値と同じスライスを渡すと、ハンドラーが並べ替えたときに期待値まで変わってしまう。
func (f *fakeMemoReader) ListMemos() ([]Memo, error) {
	return slices.Clone(f.memos), nil
}

func (f *fakeMemoReader) GetMemo(id int) (Memo, error) {
	return memo.FindById(f.memos, id)
}

// serve はハンドラーへリクエストを1件送り、記録した応答を返す。
// httptest.NewRequest はサーバーを起動せずにリクエストを作り、
// httptest.NewRecorder はステータス・ヘッダー・本文を記録するResponseWriterとして働く。
// ヘッダーは rec.Header() ではなく rec.Result().Header で確認する。rec.Header() はハンドラーが
// 書き換えるマップそのもので、WriteHeader の後に設定した（実際には送られない）値も見えてしまう。
// rec.Result().Header は、本物のサーバーと同じく WriteHeader の時点の値を返す。
func serve(t *testing.T, handler http.Handler, method string, target string) *httptest.ResponseRecorder {
	t.Helper()

	req := httptest.NewRequest(method, target, nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	return rec
}

// AC-06
func TestHandler_RootRedirectsToMemos(t *testing.T) {
	handler := web.NewHandler(&fakeMemoReader{}, io.Discard)

	rec := serve(t, handler, http.MethodGet, "/")

	if rec.Code != http.StatusFound {
		t.Errorf("status = %d, expected = %d", rec.Code, http.StatusFound)
	}
	if location := rec.Result().Header.Get("Location"); location != "/memos" {
		t.Errorf("Location = %q, expected = %q", location, "/memos")
	}
}

// AC-07
func TestHandler_StyleSheet(t *testing.T) {
	handler := web.NewHandler(&fakeMemoReader{}, io.Discard)

	rec := serve(t, handler, http.MethodGet, "/static/style.css")

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, expected = %d", rec.Code, http.StatusOK)
	}
	if contentType := rec.Result().Header.Get("Content-Type"); !strings.HasPrefix(contentType, "text/css") {
		t.Errorf("Content-Type = %q, expected to start with %q", contentType, "text/css")
	}
	// バイナリに同梱したファイルの中身が返っていること
	if rec.Body.Len() == 0 {
		t.Errorf("body is empty")
	}
}

// AC-25（/ のHEAD）
func TestHandler_HeadRoot(t *testing.T) {
	handler := web.NewHandler(&fakeMemoReader{}, io.Discard)

	rec := serve(t, handler, http.MethodHead, "/")

	if rec.Code != http.StatusFound {
		t.Errorf("status = %d, expected = %d", rec.Code, http.StatusFound)
	}
	if location := rec.Result().Header.Get("Location"); location != "/memos" {
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
	handler := web.NewHandler(&fakeMemoReader{}, io.Discard)

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
			if contentType := rec.Result().Header.Get("Content-Type"); contentType != "text/html; charset=utf-8" {
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

// GET・HEAD以外のメソッドは405（Spec 4章、AC-14）
func TestHandler_MethodNotAllowed(t *testing.T) {
	handler := web.NewHandler(&fakeMemoReader{}, io.Discard)

	for _, target := range []string{"/", "/static/style.css", "/memos"} {
		t.Run(target, func(t *testing.T) {
			rec := serve(t, handler, http.MethodPost, target)

			if rec.Code != http.StatusMethodNotAllowed {
				t.Errorf("status = %d, expected = %d", rec.Code, http.StatusMethodNotAllowed)
			}
		})
	}
}

// AC-04
func TestHandler_MemoList(t *testing.T) {
	// IDの昇順ではなく保存した順で表示されることを確認するため、IDを入れ替えて並べる。
	// 作成日もメモごとに変え、各メモの行に自分の作成日が出ていることを確認できるようにする
	memos := []Memo{
		{ID: 3, Title: "Third", CreatedAt: time.Date(2026, 10, 1, 9, 0, 0, 0, time.UTC)},
		{ID: 1, Title: "First", CreatedAt: time.Date(2026, 9, 15, 18, 30, 0, 0, time.UTC)},
		{ID: 2, Title: "Second", CreatedAt: time.Date(2026, 10, 5, 0, 0, 0, 0, time.UTC)},
	}
	handler := web.NewHandler(&fakeMemoReader{memos: memos}, io.Discard)

	rec := serve(t, handler, http.MethodGet, "/memos")

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, expected = %d", rec.Code, http.StatusOK)
	}
	if contentType := rec.Result().Header.Get("Content-Type"); contentType != "text/html; charset=utf-8" {
		t.Errorf("Content-Type = %q, expected = %q", contentType, "text/html; charset=utf-8")
	}
	body := rec.Body.String()
	if !strings.Contains(body, "<title>メモ一覧 - memo</title>") {
		t.Errorf("body = %q, expected to contain the page title %q", body, "メモ一覧 - memo")
	}

	// メモごとに「IDだけの要素 → 詳細へのリンク → 作成日」の順に並び、
	// それが保存した順に続いていることを、1つの正規表現で確認する
	pattern := memoListPattern(memos)
	if !pattern.MatchString(body) {
		t.Errorf("body = %q, expected to match %q", body, pattern)
	}
}

// memoListPattern は、一覧画面の本文に memos が順に表示されていることを表す正規表現を作る。
// メモ1件ごとに次の3つがこの順に現れることを求める。間のタグや空白は問わない。
//   - IDだけを中身に持つ要素（例: <td>3</td>）
//   - 詳細画面へのリンク（例: <a href="/memos/3">Third</a>）
//   - 作成日（例: 2026-10-01）
func memoListPattern(memos []Memo) *regexp.Regexp {
	var sb strings.Builder
	sb.WriteString(`(?s)`) // . が改行にも一致するようにする
	for _, m := range memos {
		id := strconv.Itoa(m.ID)
		sb.WriteString(`.*?>\s*` + id + `\s*<`)
		sb.WriteString(`.*?<a href="/memos/` + id + `">\s*` + regexp.QuoteMeta(m.Title) + `\s*</a>`)
		sb.WriteString(`.*?` + m.CreatedAt.Format("2006-01-02"))
	}
	return regexp.MustCompile(sb.String())
}

// newStorageReader は、本物のサービス層と保存先の読み込みを使うMemoReaderを作る。
// 保存先ファイルがない場合や不正なJSONの場合を、実際の読み込みで確認するために使う。
func newStorageReader(path string) *service.MemoService {
	return &service.MemoService{MemoPath: path, Loader: &storage.StorageOperatorImpl{}}
}

// AC-09
func TestHandler_MemoList_Empty(t *testing.T) {
	testCases := []struct {
		testName string
		reader   web.MemoReader
	}{
		{"メモが0件の場合", &fakeMemoReader{memos: []Memo{}}},
		{"保存先ファイルがない場合", newStorageReader(filepath.Join(t.TempDir(), "memos.json"))},
	}

	for _, tc := range testCases {
		t.Run(tc.testName, func(t *testing.T) {
			handler := web.NewHandler(tc.reader, io.Discard)

			rec := serve(t, handler, http.MethodGet, "/memos")

			if rec.Code != http.StatusOK {
				t.Errorf("status = %d, expected = %d", rec.Code, http.StatusOK)
			}
			if body := rec.Body.String(); !strings.Contains(body, "メモはありません") {
				t.Errorf("body = %q, expected to contain %q", body, "メモはありません")
			}
		})
	}
}

// AC-12（一覧）
func TestHandler_MemoList_EscapesTitle(t *testing.T) {
	title := `<script>alert("x")</script>`
	memos := []Memo{
		{ID: 1, Title: title, CreatedAt: time.Date(2026, 10, 1, 9, 0, 0, 0, time.UTC)},
	}
	handler := web.NewHandler(&fakeMemoReader{memos: memos}, io.Discard)

	rec := serve(t, handler, http.MethodGet, "/memos")

	body := rec.Body.String()
	// タグとして出力されていないこと
	if strings.Contains(body, title) {
		t.Errorf("body = %q, expected not to contain the raw title %q", body, title)
	}
	// 文字列として表示されるよう、エスケープして出力されていること
	escaped := template.HTMLEscapeString(title)
	if !strings.Contains(body, escaped) {
		t.Errorf("body = %q, expected to contain the escaped title %q", body, escaped)
	}
}

// AC-13（一覧）
func TestHandler_MemoList_LoadFailure(t *testing.T) {
	path := filepath.Join(t.TempDir(), "memos.json")
	invalidJSON := []byte("{invalid")
	if err := os.WriteFile(path, invalidJSON, 0o600); err != nil {
		t.Fatal(err)
	}
	reader := newStorageReader(path)

	// 画面に出してはいけない「エラーの詳細」を、同じ読み込みを実行して得ておく
	_, loadErr := reader.ListMemos()
	if loadErr == nil {
		t.Fatal("ListMemos() err = nil, expected a load error for invalid JSON")
	}
	detail := loadErr.Error()

	var errorLog bytes.Buffer
	handler := web.NewHandler(reader, &errorLog)

	rec := serve(t, handler, http.MethodGet, "/memos")

	if rec.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, expected = %d", rec.Code, http.StatusInternalServerError)
	}
	if contentType := rec.Result().Header.Get("Content-Type"); contentType != "text/html; charset=utf-8" {
		t.Errorf("Content-Type = %q, expected = %q", contentType, "text/html; charset=utf-8")
	}
	body := rec.Body.String()
	if !strings.Contains(body, "メモを読み込めませんでした") {
		t.Errorf("body = %q, expected to contain %q", body, "メモを読み込めませんでした")
	}
	if !strings.Contains(body, `href="/memos"`) {
		t.Errorf("body = %q, expected to contain a link to /memos", body)
	}

	// 画面にエラーの詳細やファイルパスを出さない（エスケープされた形でも出さない）
	for _, secret := range []string{detail, path} {
		if strings.Contains(body, secret) || strings.Contains(body, template.HTMLEscapeString(secret)) {
			t.Errorf("body = %q, expected not to contain %q", body, secret)
		}
	}

	// 標準エラー（errorLog）には、詳細を1行で出す
	log := errorLog.String()
	if !strings.Contains(log, detail) {
		t.Errorf("errorLog = %q, expected to contain %q", log, detail)
	}
	if strings.Count(log, "\n") != 1 || !strings.HasSuffix(log, "\n") {
		t.Errorf("errorLog = %q, expected exactly one line", log)
	}

	// 保存先ファイルは変化しない
	after, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(after, invalidJSON) {
		t.Errorf("file = %q, expected unchanged %q", after, invalidJSON)
	}
}

// AC-25（/memos のHEAD）
// ResponseRecorder はハンドラーが書いた本文をそのまま記録するが、本物のサーバーはHEADの本文を送らない。
// クライアントが実際に受け取る応答を確かめるため、httptest.NewServer で本物のサーバーを立てて比べる。
func TestHandler_HeadMemoList(t *testing.T) {
	memos := []Memo{
		{ID: 1, Title: "First", CreatedAt: time.Date(2026, 10, 1, 9, 0, 0, 0, time.UTC)},
	}
	server := httptest.NewServer(web.NewHandler(&fakeMemoReader{memos: memos}, io.Discard))
	t.Cleanup(server.Close)

	getResp := request(t, http.MethodGet, server.URL+"/memos")
	headResp := request(t, http.MethodHead, server.URL+"/memos")

	if headResp.status != http.StatusOK || headResp.status != getResp.status {
		t.Errorf("HEAD status = %d, GET status = %d, expected both = %d", headResp.status, getResp.status, http.StatusOK)
	}
	if headResp.contentType != getResp.contentType {
		t.Errorf("HEAD Content-Type = %q, expected to match GET %q", headResp.contentType, getResp.contentType)
	}
	if headResp.body != "" {
		t.Errorf("HEAD body = %q, expected empty", headResp.body)
	}
}

type clientResponse struct {
	status      int
	contentType string
	body        string
}

// request は本物のHTTPクライアントでリクエストを1件送り、応答を返す。
func request(t *testing.T, method string, url string) clientResponse {
	t.Helper()

	req, err := http.NewRequest(method, url, nil)
	if err != nil {
		t.Fatal(err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("%s %s: %v", method, url, err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read %s %s body: %v", method, url, err)
	}
	return clientResponse{resp.StatusCode, resp.Header.Get("Content-Type"), string(body)}
}
