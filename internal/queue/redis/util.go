package redis

// ListNames returns the Redis list names used for queueing.
func ListNames(serviceID string) (l, pl string) {
	l = "shove:" + serviceID
	pl = l + ":pending"
	return
}
