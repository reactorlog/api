package api

import (
	"bytes"
	"io"
	"io/fs"
	"net/http"
	"strings"
)

// Public serves the JSON API under /api (prefix stripped) and optional static SPA files.
// When Static is nil, only /api/* is served and other paths 404.
type Public struct {
	API    http.Handler
	Static fs.FS
}

func (s Public) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path
	if path == "/api" || strings.HasPrefix(path, "/api/") {
		r2 := r.Clone(r.Context())
		r2.URL.Path = strings.TrimPrefix(path, "/api")
		if r2.URL.Path == "" {
			r2.URL.Path = "/"
		}
		s.API.ServeHTTP(w, r2)
		return
	}
	if path == "/live" && r.Method == http.MethodGet {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok\n"))
		return
	}
	if s.Static == nil {
		http.NotFound(w, r)
		return
	}
	serveSPA(w, r, s.Static)
}

func serveSPA(w http.ResponseWriter, r *http.Request, root fs.FS) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
		return
	}
	name := strings.TrimPrefix(r.URL.Path, "/")
	if name == "" || strings.HasSuffix(name, "/") {
		name = "index.html"
	}
	if serveFile(w, r, root, name) {
		return
	}
	// Client routes such as /sites/:id fall through to index.html.
	_ = serveFile(w, r, root, "index.html")
}

func serveFile(w http.ResponseWriter, r *http.Request, root fs.FS, name string) bool {
	f, err := root.Open(name)
	if err != nil {
		return false
	}
	defer f.Close()
	info, err := f.Stat()
	if err != nil || info.IsDir() {
		return false
	}
	rs, ok := f.(io.ReadSeeker)
	if !ok {
		data, err := io.ReadAll(f)
		if err != nil {
			return false
		}
		http.ServeContent(w, r, info.Name(), info.ModTime(), bytes.NewReader(data))
		return true
	}
	http.ServeContent(w, r, info.Name(), info.ModTime(), rs)
	return true
}
