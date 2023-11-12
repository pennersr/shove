package telegram

import (
	"bytes"
	"encoding/json"
	"fmt"
	"golang.org/x/exp/slog"
	"net/http"
	"strings"
	"time"

	"gitlab.com/pennersr/shove/internal/services"
)

// TelegramService ...
type TelegramService struct {
	botToken string
	log      *slog.Logger
}

// NewTelegramService ...
func NewTelegramService(botToken string, log *slog.Logger) (tg *TelegramService, err error) {
	tg = &TelegramService{
		botToken: botToken,
		log:      log,
	}
	return
}

func (tg *TelegramService) Logger() *slog.Logger {
	return tg.log
}

// ID ...
func (tg *TelegramService) ID() string {
	return "telegram"

}

// String ...
func (tg *TelegramService) String() string {
	return "Telegram"
}

func (tg *TelegramService) NewClient() (services.PumpClient, error) {
	client := &http.Client{
		Timeout: time.Duration(15 * time.Second),
	}
	return client, nil
}

func (tg *TelegramService) SquashAndPushMessage(pclient services.PumpClient, smsgs []services.ServiceMessage, fc services.FeedbackCollector) (status services.PushStatus) {
	client := pclient.(*http.Client)
	msgs := make([]telegramMessage, len(smsgs))
	for i, smsg := range smsgs {
		msgs[i] = smsg.(telegramMessage)
	}
	dmsg, err := squashMessages(msgs)
	if err != nil {
		tg.log.Error("Squashing failed", "error", err)
		return services.PushStatusHardFail
	}
	return tg.pushMessage(client, dmsg.Method, dmsg.parsedPayload.ChatID, dmsg.Payload, fc)
}

func (tg *TelegramService) PushMessage(pclient services.PumpClient, smsg services.ServiceMessage, fc services.FeedbackCollector) (status services.PushStatus) {
	client := pclient.(*http.Client)
	msg := smsg.(telegramMessage)
	return tg.pushMessage(client, msg.Method, msg.parsedPayload.ChatID, msg.Payload, fc)
}

func (tg *TelegramService) pushMessage(client *http.Client, method string, chatID string, payload json.RawMessage, fc services.FeedbackCollector) (status services.PushStatus) {
	startedAt := time.Now()
	var success bool

	url := fmt.Sprintf("https://api.telegram.org/bot%s/%s", tg.botToken, method)
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(payload))
	if err != nil {
		tg.log.Error("Failure creating request", "error", err)
		return services.PushStatusHardFail
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		tg.log.Error("Posting failed", "error", err)
		return services.PushStatusTempFail
	}
	duration := time.Now().Sub(startedAt)

	defer func() {
		fc.CountPush(tg.ID(), success, duration)
	}()

	defer resp.Body.Close()

	if resp.StatusCode == 429 {
		tg.log.Error("Throttled, too many requests", "status", 429)
		return services.PushStatusTempFail
	}

	var respData struct {
		OK          bool   `json:"ok"`
		ErrorCode   int    `json:"error_code"`
		Description string `json:"description"`
	}

	err = json.NewDecoder(resp.Body).Decode(&respData)
	if err != nil {
		tg.log.Error("Unable to decode response", "error", err)
		return services.PushStatusTempFail
	}

	// It's a bit odd that an invalid chat ID results in a 400 instead of a
	// special response code {"ok":false,"error_code":400,"description":"Bad
	// Request: chat not found"}
	if respData.ErrorCode == 400 && strings.Contains(respData.Description, "chat not found") {
		fc.TokenInvalid(tg.ID(), chatID)
	}
	if resp.StatusCode >= 400 && resp.StatusCode < 500 {
		tg.log.Error("Rejected", "description", respData.Description, "error_code", respData.ErrorCode, "status", resp.StatusCode)
		return services.PushStatusHardFail
	}
	if resp.StatusCode >= 500 && resp.StatusCode < 600 {
		tg.log.Error("Upstream failure", "status", resp.StatusCode)
		return services.PushStatusTempFail
	}
	tg.log.Info("Pushed", "duration", duration)
	return services.PushStatusSuccess
}
