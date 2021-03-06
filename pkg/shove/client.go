package shove

import (
	"github.com/gomodule/redigo/redis"
	shvredis "gitlab.com/pennersr/shove/internal/queue/redis"
	"time"
)

// Client ...
type Client interface {
	PushRaw(serviceID string, data []byte) (err error)
}

type redisClient struct {
	pool *redis.Pool
}

// NewRedisClient ...
func NewRedisClient(redisURL string) Client {
	rc := &redisClient{
		pool: &redis.Pool{
			MaxIdle:     3,
			IdleTimeout: 240 * time.Second,
			Dial: func() (redis.Conn, error) {
				return redis.DialURL(redisURL)
			},
		},
	}
	return rc
}

// PushRaw ...
func (rc *redisClient) PushRaw(id string, data []byte) (err error) {
	waitingList := shvredis.ListName(id)
	conn := rc.pool.Get()
	defer conn.Close()
	_, err = conn.Do("RPUSH", waitingList, data)
	return
}
