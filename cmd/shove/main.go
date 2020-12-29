package main

import (
	"context"
	"flag"
	"gitlab.com/pennersr/shove/internal/queue"
	"gitlab.com/pennersr/shove/internal/queue/memory"
	"gitlab.com/pennersr/shove/internal/queue/redis"
	"gitlab.com/pennersr/shove/internal/server"
	"gitlab.com/pennersr/shove/internal/services/apns"
	"gitlab.com/pennersr/shove/internal/services/fcm"
	"gitlab.com/pennersr/shove/internal/services/telegram"
	"gitlab.com/pennersr/shove/internal/services/webpush"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"
)

var apiAddr = flag.String("api-addr", ":8322", "API address to listen to")
var apnsCertificate = flag.String("apns-certificate-path", "", "APNS certificate path")
var apnsSandboxCertificate = flag.String("apns-sandbox-certificate-path", "", "APNS sandbox certificate path")
var fcmAPIKey = flag.String("fcm-api-key", "", "FCM API key")
var redisURL = flag.String("queue-redis", "", "Use Redis queue (Redis URL)")
var webPushVAPIDPublicKey = flag.String("webpush-vapid-public-key", "", "VAPID public key")
var webPushVAPIDPrivateKey = flag.String("webpush-vapid-private-key", "", "VAPID public key")
var telegramBotToken = flag.String("telegram-bot-token", "", "Telegram bot token")

func main() {
	flag.Parse()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	var qf queue.QueueFactory
	if *redisURL == "" {
		log.Println("Using non-persistent in-memory queue")
		qf = memory.MemoryQueueFactory{}
	} else {
		log.Println("Using Redis queue at", *redisURL)
		qf = redis.NewQueueFactory(*redisURL)
	}
	s := server.NewServer(*apiAddr, qf)

	if *apnsCertificate != "" {
		apns, err := apns.NewAPNS(*apnsCertificate, true)
		if err != nil {
			log.Fatal("Error setting up APNS service:", err)
		}
		if err := s.AddService(apns); err != nil {
			log.Fatal("Error adding APNS service:", err)
		}
	}

	if *apnsSandboxCertificate != "" {
		apns, err := apns.NewAPNS(*apnsSandboxCertificate, false)
		if err != nil {
			log.Fatal("Error setting up APNS sandbox service:", err)
		}
		if err := s.AddService(apns); err != nil {
			log.Fatal("Error adding APNS sandbox service:", err)
		}
	}

	if *fcmAPIKey != "" {
		fcm, err := fcm.NewFCM(*fcmAPIKey)
		if err != nil {
			log.Fatal("Error setting up FCM service:", err)
		}
		if err := s.AddService(fcm); err != nil {
			log.Fatal("Error adding FCM service:", err)
		}
	}

	if *webPushVAPIDPrivateKey != "" {
		web, err := webpush.NewWebPush(*webPushVAPIDPublicKey, *webPushVAPIDPrivateKey)
		if err != nil {
			log.Fatal("Error setting up WebPush service:", err)
		}
		if err := s.AddService(web); err != nil {
			log.Fatal("Error adding WebPush service:", err)
		}
	}

	if *telegramBotToken != "" {
		tg, err := telegram.NewTelegramService(*telegramBotToken)
		if err != nil {
			log.Fatal("Error setting up Telegram service:", err)
		}
		if err := s.AddService(tg); err != nil {
			log.Fatal("Error adding Telegram service:", err)
		}
	}

	go func() {
		err := s.Serve()
		if err != nil {
			log.Fatal("Error serving:", err)
		}
	}()
	<-stop
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	s.Shutdown(ctx)
	log.Println("Exiting")
}
