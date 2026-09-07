package main

import (
	"net/http"
	"strings"
	"testing"
)

func TestDesktopGatewayAddressUsesLoopback(t *testing.T) {
	cases := map[string]string{
		":9090":           "127.0.0.1:9090",
		"9091":            "127.0.0.1:9091",
		"0.0.0.0:9092":    "127.0.0.1:9092",
		"[::]:9093":       "127.0.0.1:9093",
		"invalid-address": "127.0.0.1:9090",
		"":                "127.0.0.1:9090",
	}
	for input, want := range cases {
		if got := desktopGatewayAddress(input); got != want {
			t.Errorf("desktopGatewayAddress(%q) = %q, want %q", input, got, want)
		}
	}
}

func TestDesktopGatewayServesAndStops(t *testing.T) {
	server, listener, err := startDesktopGateway(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("ok"))
	}), ":0")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(listener.Addr().String(), "127.0.0.1:") {
		t.Fatalf("listener address = %q, want loopback", listener.Addr())
	}
	if err := stopDesktopGateway(server); err != nil {
		t.Fatal(err)
	}
}
