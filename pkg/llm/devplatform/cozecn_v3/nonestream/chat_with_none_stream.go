package nonestream

import (
	"log"
	"net/http"
	"simple-one-api/pkg/llm/devplatform/cozecn_v3/common"
	"simple-one-api/pkg/llm/devplatform/cozecn_v3/nonestream/chat"
	"simple-one-api/pkg/llm/devplatform/cozecn_v3/nonestream/chat_message_list"
	"simple-one-api/pkg/llm/devplatform/cozecn_v3/nonestream/chat_retrieve"
	"time"
)

func ChatWithNoneStream(token string, chatRequest *common.ChatRequest, httpTransport *http.Transport, timeout int) (*chat_message_list.MessageListResponse, error) {

	chatResp, err := chat.Chat(token, chatRequest, httpTransport)
	if err != nil {
		log.Println(err)
		return nil, err
	}

	if timeout < 0 {
		timeout = 60
	}

	for i := 0; i < timeout; i++ {
		chatRetrieveResp, err := chat_retrieve.ChatRetrieve(chatResp.Data.ID, chatResp.Data.ConversationID, token)
		if err != nil {
			log.Println(err)
			return nil, err
		}

		if chatRetrieveResp.Data.Status == chat_retrieve.StatusCreated || chatRetrieveResp.Data.Status == chat_retrieve.StatusInProgress {
			time.Sleep(1 * time.Second)
			continue
		} else if chatRetrieveResp.Data.Status == chat_retrieve.StatusCompleted {
			messageListResponse, err := chat_message_list.ChatMessageslist(chatResp.Data.ID, chatResp.Data.ConversationID, token)
			if err != nil {
				log.Println(err)
				return nil, err
			}

			return messageListResponse, nil
		} else {
			log.Println(chatRetrieveResp)
			break
		}
	}

	return nil, err
}
