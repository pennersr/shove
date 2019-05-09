package memory

type memoryQueuedMessage struct {
	msg     []byte
	pending bool
	idx     int
}

func (qm *memoryQueuedMessage) Message() []byte {
	return qm.msg
}
