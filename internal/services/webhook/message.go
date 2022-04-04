package webhook

import (
	"encoding/json"
	"errors"
	"net/url"

	"gitlab.com/pennersr/shove/internal/services"
)

type webhookMessage struct {
	URL      string            `json:"url"`
	Headers  map[string]string `json:"headers"`
	Body     string            `json:"body"`
	Data     json.RawMessage   `json:"data"`
	postData []byte
	rawData  []byte
}

func (webhookMessage) GetSquashKey() string {
	panic("not implemented")
}

func (wh *Webhook) ConvertMessage(data []byte) (smsg services.ServiceMessage, err error) {
	var msg webhookMessage
	if err := json.Unmarshal(data, &msg); err != nil {
		return nil, err
	}
	if _, err := url.ParseRequestURI(msg.URL); err != nil {
		return nil, err
	}
	if len(msg.Body) > 0 && len(msg.Data) > 0 {
		return nil, errors.New("either body or data expected")
	}
	if len(msg.Data) > 0 {
		msg.postData = []byte(msg.Data)
		if msg.Headers == nil {
			msg.Headers = make(map[string]string)
		}
		msg.Headers["content-type"] = "application/json"
	} else if len(msg.Body) > 0 {
		msg.postData = []byte(msg.Body)
	}
	msg.rawData = data
	return msg, nil
}

// Validate ...
func (wh *Webhook) Validate(data []byte) error {
	_, err := wh.ConvertMessage(data)
	return err
}
