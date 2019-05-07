package main

import (
	"context"
	"flag"
	"gitlab.com/pennersr/shove/internal/queue/memory"
	"gitlab.com/pennersr/shove/internal/server"
	"gitlab.com/pennersr/shove/internal/services/apns"
	"log"
	"os"
	"os/signal"
	"time"
)

var apiAddr = flag.String("api-addr", ":8322", "API address to listen to")
var apnsCertificate = flag.String("apns-certificate-path", "", "APNS certificate path")
var apnsSandboxCertificate = flag.String("apns-sandbox-certificate-path", "", "APNS sandbox certificate path")

func main() {
	flag.Parse()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt)
	s := server.NewServer(*apiAddr, memory.MemoryQueueFactory{})

	if *apnsCertificate != "" {
		apns, err := apns.NewAPNS(*apnsCertificate, true)
		if err != nil {
			log.Fatal("Error setting up APNS service:", err)
		}
		s.AddService(apns)
	}

	if *apnsSandboxCertificate != "" {
		apns, err := apns.NewAPNS(*apnsSandboxCertificate, false)
		if err != nil {
			log.Fatal("Error setting up APNS sandbox service:", err)
		}
		s.AddService(apns)
	}

	go func() {
		err := s.Serve()
		if err != nil {
			log.Fatal("Error serving:", err)
		}
	}()
	<-stop
	ctx, _ := context.WithTimeout(context.Background(), 5*time.Second)
	s.Shutdown(ctx)
	log.Println("Exiting")
}
