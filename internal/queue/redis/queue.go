package redis

import (
	"context"
	"errors"
	"github.com/gomodule/redigo/redis"
	"gitlab.com/pennersr/shove/internal/queue"
	"log"
	"time"
)

type redisQueueFactory struct {
	pool *redis.Pool
}

type redisQueue struct {
	id           string
	pool         *redis.Pool
	shuttingDown bool
	waitingList  string
	pendingList  string
}

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

func (rq *redisQueue) Queue(msg []byte) (err error) {
	conn := rq.pool.Get()
	defer conn.Close()
	_, err = conn.Do("RPUSH", rq.waitingList, msg)
	return nil
}

func (rq *redisQueue) Shutdown() (err error) {
	rq.shuttingDown = true
	err = rq.pool.Close()
	return
}

func (rq *redisQueue) Remove(qm queue.QueuedMessage) (err error) {
	conn := rq.pool.Get()
	defer conn.Close()
	n, err := redis.Int(conn.Do("LREM", rq.pendingList, 1, qm.Message()))
	if err != nil {
		return
	}
	if n == 0 {
		log.Println("Push message already gone from pending list", rq.pendingList)
	}
	return nil
}

func (rq *redisQueue) Requeue(qm queue.QueuedMessage) (err error) {
	conn := rq.pool.Get()
	defer conn.Close()
	if err = conn.Send("MULTI"); err != nil {
		return
	}
	if err = conn.Send("LREM", rq.pendingList, 1, qm.Message()); err != nil {
		return
	}
	if err = conn.Send("RPUSH", rq.waitingList, qm.Message()); err != nil {
		return
	}
	_, err = conn.Do("EXEC")
	return
}

func (rq *redisQueue) Get(ctx context.Context) (qm queue.QueuedMessage, err error) {
	conn := rq.pool.Get()
	defer conn.Close()

	var raw []byte
	for ctx.Err() == nil {
		raw, err = redis.Bytes(conn.Do("BRPOPLPUSH", rq.waitingList, rq.pendingList, 2))
		if err == redis.ErrNil {
			err = nil
			continue
		}
		if err != nil {
			return
		}
		qm = redisQueuedMessage(raw)
		return
	}
	err = errors.New("queue shutting down")
	return
}

func (rq *redisQueue) recover() (err error) {
	conn := rq.pool.Get()
	defer conn.Close()
	for {
		_, err := redis.Bytes(conn.Do("RPOPLPUSH", rq.pendingList, rq.waitingList))
		if err == redis.ErrNil {
			log.Println("No more", rq.id, "push notifications to recover")
			break
		}
		if err != nil {
			return err
		}
		log.Println("recovered pending", rq.id, "push notification")
	}
	return
}

func (rqf *redisQueueFactory) NewQueue(id string) (q queue.Queue, err error) {
	rq := &redisQueue{
		id:   id,
		pool: rqf.pool,
	}
	rq.waitingList, rq.pendingList = ListNames(id)
	err = rq.recover()
	if err != nil {
		return
	}
	q = rq
	return
}
