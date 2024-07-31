package translation

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/sashabaranov/go-openai"
	"go.uber.org/zap"
	"io"
	"simple-one-api/pkg/config"
	"simple-one-api/pkg/mylog"
	"simple-one-api/pkg/simple_client"
)

func createLLMTranslationPromptJson(srcText string, srcLang string, targetLang string) string {
	transReq := TranslationV1Request{
		Text:       srcText,
		SourceLang: srcLang,
		TargetLang: targetLang,
	}
	reqJsonstr, _ := json.Marshal(transReq)
	return fmt.Sprintf("你是一个机器翻译接口，遵循以下输入输出协议，当接收到输入，直接给出输出即可，不要任何多余的回复\n输入协议(json格式)：\n```\n{\"text\":\"Hello world!\",\"target_lang\":\"DE\"}\n```\n\n翻译结果直接输出：\n\nHallo, Welt!\n\n现在我的输入是：\n```\n%s\n```\n输出：\n", reqJsonstr)

}

var defaultLLMTransPrompt = "你是一个机器翻译接口，遵循以下输入输出协议，当接收到输入，直接给出输出即可，不要任何多余的回复\n输入：\n```\n将以下文本翻译为目标语言：DE\n文本:\n\n\nHello world!\n```\n\n翻译结果直接输出：\n\nHallo, Welt!\n\n现在我的输入是：\n```\n将以下文本翻译为目标语言：%s\n文本:\n\n\n%s\n```\n输出："

func createLLMTranslationPrompt(srcText string, srcLang string, targetLang string) string {
	prompt := defaultLLMTransPrompt
	if config.GTranslation.PromptTemplate != "" {
		prompt = config.GTranslation.PromptTemplate
	}

	return fmt.Sprintf(prompt, targetLang, srcText)
}

func LLMTranslate(srcText string, srcLang string, targetLang string) (string, error) {

	prompt := createLLMTranslationPrompt(srcText, srcLang, targetLang)

	var req openai.ChatCompletionRequest
	req.Stream = false
	req.Model = "random"

	message := openai.ChatCompletionMessage{
		Role:    openai.ChatMessageRoleUser,
		Content: prompt,
	}

	req.Messages = append(req.Messages, message)

	client := simple_client.NewSimpleClient("")

	resp, err := client.CreateChatCompletion(context.Background(), req)
	if err != nil {
		mylog.Logger.Error("Error creating chat completion:", zap.Error(err))
		return "", err
	}

	if len(resp.Choices) > 0 {
		mylog.Logger.Info("Received chat response", zap.String("content", resp.Choices[0].Message.Content))

		return resp.Choices[0].Message.Content, nil
	}

	return "", errors.New("no result")
}

func LLMTranslateStream(srcText string, srcLang string, targetLang string, cb func(string)) (string, error) {
	var allResult string
	prompt := createLLMTranslationPrompt(srcText, srcLang, targetLang)

	var req openai.ChatCompletionRequest
	req.Stream = false
	req.Model = "random"

	message := openai.ChatCompletionMessage{
		Role:    openai.ChatMessageRoleUser,
		Content: prompt,
	}

	req.Messages = append(req.Messages, message)

	client := simple_client.NewSimpleClient("")

	chatStream, err := client.CreateChatCompletionStream(context.Background(), req)
	if err != nil {
		mylog.Logger.Error("Error creating chat completion:", zap.Error(err))
		return "", err
	}

	for {
		var chatResp *openai.ChatCompletionStreamResponse
		chatResp, err = chatStream.Recv()
		if errors.Is(err, io.EOF) {
			mylog.Logger.Debug("Stream finished")
			return allResult, nil
		}
		if err != nil {
			mylog.Logger.Error("Error receiving chat response:", zap.Error(err))
			break
		}

		if chatResp == nil {
			continue
		}

		mylog.Logger.Info("Received chat response", zap.Any("chatResp", chatResp))
		if len(chatResp.Choices) > 0 {
			cb(chatResp.Choices[0].Delta.Content)

			allResult += chatResp.Choices[0].Delta.Content
		}
	}

	return allResult, err
}
