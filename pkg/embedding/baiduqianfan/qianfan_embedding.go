package baiduqianfan

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/baidubce/bce-qianfan-sdk/go/qianfan"
	"github.com/sashabaranov/go-openai"
	"io"
	"net/http"
	"simple-one-api/pkg/embedding/oai"
	baidu_qianfan "simple-one-api/pkg/llm/baidu-qianfan"
	"time"
)

func convertOpenAIEmbeddingRequestToBaiduEmbeddingRequest(src *oai.EmbeddingRequest) *qianfan.EmbeddingRequest {
	var inputs []string
	switch v := src.Input.(type) {
	case string:
		inputs = []string{v}
	case []string:
		inputs = v
	case []any:
		for _, item := range v {
			if str, ok := item.(string); ok {
				inputs = append(inputs, str)
			}
		}
	default:
		fmt.Println("Unsupported input type")
		return nil
	}

	return &qianfan.EmbeddingRequest{
		Input:  inputs,
		UserID: src.User,
	}
}

func convertBaiduEmbeddingResponseToOpenAIEmbeddingResponse(src *qianfan.EmbeddingResponse) *oai.EmbeddingResponse {
	var data []openai.Embedding
	for _, d := range src.Data {
		// 将浮点数从 float64 转为 float32
		embedding := make([]float32, len(d.Embedding))
		for i, val := range d.Embedding {
			embedding[i] = float32(val)
		}

		data = append(data, openai.Embedding{
			Object:    d.Object,
			Embedding: embedding,
			Index:     d.Index,
		})
	}

	return &oai.EmbeddingResponse{
		Object: src.Object,
		Data:   data,
		//Model:  src.Id,
		Usage: openai.Usage{
			PromptTokens: src.Usage.PromptTokens,
			TotalTokens:  src.Usage.TotalTokens,
		},
	}
}

func getBaiduEmbeddings(request *oai.EmbeddingRequest, accessToken string, proxyTransport *http.Transport) (*oai.EmbeddingResponse, error) {
	requestURL := fmt.Sprintf("https://aip.baidubce.com/rpc/2.0/ai_custom/v1/wenxinworkshop/embeddings/embedding-v1?access_token=%s", accessToken)

	bdReq := convertOpenAIEmbeddingRequestToBaiduEmbeddingRequest(request)

	jsonData, err := json.Marshal(bdReq)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", requestURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")

	var client *http.Client
	if proxyTransport != nil {
		client = &http.Client{
			Timeout:   60 * time.Second,
			Transport: proxyTransport,
		}
	} else {
		client = &http.Client{
			Timeout:   60 * time.Second,
			Transport: http.DefaultTransport, // 使用默认的 Transport
		}
	}
	res, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}

	var embeddingRes qianfan.EmbeddingResponse
	if err := json.Unmarshal(body, &embeddingRes); err != nil {
		return nil, err
	}

	return convertBaiduEmbeddingResponseToOpenAIEmbeddingResponse(&embeddingRes), nil
}
func BaiduQianfanEmbedding(req *oai.EmbeddingRequest, accessKey string, secretKey string, proxyTransport *http.Transport) (*oai.EmbeddingResponse, error) {

	accessToken := baidu_qianfan.GetAccessToken(accessKey, secretKey)

	return getBaiduEmbeddings(req, accessToken, proxyTransport)
}
