package queue

import (
	"context"
)

// Queue ...
type Queue interface {
	Queue([]byte) error
	Get(ctx context.Context) (QueuedMessage, error)
	Remove(QueuedMessage) error
	Requeue(QueuedMessage) error
	Shutdown() error
}

// QueuedMessage ...
type QueuedMessage interface {
	Message() []byte
}

// QueueFactory ...
type QueueFactory interface {
	NewQueue(id string) (Queue, error)
}
