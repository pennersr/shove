package webpush

import (
	"fmt"
	"log"
	"os"
	"testing"
)

const subscription = `{
	"endpoint":"https://updates.push.services.mozilla.com/wpush/v2/gAAAAA",
	"keys": {
		"auth":"bHmp2U5UKnWaL-31nal7ew",
		"p256dh":"BKedT"
	}
}`

func TestConvert(t *testing.T) {
	wp, err := NewWebPush("pub", "pvt", log.New(os.Stderr, "", log.LstdFlags))
	if err != nil {
		t.Fatal(err)
	}
	smsg, err := wp.ConvertMessage([]byte(fmt.Sprintf(`
{
	"subscription": %s,
	"headers": {
		"ttl": 10,
		"urgency": "low"
	},
	"payload": {"xxx":"z"}
}
`, subscription)))
	if err != nil {
		t.Fatal(err)
	}
	msg := smsg.(webPushMessage)
	if msg.options.TTL != 10 {
		t.Fatal("TTL wrong")
	}
	if msg.Token != subscription {
		t.Fatal("Token not derived from subscription")
	}
}

func TestConvertWithToken(t *testing.T) {
	wp, err := NewWebPush("pub", "pvt", log.New(os.Stderr, "", log.LstdFlags))
	if err != nil {
		t.Fatal(err)
	}
	smsg, err := wp.ConvertMessage([]byte(fmt.Sprintf(`
{
	"subscription": %s,
	"token": "my-token",
	"headers": {
		"ttl": 10,
		"urgency": "low"
	},
	"payload": {"xxx":"z"}
}
`, subscription)))
	if err != nil {
		t.Fatal(err)
	}
	msg := smsg.(webPushMessage)
	if msg.Token != "my-token" {
		t.Fatal(msg.Token)
	}
}
