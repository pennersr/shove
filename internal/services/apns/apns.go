package apns

import (
	"context"
	"errors"
	"github.com/sideshow/apns2"
	"github.com/sideshow/apns2/certificate"
	"gitlab.com/pennersr/shove/internal/queue"
	"gitlab.com/pennersr/shove/internal/services"
	"gitlab.com/pennersr/shove/internal/types"
	"log"
	"runtime"
	"sync"
)

type APNS struct {
	clients    []*apns2.Client
	production bool
	wg         sync.WaitGroup
}

func NewAPNS(pemFile string, production bool) (apns *APNS, err error) {
	cert, err := certificate.FromPemFile(pemFile, "")
	if err != nil {
		return
	}
	apns = &APNS{production: production}
	for i := 0; i < runtime.NumCPU(); i++ {
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

func (apns *APNS) ID() string {
	if apns.production {
		return "apns"
	}
	return "apns-sandbox"

}

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
	for ctx.Err() == nil {
		qm, err := q.Get(ctx)
		if err != nil {
			log.Println(apns, "error reading from queue:", err)
			return
		}
		msg := qm.Message()
		resp, err := apns.push(client, msg)
		sent := false
		retry := false
		if err != nil {
			log.Println(apns, "error pushing:", err)
			retry = true
		} else {
			log.Printf("%s pushed: %s", apns, resp.Reason)
			sent = resp.Sent()
			if resp.Reason == apns2.ReasonBadDeviceToken || resp.Reason == apns2.ReasonUnregistered {
				fc.TokenInvalid(apns.ID(), msg.Tokens[0])
			}
			retry = resp.StatusCode >= 500
		}
		if sent || !retry {
			if err = q.Remove(qm); err != nil {
				log.Println(apns, "error removing from the queue")
			}
		} else {
			if err = q.Requeue(qm); err != nil {
				log.Println(apns, "error putting back in the queue")
			}
		}
	}
}

func (apns *APNS) push(client *apns2.Client, pm types.PushMessage) (resp *apns2.Response, err error) {
	notification := &apns2.Notification{}
	notification.DeviceToken = pm.Tokens[0]
	notification.Topic = pm.Topic
	notification.Payload = pm.Payload
	resp, err = client.Push(notification)
	return
}

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

func (apns *APNS) Validate(pm types.PushMessage) (err error) {
	if len(pm.Tokens) != 1 {
		err = errors.New("APNS expects exactly one token")
		return
	}
	if pm.Topic == "" {
		err = errors.New("APNS requires a topic")
		return
	}
	return
}
