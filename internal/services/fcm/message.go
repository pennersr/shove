package fcm

import (
	"codeberg.org/pennersr/shove/internal/services"
	"encoding/json"
	"errors"
	"firebase.google.com/go/v4/messaging"
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
	// Token is deprecated in favor of Fid (Firebase Installation ID), but the
	// two are distinct identifiers and clients still send registration tokens.
	// Both remain supported, so we keep Token intentionally.
	// See: https://firebase.google.com/docs/cloud-messaging/send/admin-sdk
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
