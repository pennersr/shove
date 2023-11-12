package webhook

import (
	"bytes"
	"io/ioutil"
	"net/http"
	"time"

	"gitlab.com/pennersr/shove/internal/services"
	"golang.org/x/exp/slog"
)

type Webhook struct {
	log *slog.Logger
}

func NewWebhook(log *slog.Logger) (fcm *Webhook, err error) {
	fcm = &Webhook{
		log: log,
	}
	return
}

func (fcm *Webhook) Logger() *slog.Logger {
	return fcm.log
}

// ID ...
func (fcm *Webhook) ID() string {
	return "webhook"
}

// String ...
func (fcm *Webhook) String() string {
	return "Webhook"
}

func (fcm *Webhook) NewClient() (services.PumpClient, error) {
	client := &http.Client{
		Timeout: time.Duration(5 * time.Second),
		Transport: &http.Transport{
			MaxIdleConns:    5,
			IdleConnTimeout: 30 * time.Second,
		},
	}
	return client, nil
}

func (wh *Webhook) SquashAndPushMessage(services.PumpClient, []services.ServiceMessage, services.FeedbackCollector) services.PushStatus {
	panic("not implemented")
}

func (wh *Webhook) PushMessage(pclient services.PumpClient, smsg services.ServiceMessage, fc services.FeedbackCollector) services.PushStatus {
	msg := smsg.(webhookMessage)
	startedAt := time.Now()
	var success bool

	wh.log.Debug("POST", "url", msg.URL, "data", string(msg.postData))
	req, err := http.NewRequest("POST", msg.URL, bytes.NewBuffer(msg.postData))
	if err != nil {
		wh.log.Error("Failed to create request", "error", err)
		return services.PushStatusHardFail
	}
	for k, v := range msg.Headers {
		req.Header.Set(k, v)
	}

	client := pclient.(*http.Client)
	resp, err := client.Do(req)
	if err != nil {
		wh.log.Error("Failed to post", "error", err)
		return services.PushStatusHardFail
	}
	duration := time.Now().Sub(startedAt)

	defer func() {
		fc.CountPush(wh.ID(), success, duration)
	}()

	body, error := ioutil.ReadAll(resp.Body)
	if error != nil {
		wh.log.Error("Failed to read POST response", "error", error)
	} else {
		wh.log.Debug("POST response", "response", string(body))
	}

	defer resp.Body.Close()
	if resp.StatusCode >= 400 && resp.StatusCode < 500 {
		wh.log.Error("Rejected", "status", resp.StatusCode)
		return services.PushStatusHardFail
	}
	if resp.StatusCode >= 500 && resp.StatusCode < 600 {
		wh.log.Error("Upstream failure", "status", resp.StatusCode)
		// A retry might help, but currently retries are not limitted to
		// a certain number of attempts, meaning, we would keep trying
		// indefinitely.
		return services.PushStatusHardFail
	}
	success = true
	return services.PushStatusSuccess
}
