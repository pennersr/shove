package services

import (
	"context"
	"fmt"
	"gitlab.com/pennersr/shove/internal/queue"
	"gitlab.com/pennersr/shove/internal/types"
)

type FeedbackCollector interface {
	TokenInvalid(serviceID, token string)
	ReplaceToken(serviceID, token, replacement string)
}

type PushService interface {
	fmt.Stringer
	ID() string
	Serve(ctx context.Context, q queue.Queue, fc FeedbackCollector) error
	Validate(types.PushMessage) error
}
