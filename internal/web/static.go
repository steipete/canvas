package web

import (
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"
)

type StaticHandler struct {
	rootAbs string
}

func NewStaticHandler(root string) (*StaticHandler, error) {
	abs, err := filepath.Abs(root)
	if err != nil {
		return nil, err
	}
	return &StaticHandler{rootAbs: filepath.Clean(abs)}, nil
}

func (h *StaticHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	reqPath := r.URL.Path
	if reqPath == "" {
		reqPath = "/"
	}

	clean := path.Clean("/" + reqPath)
	target := filepath.Join(h.rootAbs, filepath.FromSlash(clean))
	target = filepath.Clean(target)

	if !withinRoot(h.rootAbs, target) {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}

	info, err := os.Stat(target)
	if err == nil && info.IsDir() {
		h.serveIndex(w, r, target)
		return
	}

	if err != nil {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	if info.IsDir() {
		h.serveIndex(w, r, target)
		return
	}

	http.ServeFile(w, r, target)
}

func (h *StaticHandler) serveIndex(w http.ResponseWriter, r *http.Request, dir string) {
	for _, name := range []string{"index.html", "index.htm"} {
		p := filepath.Join(dir, name)
		if fi, err := os.Stat(p); err == nil && !fi.IsDir() {
			http.ServeFile(w, r, p)
			return
		}
	}
	http.Error(w, "not found", http.StatusNotFound)
}

func withinRoot(rootAbs, targetAbs string) bool {
	if targetAbs == rootAbs {
		return true
	}
	prefix := rootAbs + string(os.PathSeparator)
	return strings.HasPrefix(targetAbs, prefix)
}
