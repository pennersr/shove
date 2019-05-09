package redis

type redisQueuedMessage []byte

func (qm redisQueuedMessage) Message() []byte {
	return []byte(qm)
}
