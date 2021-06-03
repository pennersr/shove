package fcm

import (
	"bytes"
	"context"
	"encoding/json"
	"gitlab.com/pennersr/shove/internal/queue"
	"gitlab.com/pennersr/shove/internal/services"
	"log"
	"net/http"
	"sync"
	"time"
)

// FCM ...
type FCM struct {
	wg     sync.WaitGroup
	apiKey string
	log    *log.Logger
	pump   *services.Pump
}

// NewFCM ...
func NewFCM(apiKey string, log *log.Logger) (fcm *FCM, err error) {
	fcm = &FCM{
		apiKey: apiKey,
		log:    log,
	}
	fcm.pump = services.NewPump(4, services.DigestConfig{}, fcm)
	return
}

func (fcm *FCM) Logger() *log.Logger {
	return fcm.log
}

// ID ...
func (fcm *FCM) ID() string {
	return "fcm"
}

// String ...
func (fcm *FCM) String() string {
	return "FCM"
}

func (fcm *FCM) NewClient() (services.PumpClient, error) {
	client := &http.Client{
		Timeout: time.Duration(15 * time.Second),
		Transport: &http.Transport{
			MaxIdleConns:    5,
			IdleConnTimeout: 30 * time.Second,
		},
	}
	return client, nil
}

type fcmResponse struct {
	Success int `json:"success"`
	Failure int `json:"failure"`
	Results []struct {
		MessageID      string `json:"message_id"`
		RegistrationID string `json:"registration_id"`
		Error          string `json:"error"`
	} `json:"results"`
}

func (fcm *FCM) PushDigest(services.PumpClient, []services.ServiceMessage, services.FeedbackCollector) services.PushStatus {
	panic("not implemented")
}

func (fcm *FCM) PushMessage(pclient services.PumpClient, smsg services.ServiceMessage, fc services.FeedbackCollector) services.PushStatus {
	msg := smsg.(fcmMessage)
	startedAt := time.Now()
	var success bool

	req, err := http.NewRequest("POST", "https://fcm.googleapis.com/fcm/send", bytes.NewBuffer(msg.rawData))
	if err != nil {
		fcm.log.Println("[ERROR] Creating request:", err)
		return services.PushStatusHardFail
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "key="+fcm.apiKey)

	client := pclient.(*http.Client)
	resp, err := client.Do(req)
	if err != nil {
		fcm.log.Println("[ERROR] Posting:", err)
		return services.PushStatusTempFail
	}
	duration := time.Now().Sub(startedAt)

	defer func() {
		fc.CountPush(fcm.ID(), success, duration)
	}()

	defer resp.Body.Close()
	if resp.StatusCode >= 400 && resp.StatusCode < 500 {
		fcm.log.Println("[ERROR] Rejected, status code:", resp.StatusCode)
		return services.PushStatusHardFail
	}
	if resp.StatusCode >= 500 && resp.StatusCode < 600 {
		fcm.log.Println("[ERROR] Upstream error, status code:", resp.StatusCode)
		return services.PushStatusTempFail
	}

	var fr fcmResponse
	err = json.NewDecoder(resp.Body).Decode(&fr)
	if err != nil {
		fcm.log.Println("[ERROR] Decoding response:", err)
		return services.PushStatusTempFail
	}
	regIDs := msg.RegistrationIDs
	if len(regIDs) == 0 {
		regIDs = append(regIDs, msg.To)
	}
	fcm.log.Println("Pushed, took", duration)
	for i, fb := range fr.Results {
		switch fb.Error {
		case "":
			// Noop
		case "InvalidRegistration":
			fallthrough
		case "NotRegistered":
			// you should remove the registration ID from your
			// server database because the application was
			// uninstalled from the device or it does not have a
			// broadcast receiver configured to receive
			// com.google.android.c2dm.intent.RECEIVE intents.
			fc.TokenInvalid(fcm.ID(), regIDs[i])
		case "Unavailable":
			// If it is Unavailable, you could retry to send it in
			// another request.
			fallthrough
		default:
			fcm.log.Println("[ERROR] Sending:", fb.Error)
		}
	}
	success = true
	return services.PushStatusSuccess
}

// Serve ...
func (fcm *FCM) Serve(ctx context.Context, q queue.Queue, fc services.FeedbackCollector) (err error) {
	return fcm.pump.Serve(ctx, q, fc)
}
