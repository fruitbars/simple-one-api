package appserver

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"

	"github.com/gin-gonic/gin"
	"simple-one-api/pkg/config"
)

const adminBootstrapEnvironment = "SIMPLE_ONE_API_BOOTSTRAP_TOKEN"

var generatedAdminBootstrapToken = mustGenerateAdminBootstrapToken()
var adminBootstrapLogOnce sync.Once

type AdminAccessOptions struct {
	TrustLocalBootstrap bool
}

func requireAdminAccess(options ...AdminAccessOptions) gin.HandlerFunc {
	var option AdminAccessOptions
	if len(options) > 0 {
		option = options[0]
	}
	return func(context *gin.Context) {
		expected := strings.TrimSpace(config.CurrentAPIKey())
		if expected == "" {
			if !isSameOriginAdminRequest(context.Request) {
				context.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "cross-origin bootstrap admin access is forbidden"})
				return
			}
			if !option.TrustLocalBootstrap && !isLocalAdminRequest(context.Request) {
				provided := bearerToken(context.GetHeader("Authorization"))
				bootstrap := adminBootstrapToken()
				if provided == "" || subtle.ConstantTimeCompare([]byte(provided), []byte(bootstrap)) != 1 {
					context.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
						"code":  "admin_bootstrap_required",
						"error": "enter the temporary admin bootstrap token from the server startup log",
					})
					return
				}
			}
			context.Next()
			return
		}

		provided := bearerToken(context.GetHeader("Authorization"))
		if subtle.ConstantTimeCompare([]byte(provided), []byte(expected)) != 1 {
			context.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}
		context.Next()
	}
}

func adminBootstrapToken() string {
	if configured := strings.TrimSpace(os.Getenv(adminBootstrapEnvironment)); configured != "" {
		return configured
	}
	return generatedAdminBootstrapToken
}

func announceAdminBootstrap() {
	if strings.TrimSpace(config.CurrentAPIKey()) != "" {
		return
	}
	adminBootstrapLogOnce.Do(func() {
		if strings.TrimSpace(os.Getenv(adminBootstrapEnvironment)) != "" {
			log.Printf("Admin first-run bootstrap is enabled with %s", adminBootstrapEnvironment)
			return
		}
		log.Printf("Admin temporary bootstrap token: %s", generatedAdminBootstrapToken)
		log.Println("Open /admin, enter this token, then configure and publish a permanent api_key")
	})
}

func mustGenerateAdminBootstrapToken() string {
	buffer := make([]byte, 32)
	if _, err := rand.Read(buffer); err != nil {
		panic("generate admin bootstrap token: " + err.Error())
	}
	return base64.RawURLEncoding.EncodeToString(buffer)
}

func isSameOriginAdminRequest(request *http.Request) bool {
	origin := strings.TrimSpace(request.Header.Get("Origin"))
	if origin == "" {
		return true
	}
	parsed, err := url.Parse(origin)
	if err != nil || parsed.Host == "" {
		return false
	}
	return strings.EqualFold(parsed.Host, request.Host)
}

func isLocalAdminRequest(request *http.Request) bool {
	host := request.Host
	if parsedHost, _, err := net.SplitHostPort(host); err == nil {
		host = parsedHost
	}
	host = strings.Trim(host, "[]")
	hostIsLocal := strings.EqualFold(host, "localhost") || strings.HasSuffix(strings.ToLower(host), ".localhost")
	if ip := net.ParseIP(host); ip != nil && ip.IsLoopback() {
		hostIsLocal = true
	}
	if !hostIsLocal {
		return false
	}
	if request.RemoteAddr == "" {
		return true
	}
	remoteHost, _, err := net.SplitHostPort(request.RemoteAddr)
	if err != nil {
		remoteHost = request.RemoteAddr
	}
	remoteIP := net.ParseIP(strings.Trim(remoteHost, "[]"))
	return remoteIP != nil && remoteIP.IsLoopback()
}

func requireAPIAccess() gin.HandlerFunc {
	return func(context *gin.Context) {
		expected := strings.TrimSpace(config.CurrentAPIKey())
		if expected == "" {
			context.Next()
			return
		}
		provided := bearerToken(context.GetHeader("Authorization"))
		if provided == "" {
			provided = strings.TrimSpace(context.GetHeader("x-api-key"))
		}
		if subtle.ConstantTimeCompare([]byte(provided), []byte(expected)) != 1 {
			context.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}
		context.Next()
	}
}

func bearerToken(authorization string) string {
	scheme, token, ok := strings.Cut(strings.TrimSpace(authorization), " ")
	if !ok || !strings.EqualFold(scheme, "Bearer") {
		return ""
	}
	return strings.TrimSpace(token)
}
