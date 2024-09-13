package streammode

import (
	"encoding/json"
	"log"
	"net/http"
	"simple-one-api/pkg/llm/devplatform/cozecn_v3/common"
)

func Chat(token string, chatRequest *common.ChatRequest, callback func(event, data string), httpTransport *http.Transport) error {
	serverURL := "https://api.coze.cn/v3/chat"

	reqData, _ := json.Marshal(chatRequest)

	err := common.SendCozeV3StreamHttpRequest(token, serverURL, reqData, callback, httpTransport)
	if err != nil {
		log.Println(err)
		return err
	}

	return err
}
