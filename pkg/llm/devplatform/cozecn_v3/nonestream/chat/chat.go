package chat

import (
	"encoding/json"
	"log"
	"net/http"
	"simple-one-api/pkg/llm/devplatform/cozecn_v3/common"
)

func Chat(token string, chatRequest *common.ChatRequest, httpTransport *http.Transport) (*Response, error) {
	serverURL := "https://api.coze.cn/v3/chat"

	reqData, _ := json.Marshal(chatRequest)
	respData, err := common.SendCozeV3HTTPRequest(token, serverURL, reqData, httpTransport)
	if err != nil {
		log.Println(err)
		return nil, err
	}

	var respJson Response
	json.Unmarshal(respData, &respJson)

	log.Println(respJson)

	return &respJson, err
}
