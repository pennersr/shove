package email

import (
	"context"
	"log"
	"math"
	"sync"
	"time"

	"gitlab.com/pennersr/shove/internal/queue"
	"gitlab.com/pennersr/shove/internal/services"
)

const serviceID = "email"

type EmailConfig struct {
	EmailHost string
	EmailPort int
	RateMax   int
	RatePer   time.Duration
	Log       *log.Logger
}

type EmailService struct {
	config   EmailConfig
	digester digester
	wg       sync.WaitGroup
}

func NewEmailService(config EmailConfig) (es *EmailService, err error) {
	es = &EmailService{
		config: config,
	}
	es.digester.init(config)
	return
}

func (es *EmailService) ID() string {
	return serviceID
}

func (es *EmailService) String() string {
	return "Email"
}

func (es *EmailService) push(q queue.Queue, qm queue.QueuedMessage, email email, data []byte, fc services.FeedbackCollector) (success, retry bool) {
	digested := es.digester.prepareToMail(q, qm, email)
	if digested {
		return true, false
	}
	es.config.Log.Println("Sending email")
	body, err := encodeEmail(email)
	if err != nil {
		return false, false
	}
	err = es.config.send(email.From, email.To, body, fc)
	if err != nil {
		es.config.Log.Printf("[ERROR] Sending email failed: %s", err)
		return false, false
	}
	return
}

func (es *EmailService) Serve(ctx context.Context, q queue.Queue, fc services.FeedbackCollector) (err error) {
	es.wg.Add(1)
	go func() {
		es.config.Log.Println("Digester started")
		es.digester.serve(fc)
		es.config.Log.Println("Digester stopped")
		es.wg.Add(-1)
	}()
	es.config.Log.Println("Worker started")
	failureCount := 0
	for ctx.Err() == nil {
		var qm queue.QueuedMessage
		qm, err := q.Get(ctx)
		if err != nil {
			es.config.Log.Printf("[ERROR] Reading from queue: %s", err)
			es.digester.requestShutdown()
			break
		}
		msg := qm.Message()
		emsg, err := es.convert(msg)
		if err != nil {
			es.config.Log.Printf("[ERROR] Bad message: %s", err)
			es.remove(q, qm)
			continue
		}
		success, retry := es.push(q, qm, emsg, msg, fc)
		if success || !retry {
			es.remove(q, qm)
		} else {
			if err = q.Requeue(qm); err != nil {
				es.config.Log.Printf("[ERROR] Requeue failed: %s", err)
			}
		}
		if retry {
			es.backoff(ctx, failureCount)
			failureCount++

		} else {
			failureCount = 0

		}
	}
	es.wg.Wait()
	es.config.Log.Println("Worker stopped")
	return
}

func (es *EmailService) backoff(ctx context.Context, failureCount int) {
	sleep := time.Duration(float64(time.Second) * math.Min(30, math.Pow(2., float64(failureCount))))
	es.config.Log.Printf("Backing off for %s", sleep)
	ctx, cancel := context.WithTimeout(ctx, sleep)
	defer cancel()
	<-ctx.Done()
}

func (es *EmailService) remove(q queue.Queue, qm queue.QueuedMessage) {
	if err := q.Remove(qm); err != nil {
		es.config.Log.Printf("[ERROR] %s: remove from queue failed:", err)
	}
}
