# When push comes to shove...

[![Go Report Card](https://goreportcard.com/badge/gitlab.com/pennersr/shove)](https://goreportcard.com/report/gitlab.com/pennersr/shove) [![Written in Emacs](https://pennersr.github.io/img/emacs-badge.svg)](https://www.gnu.org/software/emacs/)

## Background

This is the replacement for [Pulsus](https://github.com/pennersr/pulsus) which has been steadily serving up to 100M push notifications. But, given that it was still using the binary APNS protocol it was due for an upgrade.

## Overview

Design:
- Asynchronous: a push client can just fire & forget.
- Multiple workers per push service.
- Less moving parts: when using Redis, you can push directly to the queue, bypassing the need for the Shove server to be up and running.

Supported push services:
- APNS
- Email: supports automatic creation of email digests in case the rate limit
  is exceeded
- FCM
- Telegram: supports squashing multiple messages into one in case the rate limit
  is exceeded
- Web Push

Features:
- Feedback: asynchronously receive information on invalid device tokens.
- Queueing: both in-memory and persistent via Redis.
- Exponential back-off in case of failure.
- Prometheus support.
- Squashing of messages in case rate limits are exceeded.


## Why?

- https://github.com/appleboy/gorush/issues/386#issuecomment-479191179

- https://github.com/mercari/gaurun/issues/115


## Usage

### Running

Usage:

    $ shove -h
    Usage of ./shove:
      -api-addr string
            API address to listen to (default ":8322")
      -apns-certificate-path string
            APNS certificate path
      -apns-sandbox-certificate-path string
            APNS sandbox certificate path
      -apns-workers int
            The number of workers pushing APNS messages (default 4)
      -email-host string
            Email host
      -email-port int
            Email port (default 25)
      -email-rate-amount int
            Email max. rate (amount)
      -email-rate-per int
            Email max. rate (per seconds)
      -fcm-api-key string
            FCM API key
      -fcm-workers int
            The number of workers pushing FCM messages (default 4)
      -queue-redis string
            Use Redis queue (Redis URL)
      -telegram-bot-token string
            Telegram bot token
      -telegram-rate-amount int
            Telegram max. rate (amount)
      -telegram-rate-per int
            Telegram max. rate (per seconds)
      -telegram-workers int
            The number of workers pushing Telegram messages (default 2)
      -webpush-vapid-private-key string
            VAPID public key
      -webpush-vapid-public-key string
            VAPID public key
      -webpush-workers int
            The number of workers pushing Web messages (default 8)


Start the server:

    $ shove \
        -api-addr localhost:8322 \
        -queue-redis redis://redis:6379 \
        -fcm-api-key $FCM_API_KEY \
        -apns-certificate-path /etc/shove/apns/production/bundle.pem -apns-sandbox-certificate-path /etc/shove/apns/sandbox/bundle.pem \
        -webpush-vapid-public-key=$VAPID_PUBLIC_KEY -webpush-vapid-private-key=$VAPID_PRIVATE_KEY \
        -telegram-bot-token $TELEGRAM_BOT_TOKEN


### APNS

Push an APNS notification:

    $ curl  -i  --data '{"service": "apns", "headers": {"apns-priority": 10, "apns-topic": "com.shove.app"}, "payload": {"aps": { "alert": "hi"}}, "token": "81b8ecff8cb6d22154404d43b9aeaaf6219dfbef2abb2fe313f3725f4505cb47"}' http://localhost:8322/api/push/apns


A successful push results in:

    HTTP/1.1 202 Accepted
    Date: Tue, 07 May 2019 19:00:15 GMT
    Content-Length: 2
    Content-Type: text/plain; charset=utf-8

    OK


### FCM

Push an FCM notification:

    $ curl  -i  --data '{"to": "feE8R6apOdA:AA91PbGHMX5HUoB-tbcqBO_e75NbiOc2AiFbGL3rrYtc99Z5ejbGmCCvOhKW5liqfOzRGOXxto5l7y6b_0dCc-AQ2_bXOcDkcPZgsXGbZvmEjaZA72DfVkZ2pfRrcpcc_9IiiRT5NYC", "notification": {"title": "Hello"}}' http://localhost:8322/api/push/fcm


### WebPush

Push a WebPush notification:

    $ curl  -i  --data '{"subscription": {"endpoint":"https://updates.push.services.mozilla.com/wpush/v2/gAAAAAc4BA....UrjGlg","keys":{"auth":"Hbj3ap...al9ew","p256dh":"BeKdTC3...KLGBJlgF"}}, "headers": {"ttl": 3600, "urgency": "high"}, "token": "use-this-for-feedback-instead-of-subscription", "payload": {"hello":"world"}}' http://localhost:8322/api/push/webpush

The subscription (serialized as a JSON string) is used for receiving
feedback. Alternatively, you can specify an optional `token` parameter as done
in the example above.


### Telegram

Push a Telegram notification:

    $ curl  -i  --data '{"method": "sendMessage", "payload": {"chat_id": "12345678", "text": "Hello!"}}' http://localhost:8322/api/push/telegram

Note that the Telegram Bot API documents `chat_id` as "Integer or String" --
Shove requires strings to be passed. For users that disconnected from your bot
the chat ID will be communicated back through the feedback mechanism. Here, the
token will equal the unreachable chat ID.


### Receive Feedback

Outdated/invalid tokens are communicated back. To receive those, you can periodically query the feedback channel to receive token feedback, and remove those from your database:


    $ curl -X POST 'http://localhost:8322/api/feedback'

    {
      "feedback": [
        {"service":"apns-sandbox",
         "token":"881becff86cbd221544044d3b9aeaaf6314dfbef2abb2fe313f3725f4505cb47",
         "reason":"invalid"}
      ]
    }


### Email

In order to keep your SMTP server safe from being blacklisted, the email service
supports rate limitting. When the rate is exceeded, multiple mails are
automatically digested.

    $ shove \
        -email-host localhost \
        -email-port 1025 \
        -api-addr localhost:8322 \
        -email-rate-amount 3 \
        -email-rate-per 10 \
        -queue-redis redis://localhost:6379

Push an email:

	$ curl -i -X POST --data @./scripts/email.json http://localhost:8322/api/push/email

If you send too many emails, you'll notice that they are digested, and at a
later time, one digest mail is being sent:

    2021/03/23 21:15:57 Using Redis queue at redis://localhost:6379
    2021/03/23 21:15:57 Initializing Email service
    2021/03/23 21:15:57 Serving on localhost:8322
    2021/03/23 21:15:57 Shove server started
    2021/03/23 21:15:57 email: Worker started
    2021/03/23 21:15:57 email: Digester started
    2021/03/23 21:15:58 email: Sending email
    2021/03/23 21:15:59 email: Sending email
    2021/03/23 21:15:59 email: Sending email
    2021/03/23 21:16:00 email: Rate to john@doe.org exceeded, email digested
    2021/03/23 21:16:12 email: Rate to john@doe.org exceeded, email digested
    2021/03/23 21:16:18 email: Sending digest email


### Redis Queues

Shove is being used to push a high volume of notifications in a production
environment, consisting of various microservices interacting together. In such a
scenario, it is important that the various services are not too tightly coupled
to one another.  For that purpose, Shove offers the ability to post
notifications directly to a Redis queue.

Posting directly to the Redis queue, instead of using the HTTP service
endpoints, has the advantage that you can take Shove offline without disturbing
the operation of the clients pushing the notifications.

Shove intentionally tries to make as little assumptions on the notification
payloads being pushed, as they are mostly handed over as is to the upstream
services. So, when using Shove this way, the client is responsible for handing
over a raw payload. Here's an example:


    package main

    import (
    	"encoding/json"
    	"gitlab.com/pennersr/shove/pkg/shove"
    	"log"
    	"os"
    )

    type FCMNotification struct {
    	To       string            `json:"to"`
    	Data     map[string]string `json:"data,omitempty"`
    }

    func main() {
    	redisURL := os.Getenv("REDIS_URL")
    	if redisURL == "" {
    		redis_URL = "redis://localhost:6379"
    	}
    	client := shove.NewRedisClient(redisURL)

    	notification := FCMNotification{
    		To:   "token....",
    		Data: map[string]string{},
    	}

    	raw, err := json.Marshal(notification)
    	if err != nil {
    		log.Fatal(err)
    	}
    	err = client.PushRaw("fcm", raw)
    	if err != nil {
    		log.Fatal(err)
    	}
    }
