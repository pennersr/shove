package webpush

import (
	"context"
	wpg "github.com/SherClockHolmes/webpush-go"
	"gitlab.com/pennersr/shove/internal/queue"
	"gitlab.com/pennersr/shove/internal/services"
	"log"
	"math"
	"net/http"
	"sync"
	"time"
)

// WebPush ...
type WebPush struct {
	vapidPublicKey  string
	vapidPrivateKey string
	transport       *http.Transport
	wg              sync.WaitGroup
}

// NewWebPush ...
func NewWebPush(vapidPub, vapidPvt string) (wp *WebPush, err error) {
	wp = &WebPush{
		vapidPrivateKey: vapidPvt,
		vapidPublicKey:  vapidPub,
		transport: &http.Transport{
			MaxIdleConns:    5,
			IdleConnTimeout: 30 * time.Second,
		},
	}
	return
}

// ID ...
func (wp *WebPush) ID() string {
	return "webpush"
}

// String ...
func (wp *WebPush) String() string {
	return "WebPush"
}

func (wp *WebPush) serveClient(ctx context.Context, q queue.Queue, fc services.FeedbackCollector) {
	defer func() {
		wp.wg.Done()
	}()
	failureCount := 0
	for ctx.Err() == nil {
		qm, err := q.Get(ctx)
		if err != nil {
			log.Println(wp, "error reading from queue:", err)
			return
		}
		msg := qm.Message()
		notif, err := wp.convert(msg)
		if err != nil {
			log.Println(wp, "bad message:", err)
			wp.remove(q, qm)
			continue
		}
		success, retry := wp.push(notif, msg, fc)
		if success || !retry {
			wp.remove(q, qm)
		} else {
			if err = q.Requeue(qm); err != nil {
				log.Println(wp, "error putting back in the queue:", err)
			}
		}
		if retry {
			backoff(ctx, failureCount)
			failureCount++

		} else {
			failureCount = 0

		}
	}
}

func (wp *WebPush) push(msg *webPushMessage, data []byte, fc services.FeedbackCollector) (success, retry bool) {
	startedAt := time.Now()
	// Send Notification
	resp, err := wpg.SendNotification(msg.Payload, &msg.Subscription, &msg.options)
	if err != nil {
		log.Println(wp, "error sending:", err)
		return false, false
	}
	duration := time.Now().Sub(startedAt)
	log.Printf("%s pushed (%d), took %s", wp, resp.StatusCode, duration)
	switch resp.StatusCode {
	case 201:
		//  201 Created. The request to send a push message was received and accepted.
		return true, false

	case 429:
		// 429 Too many requests. Meaning your application server has
		// reached a rate limit with a push service. The push service
		// should include a 'Retry-After' header to indicate how long
		// before another request can be made.
		return false, true

	case 400:
		// 400 Invalid request. This generally means one of your headers is invalid or improperly formatted.
		return false, false

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

		// TODO: feedback

		return false, false

	default:
		// 413 Payload size too large. The minimum size payload a push service must support is 4096 bytes (or 4kb).
		return false, false
	}
}

func backoff(ctx context.Context, failureCount int) {
	sleep := time.Duration(float64(time.Second) * math.Min(30, math.Pow(2., float64(failureCount))))
	log.Printf("Backing off for %s", sleep)
	ctx, cancel := context.WithTimeout(ctx, sleep)
	defer cancel()
	<-ctx.Done()
}

func (wp *WebPush) remove(q queue.Queue, qm queue.QueuedMessage) {
	if err := q.Remove(qm); err != nil {
		log.Println(wp, "error removing from the queue:", err)
	}
}

// Serve ...
func (wp *WebPush) Serve(ctx context.Context, q queue.Queue, fc services.FeedbackCollector) (err error) {
	for i := 0; i < 8; i++ {
		go wp.serveClient(ctx, q, fc)
		wp.wg.Add(1)
	}
	log.Println(wp, "workers started")
	wp.wg.Wait()
	log.Println(wp, "workers stopped")

	return
}
