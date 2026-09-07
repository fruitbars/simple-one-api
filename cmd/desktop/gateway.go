package main

import (
	"context"
	"errors"
	"log"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// desktopGatewayAddress keeps the desktop-only HTTP gateway on loopback even
// when the shared configuration uses a wildcard or host-bound address.
func desktopGatewayAddress(configured string) string {
	configured = strings.TrimSpace(configured)
	if configured == "" {
		return "127.0.0.1:9090"
	}
	if strings.HasPrefix(configured, ":") {
		return "127.0.0.1" + configured
	}
	if _, port, err := net.SplitHostPort(configured); err == nil {
		return net.JoinHostPort("127.0.0.1", port)
	}
	if port, err := strconv.Atoi(configured); err == nil && port > 0 && port < 65536 {
		return net.JoinHostPort("127.0.0.1", configured)
	}
	return "127.0.0.1:9090"
}

func startDesktopGateway(handler http.Handler, configuredAddress string) (*http.Server, net.Listener, error) {
	listener, err := net.Listen("tcp", desktopGatewayAddress(configuredAddress))
	if err != nil {
		return nil, nil, err
	}
	server := &http.Server{
		Handler:           handler,
		ReadHeaderTimeout: 10 * time.Second,
		IdleTimeout:       90 * time.Second,
	}
	go func() {
		if serveErr := server.Serve(listener); serveErr != nil && !errors.Is(serveErr, http.ErrServerClosed) {
			// The caller cannot recover a failed listener after startup; keep the
			// desktop UI available while exposing the cause in the app log.
			log.Printf("desktop gateway stopped unexpectedly: %v", serveErr)
		}
	}()
	return server, listener, nil
}

func stopDesktopGateway(server *http.Server) error {
	if server == nil {
		return nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	return server.Shutdown(ctx)
}
