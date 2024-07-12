package main

import (
	"context"
	"errors"
	"fmt"
	"github.com/sashabaranov/go-openai"
	"io"
	"log"
	"simple-one-api/pkg/initializer" // 引入initializer包
	"simple-one-api/pkg/simple_client"
)

func testStream() error {
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

	chatStream, err := client.CreateChatCompletionStream(context.Background(), req)
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

	fmt.Println("")

	return nil
}

func testNoneStream() {
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

	resp, err := client.CreateChatCompletion(context.Background(), req)
	if err != nil {
		fmt.Println(err)
		return
	}

	if len(resp.Choices) > 0 {
		fmt.Println(resp.Choices[0].Message.Content)
	}
}

func main() {
	if err := initializer.Setup("../../myconfigs/config.json"); err != nil {
		log.Println(err)
		return
	}
	defer initializer.Cleanup()

	fmt.Println("stream mode===========")
	testStream()
	fmt.Println("none stream mode===========")
	testNoneStream()
}
