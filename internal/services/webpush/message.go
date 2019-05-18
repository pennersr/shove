package webpush

import (
	"encoding/json"
	wpg "github.com/SherClockHolmes/webpush-go"
	"net/http"
	"time"
)

type webPushMessage struct {
	Subscription wpg.Subscription `json:"subscription"`
	Payload      json.RawMessage  `json:"payload"`
	Headers      struct {
		TTL     int    `json:"ttl"`
		Topic   string `json:"topic"`
		Urgency string `json:"urgency"`
	} `json:"headers"`

	options wpg.Options
}

func (wp *WebPush) convert(data []byte) (*webPushMessage, error) {
	var msg webPushMessage
	if err := json.Unmarshal(data, &msg); err != nil {
		return nil, err
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
	msg.options.HTTPClient = &http.Client{
		Timeout:   time.Duration(15 * time.Second),
		Transport: wp.transport,
	}
	return &msg, nil
}

// Validate ...
func (wp *WebPush) Validate(data []byte) error {
	_, err := wp.convert(data)
	return err
}
