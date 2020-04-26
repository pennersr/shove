package telegram

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
)

type telegramMessage struct {
	Method  string          `json:"method"`
	Payload json.RawMessage `json:"payload"`
	ChatID  string
}

func (tg *TelegramService) convert(data []byte) (*telegramMessage, error) {
	var msg telegramMessage
	if err := json.Unmarshal(data, &msg); err != nil {
		return nil, err
	}
	if !strings.HasPrefix(msg.Method, "send") {
		return nil, fmt.Errorf("invalid method: %s", msg.Method)
	}
	// Telegram documents the chat_id as: "Integer or String", we're assuming string.
	var chatId struct {
		ChatID string `json:"chat_id"`
	}
	if err := json.Unmarshal(msg.Payload, &chatId); err != nil {
		return nil, err
	}
	if chatId.ChatID == "" {
		return nil, errors.New("missing `chat_id`")
	}
	msg.ChatID = chatId.ChatID
	return &msg, nil
}

// Validate ...
func (tg *TelegramService) Validate(data []byte) error {
	_, err := tg.convert(data)
	return err
}
