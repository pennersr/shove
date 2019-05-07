package apns

import (
	"encoding/json"
	"errors"
	"github.com/sideshow/apns2"
	"gitlab.com/pennersr/shove/internal/types"
	"time"
)

func (apns *APNS) convert(pm types.PushMessage) (notif *apns2.Notification, err error) {
	if len(pm.Tokens) != 1 {
		err = errors.New("APNS expects exactly one token")
		return
	}
	notif = new(apns2.Notification)
	notif.DeviceToken = pm.Tokens[0]
	topic, ok := pm.Headers["apns-topic"]
	if !ok {
		err = errors.New("APNS requires a topic")
		return
	}
	err = json.Unmarshal(topic, &notif.Topic)
	if err != nil {
		return
	}
	prio, ok := pm.Headers["apns-priority"]
	if ok {
		err = json.Unmarshal(prio, &notif.Priority)
		if err != nil {
			return
		}
	}
	collapse, ok := pm.Headers["apns-collapse-id"]
	if ok {
		err = json.Unmarshal(collapse, &notif.CollapseID)
		if err != nil {
			return
		}
	}
	exp, ok := pm.Headers["apns-expiration"]
	if ok {
		var epoch int64
		err = json.Unmarshal(exp, &epoch)
		if err != nil {
			return
		}
		notif.Expiration = time.Unix(epoch, 0)
	}
	notif.Payload = pm.Payload
	return
}

func (apns *APNS) Validate(pm types.PushMessage) (err error) {
	_, err = apns.convert(pm)
	return
}
