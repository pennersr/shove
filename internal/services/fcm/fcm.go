package fcm

import (
	"context"
	firebase "firebase.google.com/go"
	"firebase.google.com/go/messaging"
	"gitlab.com/pennersr/shove/internal/services"
	"golang.org/x/exp/slog"
	"google.golang.org/api/option"
	"strings"
	"time"
)

// FCM ...
type FCM struct {
	credentialsFile string
	log             *slog.Logger
}

// NewFCM ...
func NewFCM(credentialsFile string, log *slog.Logger) (fcm *FCM, err error) {
	fcm = &FCM{
		credentialsFile: credentialsFile,
		log:             log,
	}
	return
}

func (fcm *FCM) Logger() *slog.Logger {
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
	opt := option.WithCredentialsFile(fcm.credentialsFile)
	ctx := context.Background()
	app, err := firebase.NewApp(ctx, nil, opt)
	if err != nil {
		return nil, err
	}
	client, err := app.Messaging(ctx)
	if err != nil {
		return nil, err
	}
	return client, nil
}

func (fcm *FCM) SquashAndPushMessage(services.PumpClient, []services.ServiceMessage, services.FeedbackCollector) services.PushStatus {
	panic("not implemented")
}

func (fcm *FCM) PushMessage(pclient services.PumpClient, smsg services.ServiceMessage, fc services.FeedbackCollector) services.PushStatus {
	msg := smsg.(fcmMessage)
	startedAt := time.Now()
	var success bool

	client := pclient.(*messaging.Client)
	_, err := client.Send(context.Background(), msg.Message)
	duration := time.Now().Sub(startedAt)
	defer func() {
		fc.CountPush(fcm.ID(), success, duration)
	}()
	fcm.log.Info("Pushed", "duration", duration)
	if err != nil {
		// TODO: Isn't there a better way?
		if strings.Contains(err.Error(), "registration-token-not-registered") {
			fc.TokenInvalid(fcm.ID(), msg.Message.Token)
		} else {
			fcm.log.Error("Posting failed", "error", err)
		}
		return services.PushStatusHardFail
	}
	success = true
	return services.PushStatusSuccess
}
