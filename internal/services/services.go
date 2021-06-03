package services

import (
	"fmt"
	"time"
)

// FeedbackCollector ...
type FeedbackCollector interface {
	TokenInvalid(serviceID, token string)
	ReplaceToken(serviceID, token, replacement string)
	CountPush(serviceID string, success bool, duration time.Duration)
}

// PushService ...
type PushService interface {
	PumpAdapter
	fmt.Stringer
	ID() string
	Validate([]byte) error
}
