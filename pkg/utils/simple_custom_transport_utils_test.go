package utils

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"testing"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (fn roundTripFunc) RoundTrip(request *http.Request) (*http.Response, error) { return fn(request) }

func TestSimpleCustomTransportMergesExtraJSONFields(t *testing.T) {
	base := roundTripFunc(func(request *http.Request) (*http.Response, error) {
		body, _ := io.ReadAll(request.Body)
		var payload map[string]any
		if err := json.Unmarshal(body, &payload); err != nil {
			t.Fatal(err)
		}
		if payload["enable_thinking"] != false {
			t.Fatalf("enable_thinking = %#v", payload["enable_thinking"])
		}
		if payload["model"] != "mapped-model" {
			t.Fatalf("model = %#v", payload["model"])
		}
		return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(bytes.NewReader(nil)), Header: make(http.Header)}, nil
	})
	transport := &SimpleCustomTransport{Transport: base, ExtraJSONFields: map[string]json.RawMessage{
		"enable_thinking": json.RawMessage(`false`),
		"model":           json.RawMessage(`"client-model"`),
	}}
	request, _ := http.NewRequest(http.MethodPost, "https://example.test/v1/chat/completions", bytes.NewBufferString(`{"model":"mapped-model"}`))
	request.Header.Set("Content-Type", "application/json")
	if _, err := transport.RoundTrip(request); err != nil {
		t.Fatal(err)
	}
}
