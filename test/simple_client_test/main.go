package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"os"

	"github.com/sashabaranov/go-openai"
	"simple-one-api/pkg/initializer"
	"simple-one-api/pkg/simple_client"
)

func testStream(ctx context.Context) error {
	prompt := "你好，大模型"

	var req openai.ChatCompletionRequest
	req.Stream = true
	req.Model = "random"

	message := openai.ChatCompletionMessage{
		Role:    openai.ChatMessageRoleUser,
		Content: prompt,
	}

	req.Messages = append(req.Messages, message)

	client := simple_client.NewSimpleClient("")

	chatStream, err := client.CreateChatCompletionStream(ctx, req)
	if err != nil {
		fmt.Println(err)
		return err
	}

	for {
		chatResp, err := chatStream.Recv()
		if errors.Is(err, io.EOF) {
			fmt.Println("")
			return nil
		}
		if err != nil {
			fmt.Println(err)
			return err
		}

		if chatResp == nil {
			continue
		}

		fmt.Printf("%s", chatResp.Choices[0].Delta.Content)
	}
}

func testNonStream(ctx context.Context) error {
	prompt := "你好，大模型"

	var req openai.ChatCompletionRequest
	req.Stream = false
	req.Model = "random"

	message := openai.ChatCompletionMessage{
		Role:    openai.ChatMessageRoleUser,
		Content: prompt,
	}

	req.Messages = append(req.Messages, message)

	client := simple_client.NewSimpleClient("")

	resp, err := client.CreateChatCompletion(ctx, req)
	if err != nil {
		return err
	}

	if len(resp.Choices) > 0 {
		fmt.Println(resp.Choices[0].Message.Content)
	}
	return nil
}

func main() {
	configPath := "config.json"
	if len(os.Args) > 1 {
		configPath = os.Args[1]
	}
	if err := initializer.Setup(configPath); err != nil {
		log.Fatalf("load configuration %q: %v", configPath, err)
	}
	defer initializer.Cleanup()

	ctx := context.Background()
	fmt.Println("stream mode===========")
	if err := testStream(ctx); err != nil {
		log.Printf("stream request failed: %v", err)
	}
	fmt.Println("non-stream mode===========")
	if err := testNonStream(ctx); err != nil {
		log.Printf("non-stream request failed: %v", err)
	}
}
