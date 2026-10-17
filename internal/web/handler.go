package web

import (
	"html/template"
	"net/http"

	"github.com/co191194/memo-app/internal/memo"
	assets "github.com/co191194/memo-app/web"
)

type MemoReader interface {
	ListMemos() ([]memo.Memo, error)
	GetMemo(id int) (memo.Memo, error)
}

func NewHandler(memos MemoReader) http.Handler {
	mux := http.NewServeMux()

	h := handler{
		notFound: template.Must(template.ParseFS(assets.FS, "template/layout.html", "template/not-found.html")),
	}

	mux.HandleFunc("/", h.notFoundHandler)
	mux.HandleFunc("GET /{$}", redirectHandler)
	mux.HandleFunc("/{$}", methodNotAllowedHandler)
	mux.HandleFunc("GET /static/style.css", styleHandler)
	mux.HandleFunc("/static/style.css", methodNotAllowedHandler)

	return mux
}

type handler struct {
	notFound *template.Template
}

func (h *handler) notFoundHandler(w http.ResponseWriter, r *http.Request) {

	w.Header().Set("Content-type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusNotFound)
	h.notFound.ExecuteTemplate(w, "layout", nil)
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
