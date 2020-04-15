package redis

import (
	"context"
	"time"

	"github.com/gomodule/redigo/redis"
	"gitlab.com/pennersr/redq"
	"gitlab.com/pennersr/shove/internal/queue"
)

type redisQueueFactory struct {
	pool *redis.Pool
}

type redisQueue struct {
	q *redq.RedQueue
}

// NewQueueFactory ...
func NewQueueFactory(url string) queue.QueueFactory {
	qf := &redisQueueFactory{
		pool: &redis.Pool{
			MaxIdle:     3,
			IdleTimeout: 240 * time.Second,
			Dial: func() (redis.Conn, error) {
				return redis.DialURL(url)
			},
		},
	}
	return qf
}

func (rq redisQueue) Queue(msg []byte) (err error) {
	return rq.q.Queue(msg)
}

func (rq redisQueue) Get(ctx context.Context) (qm queue.QueuedMessage, err error) {
	qm, err = rq.q.Get(ctx)
	return
}

func (rq redisQueue) Remove(qm queue.QueuedMessage) (err error) {
	return rq.q.Remove(qm.(redq.QueuedMessage))
}

func (rq redisQueue) Requeue(qm queue.QueuedMessage) (err error) {
	return rq.q.Requeue(qm.(redq.QueuedMessage))
}

func (rq redisQueue) Shutdown() (err error) {
	return rq.q.Close()
}

func (rqf *redisQueueFactory) NewQueue(id string) (q queue.Queue, err error) {
	waitingList := ListName(id)
	rq, err := redq.NewQueue(rqf.pool, waitingList)
	if err != nil {
		return
	}
	q = redisQueue{q: rq}
	return
}

// ListName returns the Redis list name used for queueing.
func ListName(serviceID string) string {
	return "shove:" + serviceID
}
