package services

import (
	"sync"
	"time"

	"gitlab.com/pennersr/shove/internal/queue"
)

type batch struct {
	target      string
	serviceMsgs []ServiceMessage
	due         time.Time
	queuedMsgs  []queue.QueuedMessage
	q           queue.Queue
	client      PumpClient
}

type DigestConfig struct {
	RateMax int
	RatePer time.Duration
}

type digester struct {
	pushedAt     map[string][]time.Time
	batches      map[string]batch
	config       DigestConfig
	cond         *sync.Cond
	lock         sync.Mutex
	shuttingDown bool
	adapter      PumpAdapter
}

func newDigester(config DigestConfig, adapter PumpAdapter) (d *digester) {
	d = new(digester)
	d.adapter = adapter
	d.config = config
	d.pushedAt = make(map[string][]time.Time)
	d.batches = make(map[string]batch)
	d.cond = sync.NewCond(&d.lock)
	return d
}

func (d *digester) flushAndGetRate(target string) (sendCount int, sentAt time.Time) {
	var flushedTimes []time.Time
	times := d.pushedAt[target]
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
		d.pushedAt[target] = flushedTimes
	}
	if len(flushedTimes) > 0 {
		sentAt = flushedTimes[0]
	}
	return
}

func (d *digester) recordPush(target string) {
	times := d.pushedAt[target]
	times = append(times, time.Now())
	d.pushedAt[target] = times
}

func (d *digester) prepareToPush(q queue.Queue, qm queue.QueuedMessage, client PumpClient, smsg ServiceMessage) (digested bool) {
	d.cond.L.Lock()
	defer d.cond.L.Unlock()

	target := smsg.GetDigestTarget()
	sendCount, firstSendAt := d.flushAndGetRate(target)
	if sendCount < d.config.RateMax {
		d.recordPush(target)
		return false
	}
	d.adapter.Logger().Printf("Rate to %s exceeded, digested", target)

	batch, ok := d.batches[target]
	if ok {
		if batch.q != q {
			panic("digester cannot handle mixed queues")
		}
	} else {
		batch.q = q
		batch.client = client
	}
	batch.target = target
	batch.serviceMsgs = append(batch.serviceMsgs, smsg)
	batch.queuedMsgs = append(batch.queuedMsgs, qm)
	batch.due = firstSendAt.Add(d.config.RatePer)
	d.batches[target] = batch
	d.cond.Signal()
	return true
}

func (d *digester) getNextBatch() (b batch, stopped bool) {
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
		var minDueBatchTarget string
		for target, batch := range d.batches {
			if minDueBatch.due.IsZero() || minDueBatch.due.After(batch.due) {
				minDueBatch = batch
				minDueBatchTarget = target
			}
		}
		now := time.Now()
		if now.After(minDueBatch.due) {
			delete(d.batches, minDueBatchTarget)
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

func (d *digester) requestShutdown() {
	d.cond.L.Lock()
	d.shuttingDown = true
	d.cond.Signal()
	d.cond.L.Unlock()
}

func (d *digester) shutdown() {
	d.cond.L.Lock()
	defer d.cond.L.Unlock()

	d.adapter.Logger().Println("Shutting down digester:", len(d.batches), "batches unsent")
}

func (d *digester) serve(fc FeedbackCollector) {
	for {
		batch, stopped := d.getNextBatch()
		if stopped {
			d.shutdown()
			return
		}
		d.sendBatch(batch, fc)

	}
}

func (d *digester) sendBatch(b batch, fc FeedbackCollector) {
	d.adapter.Logger().Println("Sending digest")
	d.cond.L.Lock()
	d.recordPush(b.target)
	d.cond.L.Unlock()

	status := d.adapter.PushDigest(b.client, b.serviceMsgs, fc)
	switch status {
	case PushStatusTempFail:
		// TODO: We should actually attempt to retry this with a backoff
		fallthrough
	case PushStatusHardFail:
		d.adapter.Logger().Println("[ERROR] Failed to send digest")
		fallthrough
	case PushStatusSuccess:
		for _, qm := range b.queuedMsgs {
			removeFromQueue(b.q, qm, d.adapter.Logger())
		}
	}
}
