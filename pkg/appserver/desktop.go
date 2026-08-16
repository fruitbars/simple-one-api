package appserver

import (
	"net/http"
	"strings"

	"simple-one-api/internal/webui"
)

// DesktopAssetMiddleware sends API calls to the same Gin router used by the
// server while leaving Web assets to Wails. No loopback port is opened.
func DesktopAssetMiddleware(next http.Handler) http.Handler {
	api := NewRouterWithOptions(Options{EnableWeb: false, TrustedLocalAdminBootstrap: true})
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		webui.ApplySecurityHeaders(writer.Header())
		if isDesktopAPIPath(request.URL.Path) {
			api.ServeHTTP(writer, request)
			return
		}
		next.ServeHTTP(writer, request)
	})
}

func isDesktopAPIPath(requestPath string) bool {
	return strings.HasPrefix(requestPath, "/v1/") ||
		strings.HasPrefix(requestPath, "/api/") ||
		requestPath == "/translate" ||
		strings.HasPrefix(requestPath, "/v2/")
}
