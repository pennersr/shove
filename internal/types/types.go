package types

import (
	"encoding/json"
)

type PushMessage struct {
	Tokens  []string                   `json:"tokens"`
	Headers map[string]json.RawMessage `json:"headers,omitempty"`
	Payload json.RawMessage            `json:"payload,omitempty"`
}
