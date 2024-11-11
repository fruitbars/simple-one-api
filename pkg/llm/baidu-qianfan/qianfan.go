package baidu_qianfan

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"go.uber.org/zap"
	"io"
	"net/http"
	"simple-one-api/pkg/mylog"
	"strings"
	"time"
)

func QianFanCall(client *http.Client, api_key, secret_key, model string, configAddress string, qfReq *QianFanRequest) (*QianFanResponse, error) {
	mylog.Logger.Info("QianFanCall", zap.String("api_key", api_key), zap.String("secret_key", secret_key), zap.String("model", model), zap.Any("qfReq", qfReq))

	accessToken := GetAccessToken(api_key, secret_key)
	if accessToken == "" {
		err := errors.New("Failed to get access token")
		mylog.Logger.Error(err.Error())
		return nil, err
	}

	return SendChatRequest(client, accessToken, model, configAddress, qfReq)
}

func QianFanCallSSE(client *http.Client, api_key, secret_key, model string, configAddress string, qfReq *QianFanRequest, callback func(qfResp *QianFanResponse)) error {
	mylog.Logger.Info("QianFanCall", zap.String("api_key", api_key), zap.String("secret_key", secret_key), zap.String("model", model), zap.Any("qfReq", qfReq))
	accessToken := GetAccessToken(api_key, secret_key)
	if accessToken == "" {
		err := errors.New("Failed to get access token")
		mylog.Logger.Error(err.Error())
		return err
	}

	return SendChatRequestWithSSE(client, accessToken, model, configAddress, qfReq, callback)
}

// SendChatRequestWithSSE 发送 SSE 请求并处理响应
func SendChatRequestWithSSE(client *http.Client, accessToken, model string, configAddress string, qfReq *QianFanRequest, callback func(qfResp *QianFanResponse)) error {
	address, err := qianfanModel2Address(model)
	if err != nil {
		address = model
	} else {
		if address == "{}" {
			address = configAddress
		}
	}
	url := "https://aip.baidubce.com/rpc/2.0/ai_custom/v1/wenxinworkshop/chat/" + address + "?access_token=" + accessToken

	jsonData, err := json.Marshal(qfReq)
	if err != nil {
		mylog.Logger.Error(err.Error())
		return err
	}

	mylog.Logger.Info(string(jsonData))

	//client := &http.Client{}
	req, err := http.NewRequest("POST", url, bytes.NewReader(jsonData))
	if err != nil {
		mylog.Logger.Error(err.Error())
		return err
	}
	req.Header.Add("Content-Type", "application/json")

	res, err := client.Do(req)
	if err != nil {
		mylog.Logger.Error(err.Error())
		return err
	}

	mylog.Logger.Debug(url)

	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		mylog.Logger.Error("received non-200 response code:", zap.Int("StatusCode", res.StatusCode))
		return fmt.Errorf("received non-200 response code: %d", res.StatusCode)
	}

	// 使用 bufio.Scanner 解析 SSE 响应
	scanner := bufio.NewScanner(res.Body)
	for scanner.Scan() {
		line := scanner.Text()
		//log.Println(line)
		if strings.HasPrefix(line, "data:") || len(line) > 0 {
			data := strings.TrimPrefix(line, "data:")

			mylog.Logger.Debug(string(data))

			var response QianFanResponse
			err = json.Unmarshal([]byte(data), &response)
			if err != nil {
				mylog.Logger.Error(err.Error())

			} else {
				mylog.Logger.Debug("execute callback")
				callback(&response)
			}
		}
	}

	if err := scanner.Err(); err != nil {
		mylog.Logger.Error(err.Error())
		return err
	}

	return nil
}

func SendChatRequest(client *http.Client, accessToken, model string, configAddress string, qfReq *QianFanRequest) (*QianFanResponse, error) {
	address, err := qianfanModel2Address(model)
	if err != nil {
		address = model
	} else {
		if address == "{}" {
			address = configAddress
		}
	}
	url := "https://aip.baidubce.com/rpc/2.0/ai_custom/v1/wenxinworkshop/chat/" + address + "?access_token=" + accessToken

	jsonData, err := json.Marshal(qfReq)
	if err != nil {
		mylog.Logger.Error(err.Error())
		return nil, err
	}

	mylog.Logger.Info(string(jsonData))

	//client := &http.Client{}
	req, err := http.NewRequest("POST", url, bytes.NewReader(jsonData))
	if err != nil {
		mylog.Logger.Error(err.Error())
		return nil, err
	}
	req.Header.Add("Content-Type", "application/json")

	res, err := client.Do(req)
	if err != nil {
		mylog.Logger.Error(err.Error())
		return nil, err
	}
	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)
	if err != nil {
		mylog.Logger.Error(err.Error())
		return nil, err
	}

	if res.StatusCode != http.StatusOK {
		mylog.Logger.Error("received non-200 response code:", zap.Int("StatusCode", res.StatusCode), zap.String("body", string(body)))
		return nil, fmt.Errorf("received non-200 response code: %d", res.StatusCode)
	}

	var response QianFanResponse
	err = json.Unmarshal(body, &response)
	if err != nil {
		mylog.Logger.Error(err.Error())
		return nil, err
	}

	mylog.Logger.Info("", zap.Any("response", response))

	return &response, nil
}

// GetAccessToken 使用 AK，SK 生成鉴权签名（Access Token）
func GetAccessToken(api_key, secret_key string) string {
	url := "https://aip.baidubce.com/oauth/2.0/token"
	postData := fmt.Sprintf("grant_type=client_credentials&client_id=%s&client_secret=%s", api_key, secret_key)
	client := &http.Client{
		Timeout: 10 * time.Second,
		//Transport: nil, // 使用自定义的传输配置
	}

	resp, err := client.Post(url, "application/x-www-form-urlencoded", strings.NewReader(postData))

	if err != nil {
		mylog.Logger.Error(err.Error())
		return ""
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		mylog.Logger.Error(err.Error())
		return ""
	}

	//	log.Printf("AccessToken response body: %s", body)

	var accessTokenObj map[string]interface{}
	if err := json.Unmarshal(body, &accessTokenObj); err != nil {
		mylog.Logger.Error(err.Error())
		return ""
	}

	if token, ok := accessTokenObj["access_token"].(string); ok {
		return token
	} else if errDesc, ok := accessTokenObj["error_description"].(string); ok {
		mylog.Logger.Error("Error in getting access token:", zap.String("errDesc", errDesc))
	} else {
		mylog.Logger.Error("Unknown error in access token response")
	}
	return ""
}
