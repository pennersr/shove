package services

import (
	"sync"
	"time"

	"gitlab.com/pennersr/shove/internal/queue"
)

type batch struct {
	key         string
	serviceMsgs []ServiceMessage
	due         time.Time
	queuedMsgs  []queue.QueuedMessage
	q           queue.Queue
	client      PumpClient
}

type SquashConfig struct {
	RateMax int
	RatePer time.Duration
}

type squasher struct {
	pushedAt     map[string][]time.Time
	batches      map[string]batch
	config       SquashConfig
	cond         *sync.Cond
	lock         sync.Mutex
	shuttingDown bool
	adapter      PumpAdapter
}

func newSquasher(config SquashConfig, adapter PumpAdapter) (d *squasher) {
	d = new(squasher)
	d.adapter = adapter
	d.config = config
	d.pushedAt = make(map[string][]time.Time)
	d.batches = make(map[string]batch)
	d.cond = sync.NewCond(&d.lock)
	return d
}

func (d *squasher) flushAndGetRate(key string) (sendCount int, sentAt time.Time) {
	var flushedTimes []time.Time
	times := d.pushedAt[key]
	var didFlush = false
	for _, t := range times {
		if time.Since(t) > d.config.RatePer {
			didFlush = true
			continue
		}
		flushedTimes = append(flushedTimes, t)
		sendCount++
	}
	if didFlush {
		d.pushedAt[key] = flushedTimes
	}
	if len(flushedTimes) > 0 {
		sentAt = flushedTimes[0]
	}
	return
}

func (d *squasher) recordPush(key string) {
	times := d.pushedAt[key]
	times = append(times, time.Now())
	d.pushedAt[key] = times
}

func (d *squasher) prepareToPush(q queue.Queue, qm queue.QueuedMessage, client PumpClient, smsg ServiceMessage) (squashed bool) {
	d.cond.L.Lock()
	defer d.cond.L.Unlock()

	key := smsg.GetSquashKey()
	sendCount, firstSendAt := d.flushAndGetRate(key)
	if sendCount < d.config.RateMax {
		d.recordPush(key)
		return false
	}
	d.adapter.Logger().Printf("Rate to %s exceeded, squashed", key)

	batch, ok := d.batches[key]
	if ok {
		if batch.q != q {
			panic("squasher cannot handle mixed queues")
		}
	} else {
		batch.q = q
		batch.client = client
	}
	batch.key = key
	batch.serviceMsgs = append(batch.serviceMsgs, smsg)
	batch.queuedMsgs = append(batch.queuedMsgs, qm)
	batch.due = firstSendAt.Add(d.config.RatePer)
	d.batches[key] = batch
	d.cond.Signal()
	return true
}

func (d *squasher) getNextBatch() (b batch, stopped bool) {
	for {
		d.cond.L.Lock()
		if len(d.batches) == 0 {
			d.cond.Wait()
		}
		if d.shuttingDown {
			d.cond.L.Unlock()
			stopped = true
			return
		}
		var minDueBatch batch
		var minDueBatchKey string
		for key, batch := range d.batches {
			if minDueBatch.due.IsZero() || minDueBatch.due.After(batch.due) {
				minDueBatch = batch
				minDueBatchKey = key
			}
		}
		now := time.Now()
		if now.After(minDueBatch.due) {
			delete(d.batches, minDueBatchKey)
			d.cond.L.Unlock()
			return minDueBatch, false
		}
		d.cond.L.Unlock()

		zzz := minDueBatch.due.Sub(now)
		maxZzz := time.Millisecond * 500
		if zzz > maxZzz {
			zzz = maxZzz
		}
		time.Sleep(zzz)
	}
	stopped = true
	return
}

func (d *squasher) requestShutdown() {
	d.cond.L.Lock()
	d.shuttingDown = true
	d.cond.Signal()
	d.cond.L.Unlock()
}

func (d *squasher) shutdown() {
	d.cond.L.Lock()
	defer d.cond.L.Unlock()

	d.adapter.Logger().Println("Shutting down squasher:", len(d.batches), "batches unsent")
}

func (d *squasher) serve(fc FeedbackCollector) {
	for {
		batch, stopped := d.getNextBatch()
		if stopped {
			d.shutdown()
			return
		}
		d.sendBatch(batch, fc)

	}
}

func (d *squasher) sendBatch(b batch, fc FeedbackCollector) {
	d.adapter.Logger().Println("Sending batch of", len(b.serviceMsgs), "messages")
	d.cond.L.Lock()
	d.recordPush(b.key)
	d.cond.L.Unlock()

	status := d.adapter.SquashAndPushMessage(b.client, b.serviceMsgs, fc)
	switch status {
	case PushStatusTempFail:
		// TODO: We should actually attempt to retry this with a backoff
		fallthrough
	case PushStatusHardFail:
		d.adapter.Logger().Println("[ERROR] Failed to send batch")
		fallthrough
	case PushStatusSuccess:
		for _, qm := range b.queuedMsgs {
			removeFromQueue(b.q, qm, d.adapter.Logger())
		}
	}
}
