package webui

import (
	"io/fs"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestAssetsContainProductionIndex(t *testing.T) {
	data, err := fs.ReadFile(Assets(), "index.html")
	if err != nil {
		t.Fatalf("read embedded index: %v", err)
	}
	if !strings.Contains(string(data), `<div id="root"></div>`) {
		t.Fatal("embedded index does not contain the React root")
	}
}

func TestHandlerFallsBackToSPAIndex(t *testing.T) {
	request := httptest.NewRequest(http.MethodGet, "/settings", nil)
	response := httptest.NewRecorder()

	Handler().ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("unexpected status: %d", response.Code)
	}
	if !strings.Contains(response.Body.String(), `<div id="root"></div>`) {
		t.Fatal("SPA fallback did not serve the new index.html")
	}
	if response.Header().Get("Content-Security-Policy") == "" {
		t.Fatal("missing content security policy")
	}
	if !strings.Contains(response.Header().Get("Content-Security-Policy"), "style-src 'self' 'unsafe-inline'") {
		t.Fatal("content security policy blocks runtime editor styles")
	}
}
