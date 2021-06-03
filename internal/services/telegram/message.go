package telegram

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"gitlab.com/pennersr/shove/internal/services"
)

type telegramMessage struct {
	Method string `json:"method"`
	// This is intentionally kept as a raw message so that this can be fed 1:1
	// to the API.
	Payload       json.RawMessage `json:"payload"`
	parsedPayload telegramPayload
}

type telegramPayload struct {
	ChatID  string `json:"chat_id"`
	Text    string `json:"text,omitempty"`
	Caption string `json:"caption,omitempty"`
	Photo   string `json:"photo,omitempty"`
}

func (msg telegramMessage) GetSquashKey() string {
	// TODO: This should include method (`sendMessage`)
	return msg.parsedPayload.ChatID
}

func (tg *TelegramService) ConvertMessage(data []byte) (services.ServiceMessage, error) {
	var msg telegramMessage
	if err := json.Unmarshal(data, &msg); err != nil {
		return nil, err
	}
	if !strings.HasPrefix(msg.Method, "send") {
		return nil, fmt.Errorf("invalid method: %s", msg.Method)
	}
	// Telegram documents the chat_id as: "Integer or String", we're assuming string.
	if err := json.Unmarshal(msg.Payload, &msg.parsedPayload); err != nil {
		return nil, err
	}
	if msg.parsedPayload.ChatID == "" {
		return nil, errors.New("missing `chat_id`")
	}
	return msg, nil
}

// Validate ...
func (tg *TelegramService) Validate(data []byte) error {
	_, err := tg.ConvertMessage(data)
	return err
}

func squashMessages(msgs []telegramMessage) (dmsg telegramMessage, err error) {
	if len(msgs) == 0 {
		err = errors.New("need at least one message to digest")
		return
	}
	dmsg = msgs[0]
	var text strings.Builder
	for i, msg := range msgs {
		if msg.Method != dmsg.Method {
			err = errors.New("cannot digest mix of methods")
			return
		}
		if msg.parsedPayload.ChatID != dmsg.parsedPayload.ChatID {
			err = errors.New("different `chat_id` seen while digesting")
			return
		}
		if i > 0 {
			text.WriteString("\n\n")
		}
		if msg.parsedPayload.Text != "" {
			text.WriteString(msg.parsedPayload.Text)
		}
	}
	dmsg.parsedPayload.Text = text.String()
	dmsg.Payload, err = json.Marshal(&dmsg.parsedPayload)
	return
}
