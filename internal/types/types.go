package types

import (
	"encoding/json"
)

type PushMessage struct {
	Tokens  []string
	Topic   string
	Payload json.RawMessage
}
