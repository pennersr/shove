package services

import (
	"context"
	"log"
	"math"
	"sync"
	"time"

	"gitlab.com/pennersr/shove/internal/queue"
)

type Pump struct {
	wg       sync.WaitGroup
	adapter  PumpAdapter
	workers  int
	squasher *squasher
}

type ServiceMessage interface {
	GetSquashKey() string
}

type PushStatus int

const (
	// PushStatusSuccess ...
	PushStatusSuccess PushStatus = iota
	// PushStatusTempFail signals a failure that may be resolved by retrying
	PushStatusTempFail
	// PushStatusHardFail signals a failure for which a retry would not hekp
	PushStatusHardFail
)

type PumpClient interface {
}

type PumpAdapter interface {
	ConvertMessage([]byte) (ServiceMessage, error)
	NewClient() (PumpClient, error)
	PushMessage(client PumpClient, smsg ServiceMessage, fc FeedbackCollector) PushStatus
	SquashAndPushMessage(client PumpClient, smsgs []ServiceMessage, fc FeedbackCollector) PushStatus
	Logger() *log.Logger
}

// NewPump
func NewPump(workers int, squash SquashConfig, adapter PumpAdapter) (p *Pump) {
	p = &Pump{
		workers: workers,
		adapter: adapter,
	}
	if squash.RateMax > 0 {
		p.squasher = newSquasher(squash, adapter)
	}
	return p
}

func (p *Pump) push(q queue.Queue, qm queue.QueuedMessage, client PumpClient, smsg ServiceMessage, fc FeedbackCollector) (status PushStatus, squashed bool) {
	if p.squasher != nil {
		squashed = p.squasher.prepareToPush(q, qm, client, smsg)
		if squashed {
			return
		}
	}
	status = p.adapter.PushMessage(client, smsg, fc)
	return
}

func (p *Pump) serveClient(ctx context.Context, q queue.Queue, client PumpClient, fc FeedbackCollector) {
	defer func() {
		p.wg.Done()
	}()
	failureCount := 0
	log := p.adapter.Logger()
	for ctx.Err() == nil {
		qm, err := q.Get(ctx)
		if err != nil {
			log.Println("[ERROR] Reading from queue:", err)
			return
		}
		msg := qm.Message()
		smsg, err := p.adapter.ConvertMessage(msg)
		if err != nil {
			log.Println("[ERROR] Bad message:", err)
			removeFromQueue(q, qm, log)
			continue
		}
		status, squashed := p.push(q, qm, client, smsg, fc)
		if squashed {
			// Message should remain in pending queue
			continue
		}
		if status == PushStatusSuccess || status == PushStatusHardFail {
			removeFromQueue(q, qm, log)
		} else {
			if err = q.Requeue(qm); err != nil {
				log.Println("[ERROR] Putting back in the queue:", err)
			}
		}
		if status == PushStatusTempFail {
			p.backoff(ctx, failureCount)
			failureCount++

		} else {
			failureCount = 0
		}
	}
}

func removeFromQueue(q queue.Queue, qm queue.QueuedMessage, log *log.Logger) {
	if err := q.Remove(qm); err != nil {
		log.Println("[ERROR] Removing from the queue:", err)
	}
}

func (p *Pump) backoff(ctx context.Context, failureCount int) {
	sleep := time.Duration(float64(time.Second) * math.Min(30, math.Pow(2., float64(failureCount))))
	p.adapter.Logger().Printf("Backing off for %s", sleep)
	ctx, cancel := context.WithTimeout(ctx, sleep)
	defer cancel()
	<-ctx.Done()
}

func (p *Pump) Serve(ctx context.Context, q queue.Queue, fc FeedbackCollector) (err error) {
	log := p.adapter.Logger()
	if p.squasher != nil {
		p.wg.Add(1)
		go func() {
			log.Println("Squasher started")
			p.squasher.serve(fc)
			log.Println("Squasher stopped")
			p.wg.Add(-1)
		}()
	}
	clients := make([]PumpClient, p.workers)
	for i := 0; i < p.workers; i++ {
		clients[i], err = p.adapter.NewClient()
		if err != nil {
			return
		}
	}

	for i := 0; i < p.workers; i++ {
		go func(client PumpClient) {
			p.serveClient(ctx, q, client, fc)
			if p.squasher != nil {
				p.squasher.requestShutdown()
			}
		}(clients[i])
		p.wg.Add(1)
	}
	log.Println("Workers started:", p.workers)
	p.wg.Wait()
	log.Println("Workers stopped")

	return
}
