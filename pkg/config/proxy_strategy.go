package config

import (
	"context"
	"errors"
	"fmt"
	"golang.org/x/net/proxy"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// 定义代理类型常量
const (
	ProxyTypeHTTP   = "http"
	ProxyTypeSOCKS5 = "socks5"
)

// GetHttpProxyTransport 返回配置了 HTTP 代理的 http.Transport
func getHttpProxyTransport(httpProxyURL, httpsProxyURL string, timeout int) (*http.Transport, error) {
	parsedHTTPProxy, err := parseOptionalProxyURL(httpProxyURL)
	if err != nil {
		return nil, err
	}
	parsedHTTPSProxy, err := parseOptionalProxyURL(httpsProxyURL)
	if err != nil {
		return nil, err
	}
	if parsedHTTPProxy == nil && parsedHTTPSProxy == nil {
		return nil, errors.New("HTTP proxy address is empty")
	}

	transport := &http.Transport{
		Proxy: func(request *http.Request) (*url.URL, error) {
			if request.URL.Scheme == "https" && parsedHTTPSProxy != nil {
				return parsedHTTPSProxy, nil
			}
			if parsedHTTPProxy != nil {
				return parsedHTTPProxy, nil
			}
			return parsedHTTPSProxy, nil
		},
		DialContext: (&net.Dialer{
			Timeout:   time.Duration(timeout) * time.Second,
			KeepAlive: time.Duration(timeout) * time.Second,
		}).DialContext,
	}

	return transport, nil
}

func parseOptionalProxyURL(value string) (*url.URL, error) {
	if strings.TrimSpace(value) == "" {
		return nil, nil
	}
	parsed, err := url.Parse(value)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return nil, fmt.Errorf("error parsing proxy URL %s", value)
	}
	return parsed, nil
}

// GetSocks5Transport 返回配置了 SOCKS5 代理的 http.Transport，并设置超时时间
func getSocks5Transport(proxyAddr string, timeout int) (*http.Transport, error) {
	dialer, err := proxy.SOCKS5("tcp", proxyAddr, nil, proxy.Direct)
	if err != nil {
		return nil, fmt.Errorf("error creating SOCKS5 proxy at %s: %v", proxyAddr, err)
	}

	dialContext := func(ctx context.Context, network, addr string) (net.Conn, error) {
		conn, err := dialer.Dial(network, addr)
		if err != nil {
			return nil, err
		}
		return conn, nil
	}

	transport := &http.Transport{
		DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
			ctx, cancel := context.WithTimeout(ctx, time.Duration(timeout)*time.Second)
			defer cancel()
			return dialContext(ctx, network, addr)
		},
	}

	return transport, nil
}

// GetTypeProxyTransport 根据代理类型返回相应的 http.Transport
func GetTypeProxyTransport(proxyType, proxyAddr string, timeout int) (*http.Transport, error) {
	switch proxyType {
	case ProxyTypeHTTP:
		return getHttpProxyTransport(proxyAddr, "", timeout)
	case ProxyTypeSOCKS5:
		return getSocks5Transport(proxyAddr, timeout)
	default:
		return nil, errors.New("unsupported proxy type: " + proxyType)
	}
}

// GetConfProxyTransport 根据全局配置返回相应的 http.Transport
func GetConfProxyTransport() (string, string, *http.Transport, error) {
	proxyConfig := currentSnapshot().proxy
	proxyType := strings.ToLower(proxyConfig.Type)
	var proxyAddr string
	var transport *http.Transport
	var err error

	timeout := proxyConfig.Timeout
	if timeout <= 0 {
		timeout = 30
	}

	switch proxyType {
	case ProxyTypeHTTP:
		proxyAddr = proxyConfig.HTTPProxy
		if proxyAddr == "" {
			proxyAddr = proxyConfig.HTTPSProxy
		}
		transport, err = getHttpProxyTransport(proxyConfig.HTTPProxy, proxyConfig.HTTPSProxy, timeout)
	case ProxyTypeSOCKS5:
		if len(proxyConfig.Socks5Proxy) >= 7 && proxyConfig.Socks5Proxy[:7] == "socks5:" {
			proxyURL, err := url.Parse(proxyConfig.Socks5Proxy)
			if err != nil {
				return "", "", nil, fmt.Errorf("error parsing proxy URL: %w", err)
			}
			proxyAddr = proxyURL.Host
		} else {
			proxyAddr = proxyConfig.Socks5Proxy
		}

		transport, err = getSocks5Transport(proxyAddr, timeout)
	default:
		return "", "", nil, errors.New("unsupported proxy type: " + proxyType)
	}

	return proxyType, proxyAddr, transport, err
}
