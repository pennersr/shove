package memory

import (
	"gitlab.com/pennersr/shove/internal/types"
)

type memoryQueuedMessage struct {
	msg     types.PushMessage
	pending bool
	idx     int
}

func (qm *memoryQueuedMessage) Message() types.PushMessage {
	return qm.msg
}
