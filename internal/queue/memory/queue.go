package memory

import (
	"context"
	"errors"
	"gitlab.com/pennersr/shove/internal/queue"
	"sync"
)

// MemoryQueueFactory ...
type MemoryQueueFactory struct{}

type memoryQueue struct {
	buf          []*memoryQueuedMessage
	lock         sync.Mutex
	cond         *sync.Cond
	shuttingDown bool
}

func (mq *memoryQueue) Queue(msg []byte) (err error) {
	mq.lock.Lock()
	qm := &memoryQueuedMessage{
		msg: msg,
		idx: -1,
	}
	for i := 0; i < len(mq.buf); i++ {
		if mq.buf[i] == nil {
			qm.idx = i
			mq.buf[i] = qm
			break
		}
	}
	if qm.idx < 0 {
		qm.idx = len(mq.buf)
		mq.buf = append(mq.buf, qm)
	}
	mq.lock.Unlock()
	mq.cond.Signal()
	return nil
}

func (mq *memoryQueue) Shutdown() (err error) {
	mq.shuttingDown = true
	mq.cond.Broadcast()
	return
}

func (mq *memoryQueue) Remove(qm queue.QueuedMessage) (err error) {
	mq.lock.Lock()
	mqm := qm.(*memoryQueuedMessage)
	mq.buf[mqm.idx] = nil
	mq.lock.Unlock()
	return nil
}

func (mq *memoryQueue) Requeue(qm queue.QueuedMessage) (err error) {
	mq.lock.Lock()
	mqm := qm.(*memoryQueuedMessage)
	mqm.pending = false
	mq.lock.Unlock()
	mq.cond.Signal()
	return
}

func (mq *memoryQueue) Get(ctx context.Context) (queue.QueuedMessage, error) {
	mq.cond.L.Lock()
	defer mq.cond.L.Unlock()
	for ctx.Err() == nil && !mq.shuttingDown {
		mq.cond.Wait()
		for i := 0; i < len(mq.buf); i++ {
			m := mq.buf[i]
			if m != nil && !m.pending {
				m.pending = true
				return m, nil
			}
		}
	}
	return nil, errors.New("queue shut down")
}

// NewQueue ...
func (mqf MemoryQueueFactory) NewQueue(id string) (q queue.Queue, err error) {
	mq := &memoryQueue{}
	mq.cond = sync.NewCond(&mq.lock)
	q = mq
	return
}
