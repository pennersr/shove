package queue

import (
	"context"
	"gitlab.com/pennersr/shove/internal/types"
)

type Queue interface {
	Queue(types.PushMessage) error
	Get(ctx context.Context) (QueuedMessage, error)
	Remove(QueuedMessage) error
	Requeue(QueuedMessage) error
	Shutdown() error
}

type QueuedMessage interface {
	Message() types.PushMessage
}

type QueueFactory interface {
	NewQueue(id string) (Queue, error)
}
