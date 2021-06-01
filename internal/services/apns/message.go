package apns

import (
	"encoding/json"
	"errors"
	"github.com/sideshow/apns2"
	"gitlab.com/pennersr/shove/internal/services"
	"time"
)

type apnsMessage struct {
	Token   string                     `json:"token"`
	Headers map[string]json.RawMessage `json:"headers,omitempty"`
	Payload json.RawMessage            `json:"payload,omitempty"`
}

type apnsNotification struct {
	notification *apns2.Notification
}

func (notif apnsNotification) GetDigestTarget() string {
	panic("not implemented")
}

func (apns *APNS) ConvertMessage(data []byte) (smsg services.ServiceMessage, err error) {
	var msg apnsMessage
	if err = json.Unmarshal(data, &msg); err != nil {
		return
	}
	if msg.Token == "" {
		err = errors.New("token required")
		return
	}

	notif := new(apns2.Notification)
	notif.DeviceToken = msg.Token
	topic, ok := msg.Headers["apns-topic"]
	if !ok {
		err = errors.New("APNS requires a topic")
		return
	}
	err = json.Unmarshal(topic, &notif.Topic)
	if err != nil {
		return
	}
	prio, ok := msg.Headers["apns-priority"]
	if ok {
		err = json.Unmarshal(prio, &notif.Priority)
		if err != nil {
			return
		}
	}
	collapse, ok := msg.Headers["apns-collapse-id"]
	if ok {
		err = json.Unmarshal(collapse, &notif.CollapseID)
		if err != nil {
			return
		}
	}
	exp, ok := msg.Headers["apns-expiration"]
	if ok {
		var epoch int64
		err = json.Unmarshal(exp, &epoch)
		if err != nil {
			return
		}
		notif.Expiration = time.Unix(epoch, 0)
	}
	notif.Payload = msg.Payload
	smsg = apnsNotification{notification: notif}
	return
}

// Validate ...
func (apns *APNS) Validate(data []byte) (err error) {
	_, err = apns.ConvertMessage(data)
	return
}
