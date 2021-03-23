package email

import (
	"gitlab.com/pennersr/shove/internal/queue"
	"sync"
	"time"
)

type batch struct {
	key      string
	emails   []email
	due      time.Time
	messages []queue.QueuedMessage
	q        queue.Queue
}

type digester struct {
	mailedAt     map[string][]time.Time
	batches      map[string]batch
	config       EmailConfig
	cond         *sync.Cond
	lock         sync.Mutex
	shuttingDown bool
}

func (d *digester) init(config EmailConfig) {
	d.config = config
	d.mailedAt = make(map[string][]time.Time)
	d.batches = make(map[string]batch)
	d.cond = sync.NewCond(&d.lock)
}

func (d *digester) flushAndGetRate(key string) (sendCount int, sentAt time.Time) {
	var flushedTimes []time.Time
	times := d.mailedAt[key]
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
		d.mailedAt[key] = flushedTimes
	}
	if len(flushedTimes) > 0 {
		sentAt = flushedTimes[0]
	}
	return
}

func (d *digester) recordSend(key string) {
	times := d.mailedAt[key]
	times = append(times, time.Now())
	d.mailedAt[key] = times
}

func (d *digester) prepareToMail(q queue.Queue, qm queue.QueuedMessage, email email) (digested bool) {
	d.cond.L.Lock()
	defer d.cond.L.Unlock()

	key := email.To[0]
	sendCount, firstSendAt := d.flushAndGetRate(key)
	if sendCount < d.config.RateMax {
		d.recordSend(key)
		return false
	}
	d.config.Log.Printf("Rate to %s exceeded, email digested", email.To[0])

	batch, ok := d.batches[key]
	if ok {
		if batch.q != q {
			panic("digester cannot handle mixed queues")
		}
	} else {
		batch.q = q
	}
	batch.key = key
	batch.emails = append(batch.emails, email)
	batch.messages = append(batch.messages, qm)
	batch.due = firstSendAt.Add(d.config.RatePer)
	d.batches[key] = batch
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

func (d *digester) requestShutdown() {
	d.cond.L.Lock()
	d.shuttingDown = true
	d.cond.Signal()
	d.cond.L.Unlock()
}

func (d *digester) shutdown() {
	d.cond.L.Lock()
	defer d.cond.L.Unlock()

	d.config.Log.Println("Shutting down email digester:", len(d.batches), "batches unsent")
	for _, batch := range d.batches {
		for _, qm := range batch.messages {
			batch.q.Queue(qm.Message())
		}
	}
}

func (d *digester) serve() {
	for {
		batch, stopped := d.getNextBatch()
		if stopped {
			d.shutdown()
			return
		}
		d.sendBatch(batch)

	}
}

func (d *digester) sendBatch(b batch) {
	d.config.Log.Println("Sending digest email")
	body, err := encodeEmailDigest(b.emails)
	if err != nil {
		d.config.Log.Println("[ERROR] Failed to encode email digest:", err)
		return
	}
	d.cond.L.Lock()
	d.recordSend(b.key)
	d.cond.L.Unlock()

	err = d.config.send(b.emails[0].From, b.emails[0].To, body)
	if err != nil {
		d.config.Log.Println("[ERROR] Cannot send digest email:", err)
		return
	}
}
