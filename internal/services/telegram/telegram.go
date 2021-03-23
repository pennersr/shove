package telegram

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"net/http"
	"strings"
	"sync"
	"time"

	"gitlab.com/pennersr/shove/internal/queue"
	"gitlab.com/pennersr/shove/internal/services"
)

// TelegramService ...
type TelegramService struct {
	botToken string
	wg       sync.WaitGroup
	log      *log.Logger
}

// NewTelegramService ...
func NewTelegramService(botToken string, log *log.Logger) (tg *TelegramService, err error) {
	tg = &TelegramService{
		botToken: botToken,
		log:      log,
	}
	return
}

// ID ...
func (tg *TelegramService) ID() string {
	return "telegram"

}

// String ...
func (tg *TelegramService) String() string {
	return "Telegram"
}

func (tg *TelegramService) serveClient(ctx context.Context, q queue.Queue, fc services.FeedbackCollector) {
	defer func() {
		tg.wg.Done()
	}()
	failureCount := 0
	for ctx.Err() == nil {
		qm, err := q.Get(ctx)
		if err != nil {
			tg.log.Println("[ERROR] Reading from queue:", err)
			return
		}
		msg := qm.Message()
		tgmsg, err := tg.convert(msg)
		if err != nil {
			tg.log.Println("[ERROR] Bad message:", err)
			tg.remove(q, qm)
			continue
		}
		success, retry := tg.push(tgmsg, msg, fc)
		if success || !retry {
			tg.remove(q, qm)
		} else {
			if err = q.Requeue(qm); err != nil {
				tg.log.Println("[ERROR] Putting back in the queue:", err)
			}
		}
		if retry {
			tg.backoff(ctx, failureCount)
			failureCount++

		} else {
			failureCount = 0
		}
	}
}

func (tg *TelegramService) push(msg *telegramMessage, data []byte, fc services.FeedbackCollector) (done, retry bool) {
	startedAt := time.Now()
	var success bool

	url := fmt.Sprintf("https://api.telegram.org/bot%s/%s", tg.botToken, msg.Method)
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(msg.Payload))
	if err != nil {
		tg.log.Println("[ERROR] Creating request:", err)
		return false, true
	}
	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{Timeout: time.Duration(15 * time.Second)}
	resp, err := client.Do(req)
	if err != nil {
		tg.log.Println("[ERROR] Posting:", err)
		return false, true
	}
	duration := time.Now().Sub(startedAt)

	defer func() {
		fc.CountPush(tg.ID(), success, duration)
	}()

	defer resp.Body.Close()

	if resp.StatusCode == 429 {
		tg.log.Println("[ERROR] Throttled, too many requests: 429")
		return false, true
	}

	var respData struct {
		OK          bool   `json:"ok"`
		ErrorCode   int    `json:"error_code"`
		Description string `json:"description"`
	}

	err = json.NewDecoder(resp.Body).Decode(&respData)
	if err != nil {
		tg.log.Println("[ERROR] Decoding response:", err)
		return false, true
	}

	// It's a bit odd that an invalid chat ID results in a 400 instead of a
	// special response code {"ok":false,"error_code":400,"description":"Bad
	// Request: chat not found"}
	if respData.ErrorCode == 400 && strings.Contains(respData.Description, "chat not found") {
		fc.TokenInvalid(tg.ID(), msg.ChatID)
	}
	if resp.StatusCode >= 400 && resp.StatusCode < 500 {
		tg.log.Printf("[ERROR] Rejected: %s (%d), HTTP status: %d", respData.Description, respData.ErrorCode, resp.StatusCode)
		return true, false
	}
	if resp.StatusCode >= 500 && resp.StatusCode < 600 {
		tg.log.Println("[ERROR] Upstream HTTP status:", resp.StatusCode)
		return false, true
	}
	tg.log.Println("Pushed, took", duration)
	success = true
	return true, false
}

func (tg *TelegramService) remove(q queue.Queue, qm queue.QueuedMessage) {
	if err := q.Remove(qm); err != nil {
		tg.log.Println("[ERROR] Removing from the queue:", err)
	}
}

func (tg *TelegramService) backoff(ctx context.Context, failureCount int) {
	sleep := time.Duration(float64(time.Second) * math.Min(30, math.Pow(2., float64(failureCount))))
	tg.log.Printf("Backing off for %s", sleep)
	ctx, cancel := context.WithTimeout(ctx, sleep)
	defer cancel()
	<-ctx.Done()
}

// Serve ...
func (tg *TelegramService) Serve(ctx context.Context, q queue.Queue, fc services.FeedbackCollector) (err error) {
	for i := 0; i < 2; i++ {
		go tg.serveClient(ctx, q, fc)
		tg.wg.Add(1)
	}
	tg.log.Println("Workers started")
	tg.wg.Wait()
	tg.log.Println("Workers stopped")

	return
}
