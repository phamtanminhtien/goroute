package httpapi

import (
	"bytes"
	"io/fs"
	"net/http"
	"path"
	"path/filepath"
	"strings"
)

func webUIHandler(root fs.FS) http.Handler {
	fileServer := http.FileServer(http.FS(root))

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			http.NotFound(w, r)
			return
		}

		requestPath := normalizeWebUIPath(r.URL.Path)
		if isReservedBackendPath(requestPath) {
			http.NotFound(w, r)
			return
		}
		if requestPath == "/" {
			serveWebUIFile(w, r, root, "index.html")
			return
		}

		if webUIFileExists(root, requestPath) {
			fileServer.ServeHTTP(w, cloneRequestWithPath(r, requestPath))
			return
		}

		if filepath.Ext(requestPath) != "" {
			http.NotFound(w, r)
			return
		}

		serveWebUIFile(w, r, root, "index.html")
	})
}

func normalizeWebUIPath(requestPath string) string {
	cleaned := path.Clean("/" + strings.TrimSpace(requestPath))
	if cleaned == "." {
		return "/"
	}

	return cleaned
}

func isReservedBackendPath(requestPath string) bool {
	return requestPath == "/healthz" ||
		requestPath == "/v1" ||
		strings.HasPrefix(requestPath, "/v1/") ||
		requestPath == "/admin" ||
		strings.HasPrefix(requestPath, "/admin/")
}

func webUIFileExists(root fs.FS, requestPath string) bool {
	filePath := strings.TrimPrefix(requestPath, "/")
	if filePath == "" {
		filePath = "index.html"
	}

	info, err := fs.Stat(root, filePath)
	if err != nil {
		return false
	}

	return !info.IsDir()
}

func cloneRequestWithPath(r *http.Request, requestPath string) *http.Request {
	clone := r.Clone(r.Context())
	clone.URL.Path = requestPath
	clone.URL.RawPath = requestPath
	return clone
}

func serveWebUIFile(w http.ResponseWriter, r *http.Request, root fs.FS, filePath string) {
	fileBytes, err := fs.ReadFile(root, filePath)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	info, err := fs.Stat(root, filePath)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	http.ServeContent(w, r, info.Name(), info.ModTime(), bytes.NewReader(fileBytes))
}
