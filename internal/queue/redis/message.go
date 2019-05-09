package redis

import (
	"gitlab.com/pennersr/shove/internal/types"
)

type redisQueuedMessage struct {
	raw []byte
	msg types.PushMessage
}

func (qm *redisQueuedMessage) Message() types.PushMessage {
	return qm.msg
}
