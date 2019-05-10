package services

import (
	"context"
	"fmt"
	"gitlab.com/pennersr/shove/internal/queue"
)

// FeedbackCollector ...
type FeedbackCollector interface {
	TokenInvalid(serviceID, token string)
	ReplaceToken(serviceID, token, replacement string)
}

// PushService ...
type PushService interface {
	fmt.Stringer
	ID() string
	Serve(ctx context.Context, q queue.Queue, fc FeedbackCollector) error
	Validate([]byte) error
}
