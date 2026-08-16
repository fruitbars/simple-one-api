package webui

import (
	"embed"
	"io/fs"
	"net/http"
	"path"
	"strings"
)

//go:embed dist
var bundled embed.FS

var assets = mustSub(bundled, "dist")

func mustSub(source fs.FS, directory string) fs.FS {
	result, err := fs.Sub(source, directory)
	if err != nil {
		panic(err)
	}
	return result
}

// Assets is shared by the HTTP server and the Wails desktop shell.
func Assets() fs.FS {
	return assets
}

// Handler serves immutable build assets and falls back to index.html for SPA
// routes. The application router keeps API paths away from this handler.
func Handler() http.Handler {
	files := http.FileServer(http.FS(assets))
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		ApplySecurityHeaders(writer.Header())

		requestedPath := strings.TrimPrefix(path.Clean(request.URL.Path), "/")
		if requestedPath == "." || requestedPath == "" {
			requestedPath = "index.html"
		}
		entry, err := fs.Stat(assets, requestedPath)
		if err != nil || entry.IsDir() {
			clone := request.Clone(request.Context())
			clone.URL.Path = "/"
			files.ServeHTTP(writer, clone)
			return
		}
		files.ServeHTTP(writer, request)
	})
}

// ApplySecurityHeaders keeps the server and desktop WebView on the same
// restrictive browser policy.
func ApplySecurityHeaders(header http.Header) {
	header.Set("Content-Security-Policy", "default-src 'self'; connect-src 'self' http://127.0.0.1:*; img-src 'self' data:; style-src 'self' 'unsafe-inline'; script-src 'self'; object-src 'none'; base-uri 'none'; frame-ancestors 'none'")
	header.Set("Referrer-Policy", "no-referrer")
	header.Set("X-Content-Type-Options", "nosniff")
	header.Set("X-Frame-Options", "DENY")
}
