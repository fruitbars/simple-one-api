package appserver

import (
	"net/http"
	"strings"

	"simple-one-api/internal/webui"
	"simple-one-api/pkg/config"
)

// NewDesktopRouter builds the shared router used by the Wails bridge and its
// loopback gateway. Web routes follow the persisted enable_web setting just as
// they do in the standalone server.
func NewDesktopRouter() http.Handler {
	return NewRouterWithOptions(Options{
		EnableWeb:                  config.CurrentConfiguration().EnableWeb,
		TrustedLocalAdminBootstrap: true,
	})
}

// DesktopAssetMiddleware sends API calls to the same Gin router used by the
// server while leaving Web assets to Wails.
func DesktopAssetMiddleware(next http.Handler) http.Handler {
	api := NewDesktopRouter()
	return DesktopAssetMiddlewareFor(api)(next)
}

// DesktopAssetMiddlewareFor routes API requests through a shared in-process
// handler so the Wails bridge and asset server use the same router.
func DesktopAssetMiddlewareFor(api http.Handler) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
			webui.ApplySecurityHeaders(writer.Header())
			if isDesktopAPIPath(request.URL.Path) {
				authorizeDesktopRequest(request)
				api.ServeHTTP(writer, request)
				return
			}
			next.ServeHTTP(writer, request)
		})
	}
}

func authorizeDesktopRequest(request *http.Request) {
	if strings.TrimSpace(request.Header.Get("Authorization")) != "" || strings.TrimSpace(request.Header.Get("x-api-key")) != "" {
		return
	}
	if key := strings.TrimSpace(config.CurrentAPIKey()); key != "" {
		request.Header.Set("Authorization", "Bearer "+key)
	}
}

func isDesktopAPIPath(requestPath string) bool {
	return strings.HasPrefix(requestPath, "/v1/") ||
		strings.HasPrefix(requestPath, "/api/") ||
		requestPath == "/translate" ||
		strings.HasPrefix(requestPath, "/v2/")
}
