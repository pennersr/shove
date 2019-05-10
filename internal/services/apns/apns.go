package apns

import (
	"context"
	"github.com/sideshow/apns2"
	"github.com/sideshow/apns2/certificate"
	"gitlab.com/pennersr/shove/internal/queue"
	"gitlab.com/pennersr/shove/internal/services"
	"log"
	"math"
	"sync"
	"time"
)

// APNS ...
type APNS struct {
	clients    []*apns2.Client
	production bool
	wg         sync.WaitGroup
}

// NewAPNS ...
func NewAPNS(pemFile string, production bool) (apns *APNS, err error) {
	cert, err := certificate.FromPemFile(pemFile, "")
	if err != nil {
		return
	}
	apns = &APNS{production: production}
	for i := 0; i < 4; i++ {
		client := apns2.NewClient(cert)
		if production {
			client.Production()
		} else {
			client.Development()
		}
		apns.clients = append(apns.clients, client)
	}
	return
}

// ID ...
func (apns *APNS) ID() string {
	if apns.production {
		return "apns"
	}
	return "apns-sandbox"

}

// String ...
func (apns *APNS) String() string {
	if apns.production {
		return "APNS"
	}
	return "APNS-sandbox"
}

func (apns *APNS) serveClient(ctx context.Context, q queue.Queue, id int, client *apns2.Client, fc services.FeedbackCollector) {
	defer func() {
		apns.wg.Done()
	}()
	failureCount := 0
	for ctx.Err() == nil {
		qm, err := q.Get(ctx)
		if err != nil {
			log.Println(apns, "error reading from queue:", err)
			return
		}
		var sent, retry bool
		msg := qm.Message()
		notif, err := apns.convert(msg)
		if err != nil {
			log.Println(apns, "bad message:", err)
			apns.remove(q, qm)
			continue
		}
		t := time.Now()
		resp, err := client.Push(notif)
		if err != nil {
			log.Println(apns, "error pushing:", err)
			retry = true
		} else {
			status := resp.Reason
			if status == "" {
				status = "OK"
			}
			log.Printf("%s pushed (%s), took %s", apns, status, time.Now().Sub(t))
			sent = resp.Sent()
			if resp.Reason == apns2.ReasonBadDeviceToken || resp.Reason == apns2.ReasonUnregistered {
				fc.TokenInvalid(apns.ID(), notif.DeviceToken)
			}
			retry = resp.StatusCode >= 500
		}
		if sent || !retry {
			apns.remove(q, qm)
		} else {
			if err = q.Requeue(qm); err != nil {
				log.Println(apns, "error putting back in the queue:", err)
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

func (apns *APNS) remove(q queue.Queue, qm queue.QueuedMessage) {
	if err := q.Remove(qm); err != nil {
		log.Println(apns, "error removing from the queue:", err)
	}
}

func backoff(ctx context.Context, failureCount int) {
	sleep := time.Duration(float64(time.Second) * math.Min(30, math.Pow(2., float64(failureCount))))
	log.Printf("Backing off for %s", sleep)
	ctx, cancel := context.WithTimeout(ctx, sleep)
	defer cancel()
	<-ctx.Done()
}

// Serve ...
func (apns *APNS) Serve(ctx context.Context, q queue.Queue, fc services.FeedbackCollector) (err error) {
	apns.wg.Add(len(apns.clients))
	for id, client := range apns.clients {
		go apns.serveClient(ctx, q, id, client, fc)
	}
	log.Println(apns, "workers started")
	apns.wg.Wait()
	log.Println(apns, "workers stopped")
	return
}
