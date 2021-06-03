package apns

import (
	"crypto/tls"
	"github.com/sideshow/apns2"
	"github.com/sideshow/apns2/certificate"
	"gitlab.com/pennersr/shove/internal/services"
	"log"
	"time"
)

// APNS ...
type APNS struct {
	production bool
	log        *log.Logger
	cert       tls.Certificate
}

// NewAPNS ...
func NewAPNS(pemFile string, production bool, log *log.Logger) (apns *APNS, err error) {
	cert, err := certificate.FromPemFile(pemFile, "")
	if err != nil {
		return
	}
	apns = &APNS{
		cert:       cert,
		production: production,
		log:        log,
	}
	return
}

func (apns *APNS) Logger() *log.Logger {
	return apns.log
}

func (apns *APNS) NewClient() (pclient services.PumpClient, err error) {
	client := apns2.NewClient(apns.cert)
	if apns.production {
		client.Production()
	} else {
		client.Development()
	}
	pclient = client
	return
}

// ID ...
func (apns *APNS) ID() string {
	if apns.production {
		return "apns"
	}
	return "apns-sandbox"

}

// String ...
func (apns *APNS) String() string {
	if apns.production {
		return "APNS"
	}
	return "APNS-sandbox"
}

func (apns *APNS) SquashAndPushMessage(client services.PumpClient, smsgs []services.ServiceMessage, fc services.FeedbackCollector) services.PushStatus {
	panic("not implemented")
}

func (apns *APNS) PushMessage(pclient services.PumpClient, smsg services.ServiceMessage, fc services.FeedbackCollector) (status services.PushStatus) {
	client := pclient.(*apns2.Client)
	notif := smsg.(apnsNotification)
	t := time.Now()
	resp, err := client.Push(notif.notification)
	duration := time.Now().Sub(t)
	sent := false
	if err != nil {
		apns.log.Println("[ERROR] Pushing:", err)
		status = services.PushStatusTempFail
	} else {
		reason := resp.Reason
		if reason == "" {
			reason = "OK"
		}
		apns.log.Printf("Pushed (%s), took %s", reason, duration)
		sent = resp.Sent()
		if resp.Reason == apns2.ReasonBadDeviceToken || resp.Reason == apns2.ReasonUnregistered {
			fc.TokenInvalid(apns.ID(), notif.notification.DeviceToken)
		}
		retry := resp.StatusCode >= 500
		if sent {
			status = services.PushStatusSuccess
		} else if retry {
			status = services.PushStatusTempFail
		} else {
			status = services.PushStatusHardFail
		}
	}
	fc.CountPush(apns.ID(), sent, duration)
	return
}
