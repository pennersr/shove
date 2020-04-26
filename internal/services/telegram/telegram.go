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
}

// NewTelegramService ...
func NewTelegramService(botToken string) (tg *TelegramService, err error) {
	tg = &TelegramService{botToken: botToken}
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
			log.Println(tg, "error reading from queue:", err)
			return
		}
		msg := qm.Message()
		tgmsg, err := tg.convert(msg)
		if err != nil {
			log.Println(tg, "bad message:", err)
			tg.remove(q, qm)
			continue
		}
		success, retry := tg.push(tgmsg, msg, fc)
		if success || !retry {
			tg.remove(q, qm)
		} else {
			if err = q.Requeue(qm); err != nil {
				log.Println(tg, "error putting back in the queue:", err)
			}
		}
		if retry {
			backoff(ctx, failureCount)
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
		log.Println(tg, "error creating request:", err)
		return false, true
	}
	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{Timeout: time.Duration(15 * time.Second)}
	resp, err := client.Do(req)
	if err != nil {
		log.Println(tg, "error posting:", err)
		return false, true
	}
	duration := time.Now().Sub(startedAt)

	defer func() {
		fc.CountPush(tg.ID(), success, duration)
	}()

	defer resp.Body.Close()

	if resp.StatusCode == 429 {
		log.Println(tg, "throttled, too many requests: 429")
		return false, true
	}

	var respData struct {
		OK          bool   `json:"ok"`
		ErrorCode   int    `json:"error_code"`
		Description string `json:"description"`
	}

	err = json.NewDecoder(resp.Body).Decode(&respData)
	if err != nil {
		log.Println(tg, "error decoding response:", err)
		return false, true
	}

	// It's a bit odd that an invalid chat ID results in a 400 instead of a
	// special response code {"ok":false,"error_code":400,"description":"Bad
	// Request: chat not found"}
	if respData.ErrorCode == 400 && strings.Contains(respData.Description, "chat not found") {
		fc.TokenInvalid(tg.ID(), msg.ChatID)
	}
	if resp.StatusCode >= 400 && resp.StatusCode < 500 {
		log.Println(tg, "rejected, status code:", resp.StatusCode)
		return true, false
	}
	if resp.StatusCode >= 500 && resp.StatusCode < 600 {
		log.Println(tg, "upstream error, status code:", resp.StatusCode)
		return false, true
	}
	log.Println(tg, "pushed, took", duration)
	success = true
	return true, false
}

func (tg *TelegramService) remove(q queue.Queue, qm queue.QueuedMessage) {
	if err := q.Remove(qm); err != nil {
		log.Println(tg, "error removing from the queue:", err)
	}
}

func backoff(ctx context.Context, failureCount int) {
	sleep := time.Duration(float64(time.Second) * math.Min(30, math.Pow(2., float64(failureCount))))
	log.Printf("Backing off for %s", sleep)
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
	log.Println(tg, "workers started")
	tg.wg.Wait()
	log.Println(tg, "workers stopped")

	return
}
