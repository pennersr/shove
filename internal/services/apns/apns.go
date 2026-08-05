package apns

import (
	"codeberg.org/pennersr/shove/internal/services"
	"crypto/tls"
	"fmt"
	"github.com/sideshow/apns2"
	"github.com/sideshow/apns2/certificate"
	"github.com/sideshow/apns2/token"
	"golang.org/x/exp/slog"
	"time"
)

// APNS ...
type APNS struct {
	production bool
	log        *slog.Logger
	cert       *tls.Certificate
	token      *token.Token
}

// NewAPNS creates a certificate-based (.pem) APNS service.
func NewAPNS(pemFile string, production bool, log *slog.Logger) (apns *APNS, err error) {
	cert, err := certificate.FromPemFile(pemFile, "")
	if err != nil {
		return
	}
	apns = &APNS{
		cert:       &cert,
		production: production,
		log:        log,
	}
	return
}

// NewAPNSToken creates a token-based (.p8 auth key) APNS service. A single
// auth key is valid for both production and sandbox and does not expire.
func NewAPNSToken(authKeyPath, keyID, teamID string, production bool, log *slog.Logger) (apns *APNS, err error) {
	authKey, err := token.AuthKeyFromFile(authKeyPath)
	if err != nil {
		return nil, fmt.Errorf("APNS auth key: %w", err)
	}
	apns = &APNS{
		token: &token.Token{
			AuthKey: authKey,
			KeyID:   keyID,
			TeamID:  teamID,
		},
		production: production,
		log:        log,
	}
	return
}

func (apns *APNS) Logger() *slog.Logger {
	return apns.log
}

func (apns *APNS) NewClient() (pclient services.PumpClient, err error) {
	var client *apns2.Client
	if apns.token != nil {
		client = apns2.NewTokenClient(apns.token)
	} else {
		client = apns2.NewClient(*apns.cert)
	}
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
		apns.log.Error("Push message failed", "error", err)
		status = services.PushStatusTempFail
	} else {
		reason := resp.Reason
		if reason == "" {
			reason = "OK"
		}
		apns.log.Info("Pushed", "reason", reason, "duration", duration)
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
