package queue

import (
	"context"
)

type Queue interface {
	Queue([]byte) error
	Get(ctx context.Context) (QueuedMessage, error)
	Remove(QueuedMessage) error
	Requeue(QueuedMessage) error
	Shutdown() error
}

type QueuedMessage interface {
	Message() []byte
}

type QueueFactory interface {
	NewQueue(id string) (Queue, error)
}
