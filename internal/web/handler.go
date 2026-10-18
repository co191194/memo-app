package web

import (
	"errors"
	"fmt"
	"html/template"
	"io"
	"net/http"
	"strconv"

	"github.com/co191194/memo-app/internal/memo"
	assets "github.com/co191194/memo-app/web"
)

type MemoReader interface {
	ListMemos() ([]memo.Memo, error)
	GetMemo(id int) (memo.Memo, error)
}

func NewHandler(memos MemoReader, errorLog io.Writer) http.Handler {
	mux := http.NewServeMux()

	h := handler{
		errorTemplate: getTemplate("error"),
		memoList:      getTemplate("memo-list"),
		memoDetail:    getTemplate("memo-detail"),
		memos:         memos,
		errorLog:      errorLog,
	}

	mux.HandleFunc("/", h.notFoundHandler)
	mux.HandleFunc("GET /{$}", redirectHandler)
	mux.HandleFunc("/{$}", methodNotAllowedHandler)
	mux.HandleFunc("GET /static/style.css", styleHandler)
	mux.HandleFunc("/static/style.css", methodNotAllowedHandler)
	mux.HandleFunc("GET /memos", h.getMemosHandler)
	mux.HandleFunc("/memos", methodNotAllowedHandler)
	mux.HandleFunc("GET /memos/{id}", h.getMemoHandler)
	mux.HandleFunc("/memos/{id}", methodNotAllowedHandler)

	return mux
}

func getTemplate(pageName string) *template.Template {
	return template.Must(template.ParseFS(assets.FS, "template/layout.html", "template/"+pageName+".html"))
}

type handler struct {
	errorTemplate *template.Template
	memoList      *template.Template
	memoDetail    *template.Template
	memos         MemoReader
	errorLog      io.Writer
}

type errorData struct {
	Title string
}

func (h *handler) notFoundHandler(w http.ResponseWriter, r *http.Request) {
	h.executeErrorTemplate(w, http.StatusNotFound, "ページが見つかりません")
}

func redirectHandler(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, "/memos", http.StatusFound)
}

func methodNotAllowedHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Allow", "GET, HEAD")
	http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
}

func styleHandler(w http.ResponseWriter, r *http.Request) {
	http.ServeFileFS(w, r, assets.FS, "static/style.css")
}

func (h *handler) getMemosHandler(w http.ResponseWriter, r *http.Request) {
	memos, err := h.memos.ListMemos()
	if err != nil {
		errorTitle := "メモを読み込めませんでした"
		h.printError(errorTitle, err)
		h.executeErrorTemplate(w, http.StatusInternalServerError, errorTitle)
		return
	}

	h.executeOkTemplate(w, h.memoList, memos)
}

func (h *handler) getMemoHandler(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(r.PathValue("id"))
	if err != nil || id <= 0 {
		h.executeNotFoundMemo(w)
		return
	}
	m, err := h.memos.GetMemo(id)
	if err != nil {
		if errors.Is(err, memo.ErrNotFound) {
			h.executeNotFoundMemo(w)
		} else {
			errMsg := "メモを読み込めませんでした"
			h.printError(errMsg, err)
			h.executeErrorTemplate(w, http.StatusInternalServerError, errMsg)
		}
		return
	}

	h.executeOkTemplate(w, h.memoDetail, m)
}

func (h *handler) executeNotFoundMemo(w http.ResponseWriter) {
	h.executeErrorTemplate(w, http.StatusNotFound, "メモが見つかりません")
}

func (h *handler) executeTemplate(w http.ResponseWriter, statusCode int, t *template.Template, data any) {
	w.Header().Set("Content-type", "text/html; charset=utf-8")
	w.WriteHeader(statusCode)
	if err := t.ExecuteTemplate(w, "layout", data); err != nil {
		errMsg := "テンプレートの展開に失敗しました"
		h.printError(errMsg, err)
	}
}

func (h *handler) executeErrorTemplate(w http.ResponseWriter, statusCode int, errorTitle string) {
	h.executeTemplate(w, statusCode, h.errorTemplate, errorData{Title: errorTitle})
}

func (h *handler) executeOkTemplate(w http.ResponseWriter, t *template.Template, data any) {
	h.executeTemplate(w, http.StatusOK, t, data)
}

func (h *handler) printError(errMsg string, err error) {
	fmt.Fprintf(h.errorLog, "%s: %q\n", errMsg, err)
}
