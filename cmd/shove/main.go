package main

import (
	"context"
	"flag"
	"os"
	"os/signal"
	"syscall"
	"time"

	"codeberg.org/pennersr/shove/internal/queue"
	"codeberg.org/pennersr/shove/internal/queue/memory"
	"codeberg.org/pennersr/shove/internal/queue/redis"
	"codeberg.org/pennersr/shove/internal/server"
	"codeberg.org/pennersr/shove/internal/services"
	"codeberg.org/pennersr/shove/internal/services/apns"
	"codeberg.org/pennersr/shove/internal/services/email"
	"codeberg.org/pennersr/shove/internal/services/fcm"
	"codeberg.org/pennersr/shove/internal/services/telegram"
	"codeberg.org/pennersr/shove/internal/services/webhook"
	"codeberg.org/pennersr/shove/internal/services/webpush"
	"golang.org/x/exp/slog"
)

var debug = flag.Bool("debug", false, "Enable debug logging")
var apiAddr = flag.String("api-addr", ":8322", "API address to listen to")

var apnsCertificate = flag.String("apns-certificate-path", "", "APNS certificate path")
var apnsSandboxCertificate = flag.String("apns-sandbox-certificate-path", "", "APNS sandbox certificate path")
var apnsWorkers = flag.Int("apns-workers", 4, "The number of workers pushing APNS messages")

var fcmCredentialsFile = flag.String("fcm-credentials-file", "", "FCM credentials file")
var fcmWorkers = flag.Int("fcm-workers", 4, "The number of workers pushing FCM messages")

var redisURL = flag.String("queue-redis", "", "Use Redis queue (Redis URL)")

var webhookWorkers = flag.Int("webhook-workers", 0, "The number of workers pushing Webhook messages")

var webPushVAPIDPublicKey = flag.String("webpush-vapid-public-key", "", "VAPID public key")
var webPushVAPIDPrivateKey = flag.String("webpush-vapid-private-key", "", "VAPID public key")
var webPushWorkers = flag.Int("webpush-workers", 8, "The number of workers pushing Web messages")

var telegramBotToken = flag.String("telegram-bot-token", "", "Telegram bot token")
var telegramWorkers = flag.Int("telegram-workers", 2, "The number of workers pushing Telegram messages")
var telegramRateAmount = flag.Int("telegram-rate-amount", 0, "Telegram max. rate (amount)")
var telegramRatePer = flag.Int("telegram-rate-per", 0, "Telegram max. rate (per seconds)")

var emailHost = flag.String("email-host", "", "Email host")
var emailPort = flag.Int("email-port", 25, "Email port")
var emailPlainAuth = flag.Bool("email-plain-auth", false, "Email plain auth(username and password)")
var emailUsername = flag.String("email-username", "", "Email username")
var emailPassword = flag.String("email-password", "", "Email password")
var emailTLS = flag.Bool("email-tls", false, "Use TLS")
var emailTLSInsecure = flag.Bool("email-tls-insecure", false, "Skip TLS verification")
var emailRateAmount = flag.Int("email-rate-amount", 0, "Email max. rate (amount)")
var emailRatePer = flag.Int("email-rate-per", 0, "Email max. rate (per seconds)")

func newLogger() *slog.Logger {
	var opts *slog.HandlerOptions
	if *debug {
		opts = &slog.HandlerOptions{
			Level: slog.LevelDebug,
		}
	}
	logger := slog.New(slog.NewTextHandler(os.Stderr, opts))
	return logger
}

func newServiceLogger(service string) *slog.Logger {
	logger := newLogger()
	return logger.With(
		slog.String("service", service),
	)
}

func main() {
	flag.Parse()

	logger := newLogger()
	slog.SetDefault(logger)

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	var qf queue.QueueFactory
	if *redisURL == "" {
		slog.Info("Using non-persistent in-memory queue")
		qf = memory.MemoryQueueFactory{}
	} else {
		slog.Info("Using Redis queue at", "address", *redisURL)
		qf = redis.NewQueueFactory(*redisURL)
	}
	s := server.NewServer(*apiAddr, qf)

	if *apnsCertificate != "" {
		apns, err := apns.NewAPNS(*apnsCertificate, true, newServiceLogger("apns"))
		if err != nil {
			slog.Error("Failed to setup APNS service", "error", err)
			os.Exit(1)
		}
		if err := s.AddService(apns, *apnsWorkers, services.SquashConfig{}); err != nil {
			slog.Error("Failed to add APNS service", "error", err)
			os.Exit(1)
		}
	}

	if *apnsSandboxCertificate != "" {
		apns, err := apns.NewAPNS(*apnsSandboxCertificate, false, newServiceLogger("apns-sandbox"))
		if err != nil {
			slog.Error("Failed to setup APNS sandbox service", "error", err)
			os.Exit(1)
		}
		if err := s.AddService(apns, *apnsWorkers, services.SquashConfig{}); err != nil {
			slog.Error("Failed to add APNS sandbox service", "error", err)
			os.Exit(1)
		}
	}

	if *fcmCredentialsFile != "" {
		fcm, err := fcm.NewFCM(*fcmCredentialsFile, newServiceLogger("fcm"))
		if err != nil {
			slog.Error("Failed to setup FCM service", "error", err)
			os.Exit(1)
		}
		if err := s.AddService(fcm, *fcmWorkers, services.SquashConfig{}); err != nil {
			slog.Error("Failed to add FCM service", "error", err)
			os.Exit(1)
		}
	}

	if *webhookWorkers > 0 {
		wh, err := webhook.NewWebhook(newServiceLogger("webhook"))
		if err != nil {
			slog.Error("Failed to setup Webhook service", "error", err)
			os.Exit(1)
		}
		if err := s.AddService(wh, *webhookWorkers, services.SquashConfig{}); err != nil {
			slog.Error("Failed to add Webhook service", "error", err)
			os.Exit(1)
		}
	}

	if *webPushVAPIDPrivateKey != "" {
		web, err := webpush.NewWebPush(*webPushVAPIDPublicKey, *webPushVAPIDPrivateKey, newServiceLogger("webpush"))
		if err != nil {
			slog.Error("Failed to setup WebPush service", "error", err)
			os.Exit(1)
		}
		if err := s.AddService(web, *webPushWorkers, services.SquashConfig{}); err != nil {
			slog.Error("Failed to add WebPush service", "error", err)
			os.Exit(1)
		}
	}

	if *telegramBotToken != "" {
		tg, err := telegram.NewTelegramService(*telegramBotToken, newServiceLogger("telegram"))
		if err != nil {
			slog.Error("Failed to setup Telegram service", "error", err)
			os.Exit(1)
		}
		if err := s.AddService(tg, *telegramWorkers, services.SquashConfig{
			RateMax: *telegramRateAmount,
			RatePer: time.Second * time.Duration(*telegramRatePer),
		}); err != nil {
			slog.Error("Failed to add Telegram service", "error", err)
			os.Exit(1)
		}
	}

	if *emailHost != "" {
		config := email.EmailConfig{
			EmailHost:     *emailHost,
			EmailPort:     *emailPort,
			TLS:           *emailTLS,
			TLSInsecure:   *emailTLSInsecure,
			Log:           newServiceLogger("email"),
			PlainAuth:     *emailPlainAuth,
			EmailUsername: *emailUsername,
			EmailPassword: *emailPassword,
		}
		email, err := email.NewEmailService(config)
		if err != nil {
			slog.Error("Failed to setup email service", "error", err)
			os.Exit(1)
		}
		if err := s.AddService(email, 1, services.SquashConfig{
			RateMax: *emailRateAmount,
			RatePer: time.Second * time.Duration(*emailRatePer),
		}); err != nil {
			slog.Error("Failed to add email service", "error", err)
			os.Exit(1)
		}
	}

	go func() {
		slog.Info("Serving", "address", *apiAddr)
		err := s.Serve()
		if err != nil {
			slog.Error("Serve failed", "error", err)
			os.Exit(1)
		}
	}()
	<-stop
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	s.Shutdown(ctx)
	slog.Info("Exiting")
}
