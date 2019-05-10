package fcm

import (
	"bytes"
	"context"
	"encoding/json"
	"gitlab.com/pennersr/shove/internal/queue"
	"gitlab.com/pennersr/shove/internal/services"
	"log"
	"math"
	"net/http"
	"sync"
	"time"
)

// FCM ...
type FCM struct {
	transport *http.Transport
	wg        sync.WaitGroup
	apiKey    string
}

// NewFCM ...
func NewFCM(apiKey string) (fcm *FCM, err error) {
	fcm = &FCM{
		apiKey: apiKey,
		transport: &http.Transport{
			MaxIdleConns:    5,
			IdleConnTimeout: 30 * time.Second,
		},
	}
	return
}

// ID ...
func (fcm *FCM) ID() string {
	return "fcm"
}

// String ...
func (fcm *FCM) String() string {
	return "FCM"
}

func (fcm *FCM) serveClient(ctx context.Context, q queue.Queue, fc services.FeedbackCollector) {
	defer func() {
		fcm.wg.Done()
	}()
	failureCount := 0
	for ctx.Err() == nil {
		qm, err := q.Get(ctx)
		if err != nil {
			log.Println(fcm, "error reading from queue:", err)
			return
		}
		msg := qm.Message()
		notif, err := fcm.convert(msg)
		if err != nil {
			log.Println(fcm, "bad message:", err)
			fcm.remove(q, qm)
			continue
		}
		done, retry := fcm.push(notif, msg, fc)
		if done {
			fcm.remove(q, qm)
		} else {
			if err = q.Requeue(qm); err != nil {
				log.Println(fcm, "error putting back in the queue:", err)
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

type fcmResponse struct {
	Success int `json:"success"`
	Failure int `json:"failure"`
	Results []struct {
		MessageID      string `json:"message_id"`
		RegistrationID string `json:"registration_id"`
		Error          string `json:"error"`
	} `json:"results"`
}

func (fcm *FCM) push(msg *fcmMessage, data []byte, fc services.FeedbackCollector) (done, retry bool) {
	req, err := http.NewRequest("POST", "https://fcm.googleapis.com/fcm/send", bytes.NewBuffer(data))
	if err != nil {
		log.Println(fcm, "error creating request:", err)
		return false, true
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "key="+fcm.apiKey)

	client := &http.Client{
		Timeout:   time.Duration(15 * time.Second),
		Transport: fcm.transport}
	t := time.Now()
	resp, err := client.Do(req)
	if err != nil {
		log.Println(fcm, "error posting:", err)
		return false, true
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 && resp.StatusCode < 500 {
		log.Println(fcm, "rejected, status code:", resp.StatusCode)
		return true, false
	}
	if resp.StatusCode >= 500 && resp.StatusCode < 600 {
		log.Println(fcm, "upstream error, status code:", resp.StatusCode)
		return false, true
	}

	var fr fcmResponse
	err = json.NewDecoder(resp.Body).Decode(&fr)
	if err != nil {
		log.Println(fcm, "error decoding response:", err)
		return false, true
	}
	regIDs := msg.RegistrationIDs
	if len(regIDs) == 0 {
		regIDs = append(regIDs, msg.To)
	}
	log.Println(fcm, "pushed, took", time.Now().Sub(t))
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
			log.Println(fcm, "error sending:", fb.Error)
		}
	}
	return true, false
}

func backoff(ctx context.Context, failureCount int) {
	sleep := time.Duration(float64(time.Second) * math.Min(30, math.Pow(2., float64(failureCount))))
	log.Printf("Backing off for %s", sleep)
	ctx, cancel := context.WithTimeout(ctx, sleep)
	defer cancel()
	<-ctx.Done()
}

func (fcm *FCM) remove(q queue.Queue, qm queue.QueuedMessage) {
	if err := q.Remove(qm); err != nil {
		log.Println(fcm, "error removing from the queue:", err)
	}
}

// Serve ...
func (fcm *FCM) Serve(ctx context.Context, q queue.Queue, fc services.FeedbackCollector) (err error) {
	for i := 0; i < 4; i++ {
		go fcm.serveClient(ctx, q, fc)
		fcm.wg.Add(1)
	}
	log.Println(fcm, "workers started")
	fcm.wg.Wait()
	log.Println(fcm, "workers stopped")

	return
}
