package webpush

import (
	"encoding/json"
	wpg "github.com/SherClockHolmes/webpush-go"
	"gitlab.com/pennersr/shove/internal/services"
)

type webPushMessage struct {
	Subscription json.RawMessage `json:"subscription"`
	Payload      json.RawMessage `json:"payload"`
	Token        string          `json:"token"`
	Headers      struct {
		TTL     int    `json:"ttl"`
		Topic   string `json:"topic"`
		Urgency string `json:"urgency"`
	} `json:"headers"`

	options      wpg.Options
	subscription wpg.Subscription
}

func (msg webPushMessage) GetDigestTarget() string {
	panic("not implemented")
}

func (wp *WebPush) ConvertMessage(data []byte) (services.ServiceMessage, error) {
	var msg webPushMessage
	if err := json.Unmarshal(data, &msg); err != nil {
		return nil, err
	}
	if err := json.Unmarshal(msg.Subscription, &msg.subscription); err != nil {
		return nil, err
	}
	if msg.Token == "" {
		msg.Token = string(msg.Subscription)
	}
	msg.options = wpg.Options{
		VAPIDPublicKey:  wp.vapidPublicKey,
		VAPIDPrivateKey: wp.vapidPrivateKey,
	}
	msg.options.Topic = msg.Headers.Topic
	if msg.Headers.Urgency != "" {
		msg.options.Urgency = wpg.Urgency(msg.Headers.Urgency)
	}
	if msg.Headers.TTL > 0 {
		msg.options.TTL = msg.Headers.TTL
	}
	return msg, nil
}

// Validate ...
func (wp *WebPush) Validate(data []byte) error {
	_, err := wp.ConvertMessage(data)
	return err
}
