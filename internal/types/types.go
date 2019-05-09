package types

import (
	"encoding/json"
)

type PushMessage struct {
	Tokens  []string                   `json:"tokens"`
	Headers map[string]json.RawMessage `json:"headers,omitempty"`
	Payload json.RawMessage            `json:"payload,omitempty"`
}

func (pm PushMessage) Marshal() (b []byte, err error) {
	return json.Marshal(pm)
}

func UnmarshalPushMessage(raw []byte) (pm PushMessage, err error) {
	err = json.Unmarshal(raw, &pm)
	return
}
