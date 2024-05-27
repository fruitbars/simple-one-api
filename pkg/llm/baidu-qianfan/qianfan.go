package baidu_qianfan

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
)

func QianFanCall(api_key, secret_key, model string, qfReq *QianFanRequest) (*QianFanResponse, error) {
	log.Println(api_key, secret_key, model, qfReq)
	accessToken := GetAccessToken(api_key, secret_key)
	if accessToken == "" {
		log.Println("Failed to get access token")
		return nil, nil
	}

	return SendChatRequest(accessToken, model, qfReq)
}

func QianFanCallSSE(api_key, secret_key, model string, qfReq *QianFanRequest, callback func(qfResp *QianFanResponse)) error {
	log.Println(api_key, secret_key, model, qfReq)
	accessToken := GetAccessToken(api_key, secret_key)
	if accessToken == "" {
		log.Println("Failed to get access token")
		return nil
	}

	return SendChatRequestWithSSE(accessToken, model, qfReq, callback)
}

// SendChatRequestWithSSE 发送 SSE 请求并处理响应
func SendChatRequestWithSSE(accessToken, model string, qfReq *QianFanRequest, callback func(qfResp *QianFanResponse)) error {
	address := qianfanModelName2Address(model)
	url := "https://aip.baidubce.com/rpc/2.0/ai_custom/v1/wenxinworkshop/chat/" + address + "?access_token=" + accessToken

	jsonData, err := json.Marshal(qfReq)
	if err != nil {
		log.Println(err)
		return err
	}

	log.Println(string(jsonData))

	client := &http.Client{}
	req, err := http.NewRequest("POST", url, bytes.NewReader(jsonData))
	if err != nil {
		log.Println(err)
		return err
	}
	req.Header.Add("Content-Type", "application/json")

	res, err := client.Do(req)
	if err != nil {
		log.Println(err)
		return err
	}

	log.Println(url, "request ok")
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		log.Println("received non-200 response code:", res.StatusCode)
		return fmt.Errorf("received non-200 response code: %d", res.StatusCode)
	}

	// 使用 bufio.Scanner 解析 SSE 响应
	scanner := bufio.NewScanner(res.Body)
	for scanner.Scan() {
		line := scanner.Text()
		//log.Println(line)
		if strings.HasPrefix(line, "data:") || len(line) > 0 {
			data := strings.TrimPrefix(line, "data:")

			log.Println(string(data))

			var response QianFanResponse
			err = json.Unmarshal([]byte(data), &response)
			if err != nil {
				log.Println(err)

			} else {
				log.Println("execute callback")
				callback(&response)
			}
		}
	}

	if err := scanner.Err(); err != nil {
		log.Println(err)
		return err
	}

	return nil
}

func SendChatRequest(accessToken, model string, qfReq *QianFanRequest) (*QianFanResponse, error) {
	address := qianfanModelName2Address(model)
	url := "https://aip.baidubce.com/rpc/2.0/ai_custom/v1/wenxinworkshop/chat/" + address + "?access_token=" + accessToken

	jsonData, err := json.Marshal(qfReq)
	if err != nil {
		log.Println(err)
		return nil, err
	}

	log.Println(string(jsonData))

	client := &http.Client{}
	req, err := http.NewRequest("POST", url, bytes.NewReader(jsonData))
	if err != nil {
		log.Println(err)
		return nil, err
	}
	req.Header.Add("Content-Type", "application/json")

	res, err := client.Do(req)
	if err != nil {
		log.Println(err)
		return nil, err
	}
	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)
	if err != nil {
		log.Println(err)
		return nil, err
	}

	log.Println(string(body))

	var response QianFanResponse
	err = json.Unmarshal(body, &response)
	if err != nil {
		log.Println(err)
		return nil, err
	}

	log.Println(response)

	return &response, nil
}

func qianfanModelName2Address(modelName string) string {
	address := strings.ToLower(modelName)
	switch modelName {
	case "ERNIE-Speed-8K":
		address = "ernie_speed"
	case "ERNIE-Lite-8K-0922":
		address = "eb-instant"
	case "Yi-34B-Chat":
		address = "yi_34b_chat"
	}

	return address
}

// GetAccessToken 使用 AK，SK 生成鉴权签名（Access Token）
func GetAccessToken(api_key, secret_key string) string {
	url := "https://aip.baidubce.com/oauth/2.0/token"
	postData := fmt.Sprintf("grant_type=client_credentials&client_id=%s&client_secret=%s", api_key, secret_key)
	resp, err := http.Post(url, "application/x-www-form-urlencoded", strings.NewReader(postData))
	if err != nil {
		log.Printf("Failed to send request for access token: %v", err)
		return ""
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("Failed to read response body: %v", err)
		return ""
	}

	//	log.Printf("AccessToken response body: %s", body)

	var accessTokenObj map[string]interface{}
	if err := json.Unmarshal(body, &accessTokenObj); err != nil {
		log.Printf("Failed to unmarshal response body: %v", err)
		return ""
	}

	if token, ok := accessTokenObj["access_token"].(string); ok {
		return token
	} else if errDesc, ok := accessTokenObj["error_description"].(string); ok {
		log.Printf("Error in getting access token: %s", errDesc)
	} else {
		log.Printf("Unknown error in access token response")
	}
	return ""
}
