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
	wg      sync.WaitGroup
	log     *log.Logger
	adapter PumpAdapter
	workers int
}

type ServiceMessage interface {
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
	PushMessage(client PumpClient, sm ServiceMessage, fc FeedbackCollector) PushStatus
}

// NewPump
func NewPump(workers int, log *log.Logger, adapter PumpAdapter) (p *Pump) {
	return &Pump{
		log:     log,
		workers: workers,
		adapter: adapter,
	}
}

func (p *Pump) serveClient(ctx context.Context, q queue.Queue, client PumpClient, fc FeedbackCollector) {
	defer func() {
		p.wg.Done()
	}()
	failureCount := 0
	for ctx.Err() == nil {
		qm, err := q.Get(ctx)
		if err != nil {
			p.log.Println("[ERROR] Reading from queue:", err)
			return
		}
		msg := qm.Message()
		cmsg, err := p.adapter.ConvertMessage(msg)
		if err != nil {
			p.log.Println("[ERROR] Bad message:", err)
			p.remove(q, qm)
			continue
		}
		status := p.adapter.PushMessage(client, cmsg, fc)
		if status == PushStatusSuccess || status == PushStatusHardFail {
			p.remove(q, qm)
		} else {
			if err = q.Requeue(qm); err != nil {
				p.log.Println("[ERROR] Putting back in the queue:", err)
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

func (p *Pump) remove(q queue.Queue, qm queue.QueuedMessage) {
	if err := q.Remove(qm); err != nil {
		p.log.Println("[ERROR] Removing from the queue:", err)
	}
}

func (p *Pump) backoff(ctx context.Context, failureCount int) {
	sleep := time.Duration(float64(time.Second) * math.Min(30, math.Pow(2., float64(failureCount))))
	p.log.Printf("Backing off for %s", sleep)
	ctx, cancel := context.WithTimeout(ctx, sleep)
	defer cancel()
	<-ctx.Done()
}

func (p *Pump) Serve(ctx context.Context, q queue.Queue, fc FeedbackCollector) (err error) {
	clients := make([]PumpClient, p.workers)
	for i := 0; i < p.workers; i++ {
		clients[i], err = p.adapter.NewClient()
		if err != nil {
			return
		}
	}

	for i := 0; i < p.workers; i++ {
		go p.serveClient(ctx, q, clients[i], fc)
		p.wg.Add(1)
	}
	p.log.Println("Workers started")
	p.wg.Wait()
	p.log.Println("Workers stopped")

	return
}
