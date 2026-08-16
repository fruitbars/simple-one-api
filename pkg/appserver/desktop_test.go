package appserver

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"simple-one-api/pkg/config"
)

func TestDesktopMiddlewareSeparatesAssetsAndAPI(t *testing.T) {
	nextCalled := false
	next := http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		nextCalled = true
		writer.WriteHeader(http.StatusTeapot)
	})
	handler := DesktopAssetMiddleware(next)

	assetResponse := httptest.NewRecorder()
	handler.ServeHTTP(assetResponse, httptest.NewRequest(http.MethodGet, "/assets/app.js", nil))
	if !nextCalled || assetResponse.Code != http.StatusTeapot {
		t.Fatal("asset request did not continue to the Wails asset handler")
	}
	if assetResponse.Header().Get("Content-Security-Policy") == "" {
		t.Fatal("desktop assets are missing the browser security policy")
	}

	nextCalled = false
	apiResponse := httptest.NewRecorder()
	handler.ServeHTTP(apiResponse, httptest.NewRequest(http.MethodPost, "/v1/not-found", nil))
	if nextCalled {
		t.Fatal("API request escaped to the Wails asset handler")
	}
	if apiResponse.Code != http.StatusNotFound {
		t.Fatalf("unexpected API status: %d", apiResponse.Code)
	}
}

func TestDesktopMiddlewareAllowsFirstRunAdminWithoutBootstrapToken(t *testing.T) {
	setTestConfiguration(t, config.Configuration{})
	next := http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		t.Fatal("desktop admin API request escaped to the Wails asset handler")
	})
	handler := DesktopAssetMiddleware(next)

	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/admin/status", nil))
	if response.Code != http.StatusOK {
		t.Fatalf("expected desktop first-run admin to open without bootstrap token, got %d: %s", response.Code, response.Body.String())
	}
}
