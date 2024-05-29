package utils

import (
	"errors"
	"github.com/gin-gonic/gin"
	"net/http"
	"strings"
)

func SetEventStreamHeaders(c *gin.Context) {
	c.Writer.Header().Set("Content-Type", "text/event-stream")
	c.Writer.Header().Set("Cache-Control", "no-cache")
	c.Writer.Header().Set("Connection", "keep-alive")
	c.Writer.Header().Set("Transfer-Encoding", "chunked")
	c.Writer.Header().Set("X-Accel-Buffering", "no")
}

func SendOpenAIStreamEOFData(c *gin.Context) {
	c.Writer.WriteString("data: [DONE]\n\n")
	c.Writer.(http.Flusher).Flush()
}

// 从Authorization头部中获取API密钥
func GetAPIKeyFromHeader(c *gin.Context) (string, error) {
	authHeader := c.GetHeader("Authorization")
	if authHeader == "" {
		return "", errors.New("invalid authorization header format")
	}

	parts := strings.Split(authHeader, " ")
	if len(parts) != 2 || parts[0] != "Bearer" {
		return "", errors.New("authorization header not found")
	}
	return parts[1], nil
}
