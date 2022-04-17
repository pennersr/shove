package webpush

import (
	wpg "github.com/SherClockHolmes/webpush-go"
	"gitlab.com/pennersr/shove/internal/queue"
	"gitlab.com/pennersr/shove/internal/services"
	"log"
	"net/http"
	"time"
)

// WebPush ...
type WebPush struct {
	vapidPublicKey  string
	vapidPrivateKey string
	log             *log.Logger
}

// NewWebPush ...
func NewWebPush(vapidPub, vapidPvt string, log *log.Logger) (wp *WebPush, err error) {
	wp = &WebPush{
		vapidPrivateKey: vapidPvt,
		vapidPublicKey:  vapidPub,
		log:             log,
	}
	return
}

func (wp *WebPush) Logger() *log.Logger {
	return wp.log
}

func (wp *WebPush) NewClient() (services.PumpClient, error) {
	client := &http.Client{
		Timeout: time.Duration(15 * time.Second),
		Transport: &http.Transport{
			MaxIdleConns:    5,
			IdleConnTimeout: 30 * time.Second,
		},
	}
	return client, nil
}

// ID ...
func (wp *WebPush) ID() string {
	return "webpush"
}

// String ...
func (wp *WebPush) String() string {
	return "WebPush"
}

func (wp *WebPush) SquashAndPushMessage(client services.PumpClient, smsgs []services.ServiceMessage, fc services.FeedbackCollector) services.PushStatus {
	panic("not implemented")
}

func (wp *WebPush) PushMessage(pclient services.PumpClient, smsg services.ServiceMessage, fc services.FeedbackCollector) services.PushStatus {
	success := false
	msg := smsg.(webPushMessage)
	msg.options.HTTPClient = pclient.(*http.Client)
	startedAt := time.Now()
	// Send Notification
	resp, err := wpg.SendNotification(msg.Payload, &msg.subscription, &msg.options)
	if err != nil {
		wp.log.Println("[ERROR] Sending:", err)
		return services.PushStatusHardFail
	}
	defer resp.Body.Close()
	duration := time.Now().Sub(startedAt)
	wp.log.Printf("Pushed (%d), took %s", resp.StatusCode, duration)
	defer func() {
		fc.CountPush(wp.ID(), success, duration)
	}()
	switch resp.StatusCode {
	case 201:
		//  201 Created. The request to send a push message was received and accepted.
		success = true
		return services.PushStatusSuccess

	case 429:
		// 429 Too many requests. Meaning your application server has
		// reached a rate limit with a push service. The push service
		// should include a 'Retry-After' header to indicate how long
		// before another request can be made.
		return services.PushStatusTempFail

	case 400:
		// 400 Invalid request. This generally means one of your headers is invalid or improperly formatted.
		return services.PushStatusHardFail

	case 404:
		// 404 Not Found. This is an indication that the subscription is
		// expired and can't be used. In this case you should delete the
		// `PushSubscription` and wait for the client to resubscribe the
		// user.
		fallthrough
	case 410:
		// 410 Gone. The subscription is no longer valid and should be
		// removed from application server. This can be reproduced by
		// calling `unsubscribe()` on a `PushSubscription`.
		fc.TokenInvalid(wp.ID(), msg.Token)
		return services.PushStatusHardFail

	default:
		// 413 Payload size too large. The minimum size payload a push service must support is 4096 bytes (or 4kb).
		return services.PushStatusHardFail
	}
}

func (wp *WebPush) remove(q queue.Queue, qm queue.QueuedMessage) {
	if err := q.Remove(qm); err != nil {
		wp.log.Println("[ERROR] Removing from the queue:", err)
	}
}
