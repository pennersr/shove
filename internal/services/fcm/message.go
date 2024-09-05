package fcm

import (
	"encoding/json"
	"errors"
	"firebase.google.com/go/messaging"
	"gitlab.com/pennersr/shove/internal/services"
)

type fcmMessage struct {
	Message *messaging.Message `json:"message"`
}

func (fcmMessage) GetSquashKey() string {
	panic("not implemented")
}

func (fcm *FCM) ConvertMessage(data []byte) (smsg services.ServiceMessage, err error) {
	var msg fcmMessage
	if err := json.Unmarshal(data, &msg); err != nil {
		return nil, err
	}
	if msg.Message == nil {
		return nil, errors.New("message key missing")
	}
	if msg.Message.Token == "" {
		return nil, errors.New("no token specified")
	}
	return msg, nil
}

// Validate ...
func (fcm *FCM) Validate(data []byte) error {
	_, err := fcm.ConvertMessage(data)
	return err
}
