package baidu_agentbuilder

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/google/uuid"
	"github.com/patrickmn/go-cache"
	"go.uber.org/zap"
	"io"
	"net/http"
	"net/url"
	"simple-one-api/pkg/mylog"
	"strings"
	"time"
)

// 定义一个全局缓存对象
var tokenCache = cache.New(5*time.Minute, 10*time.Minute)

// AccessToken 定义 access_token 的结构
type AccessToken struct {
	Token     string    `json:"access_token"`
	ExpiresIn int       `json:"expires_in"`
	ExpiresAt time.Time `json:"expires_at"`
}

// getAccessToken 获取 access_token，如果缓存中存在且未过期，则使用缓存中的 token，否则重新获取
func getAccessToken(clientID, clientSecret string) (string, error) {
	// 尝试从缓存中获取 access_token
	if token, found := tokenCache.Get("access_token"); found {
		fmt.Println("Using cached access token.")
		return token.(string), nil
	}

	// 如果缓存中没有，或者过期，重新获取
	newToken, err := fetchAccessTokenFromAPI(clientID, clientSecret)
	if err != nil {
		return "", err
	}

	mylog.Logger.Info("getAccessToken|fetchAccessTokenFromAPI", zap.Int("newToken.ExpiresIn", newToken.ExpiresIn))
	if newToken.ExpiresIn > 0 {
		// 将新的 token 存入缓存，设置缓存时间为 token 的有效期
		tokenCache.Set("access_token", newToken.Token, time.Until(newToken.ExpiresAt))
	}

	return newToken.Token, nil
}

// fetchAccessTokenFromAPI 通过 API 获取 access_token
func fetchAccessTokenFromAPI(clientID, clientSecret string) (*AccessToken, error) {
	baseURL := "https://openapi.baidu.com/oauth/2.0/token"
	params := url.Values{}
	params.Add("grant_type", "client_credentials")
	params.Add("client_id", clientID)
	params.Add("client_secret", clientSecret)

	requestURL := fmt.Sprintf("%s?%s", baseURL, params.Encode())

	resp, err := http.Get(requestURL)
	if err != nil {
		return nil, fmt.Errorf("failed to request access token: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %v", err)
	}

	var result map[string]interface{}
	err = json.Unmarshal(body, &result)
	if err != nil {
		return nil, fmt.Errorf("failed to parse JSON: %v", err)
	}

	if errVal, ok := result["error"]; ok {
		return nil, fmt.Errorf("error: %s, description: %s", errVal, result["error_description"])
	}

	token, ok := result["access_token"].(string)
	if !ok {
		// 处理错误或采取其他措施
		return nil, errors.New("no access_token," + string(body))
	}
	expiresIn := result["expires_in"].(int)
	if !ok {
		// 处理错误或采取其他措施
		expiresIn = 0
	}

	// 计算过期时间
	expiresAt := time.Now().Add(time.Duration(expiresIn) * time.Second)

	return &AccessToken{
		Token:     token,
		ExpiresIn: expiresIn,
		ExpiresAt: expiresAt,
	}, nil
}

// getAnswer 函数实现
func GetAnswer(agentID, secretKey, question string) (*GetAnswerResponse, error) {
	url := fmt.Sprintf("https://agentapi.baidu.com/assistant/getAnswer?appId=%s&secretKey=%s", agentID, secretKey)

	// 构建请求内容
	requestBody := GetAnswerRequest{
		Message: GetAnswerMessage{
			Content: GetAnswerMessageContent{
				Type: "text",
				Value: map[string]string{
					"showText": question,
				},
			},
		},
		Source: agentID,
		From:   "openapi",
		OpenID: agentID,
	}

	// 将请求内容编码为 JSON
	jsonData, err := json.Marshal(requestBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request body: %v", err)
	}

	// 创建 POST 请求
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")

	// 发送请求
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %v", err)
	}
	defer resp.Body.Close()

	// 读取响应
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %v", err)
	}

	// 解析响应 JSON
	var response GetAnswerResponse
	err = json.Unmarshal(body, &response)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %v", err)
	}

	// 检查请求是否成功
	if response.Status != 0 {
		return &response, fmt.Errorf("API error: %s", response.Message)
	}

	return &response, nil
}

// conversation 函数实现
func Conversation(agentID, secretKey, question string, callBack func(data string)) error {
	url := fmt.Sprintf("https://agentapi.baidu.com/assistant/conversation?appId=%s&secretKey=%s", agentID, secretKey)

	requestBody := ConversationRequest{
		Message: ConversationMessage{
			Content: ConversationMessageContent{
				Type: "text",
				Value: map[string]interface{}{
					"showText": question,
				},
			},
		},
		Source: agentID,
		From:   "openapi",
		OpenID: uuid.New().String(),
	}

	jsonData, err := json.Marshal(requestBody)
	if err != nil {
		return fmt.Errorf("failed to marshal request body: %v", err)
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %v", err)
	}
	defer resp.Body.Close()

	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		line := scanner.Text()
		if len(line) > 0 {
			if strings.HasPrefix(line, "data:") {
				callBack(line[5:])
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("error reading stream: %v", err)
	}

	return nil
}
