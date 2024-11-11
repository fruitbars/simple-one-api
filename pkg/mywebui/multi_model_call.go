package mywebui

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/sashabaranov/go-openai"
	"go.uber.org/zap"
	"io"
	"net/http"
	"simple-one-api/pkg/mylog"
	"simple-one-api/pkg/simple_client"
	"sync"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	// 注意：生产环境下应更严格地检查来源
	CheckOrigin: func(r *http.Request) bool {
		return true // 允许所有CORS请求
	},
}

type MMFormData struct {
	Prompt      string   `json:"prompt"`
	Temperature float32  `json:"temperature"`
	MaxTokens   int      `json:"maxTokens"`
	TopP        float32  `json:"topP"`
	Models      []string `json:"models"`
	System      string   `json:"system"`
	MsgId       string   `json:"msgid"`
}

type MMResp struct {
	Model  string `json:"model"`
	Result string `json:"result"`
	MsgId  string `json:"msgid"`
}

func WSMultiModelCallHandler(c *gin.Context) {
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		mylog.Logger.Error("Upgrade failed", zap.Error(err))
		return
	}
	defer conn.Close()

	requestData, err := readAndUnmarshalClientMessage(conn)
	if err != nil {
		mylog.Logger.Error("Failed to read and unmarshal message", zap.Error(err))
		return
	}

	mylog.Logger.Info("WSMultiModelCallHandler|readAndUnmarshalClientMessage", zap.Any("requestData", requestData))

	baseRequest := constructBaseRequest(requestData)

	mylog.Logger.Info("WSMultiModelCallHandler|constructBaseRequest", zap.Any("baseRequest", baseRequest))

	var wg sync.WaitGroup
	var mu sync.Mutex

	msgId := uuid.New().String()

	for _, modelName := range requestData.Models {
		wg.Add(1)
		go handleModelRequest(&wg, &mu, conn, modelName, baseRequest, msgId)
	}

	wg.Wait()
}

func readAndUnmarshalClientMessage(conn *websocket.Conn) (*MMFormData, error) {
	_, message, err := conn.ReadMessage()
	if err != nil {
		return nil, err
	}
	mylog.Logger.Debug("Received message from client", zap.String("message", string(message)))

	var requestData MMFormData
	if err := json.Unmarshal(message, &requestData); err != nil {
		return nil, err
	}
	mylog.Logger.Debug("Received models", zap.Any("models", requestData.Models))

	return &requestData, nil
}

func constructBaseRequest(requestData *MMFormData) openai.ChatCompletionRequest {
	baseRequest := openai.ChatCompletionRequest{
		Stream: true,
		Messages: []openai.ChatCompletionMessage{
			{
				Role:    openai.ChatMessageRoleUser,
				Content: requestData.Prompt,
			},
		},
		MaxTokens:   requestData.MaxTokens,
		Temperature: requestData.Temperature,
		TopP:        requestData.TopP,
	}

	if requestData.System != "" {
		sysMsg := openai.ChatCompletionMessage{Role: openai.ChatMessageRoleSystem, Content: requestData.System}
		baseRequest.Messages = append([]openai.ChatCompletionMessage{sysMsg}, baseRequest.Messages...)
	}
	return baseRequest
}

func handleModelRequest(wg *sync.WaitGroup, mu *sync.Mutex, conn *websocket.Conn, modelName string, baseRequest openai.ChatCompletionRequest, msgId string) {
	defer wg.Done()

	modelReq := baseRequest
	modelReq.Model = modelName

	client := simple_client.NewSimpleClient("")
	chatStream, err := client.CreateChatCompletionStream(context.Background(), modelReq)
	if err != nil {
		mylog.Logger.Error("Failed to create chat completion stream", zap.Error(err))
		return
	}

	processChatStream(conn, chatStream, msgId, mu, modelName)
}

func processChatStream(conn *websocket.Conn, chatStream *simple_client.SimpleChatCompletionStream, msgId string, mu *sync.Mutex, modelName string) {
	for {
		chatResp, err := chatStream.Recv()
		if errors.Is(err, io.EOF) {
			mylog.Logger.Info("Stream finished")
			return
		}
		if err != nil {
			mylog.Logger.Error("Stream error", zap.Error(err))
			errResp := MMResp{
				Result: err.Error(),
				MsgId:  msgId,
				Model:  modelName,
			}

			mylog.Logger.Error("", zap.Any("errResp", errResp))

			mu.Lock()
			if err := conn.WriteJSON(errResp); err != nil {
				mylog.Logger.Error("Failed to write JSON response", zap.Error(err))
				mu.Unlock()
				break
			}
			mu.Unlock()

			return
		}

		if chatResp == nil {
			continue
		}

		mylog.Logger.Debug("Received chat response", zap.Any("chatResp", chatResp), zap.Int("len(chatResp.Choices)", len(chatResp.Choices)))
		if len(chatResp.Choices) > 0 {

			resp := MMResp{
				Result: chatResp.Choices[0].Delta.Content,
				MsgId:  msgId,
				Model:  modelName,
			}

			mu.Lock()
			if err := conn.WriteJSON(resp); err != nil {
				mylog.Logger.Error("Failed to write JSON response", zap.Error(err))
				mu.Unlock()
				break
			}
			mu.Unlock()
		}
	}
}
