# When push comes to shove...

[![Go Report Card](https://goreportcard.com/badge/gitlab.com/pennersr/shove)](https://goreportcard.com/report/gitlab.com/pennersr/shove)

## Background

This is the replacement for [Pulsus](https://github.com/pennersr/pulsus) which has been steadily serving up to 100M push notifications. But, given that it was still using the binary APNS protocol it was due for an upgrade.

## Features

- Asynchronous: a push client can just fire & forget.
- Feedback: asynchronously receive information on invalid device tokens.
- Services:
  - APNS
  - Email: supports automatic creation of email digests in case the rate limit
    is exceeded
  - FCM
  - Web Push
  - Telegram
- Multiple workers per push service.
- Queueing: both in-memory and persistent via Redis.
- Exponential back-off in case of failure.
- Less moving parts: when using Redis, you can push directly to the queue, bypassing the need for the Shove server to be up and running.
- Prometheus support


## Why?

- https://github.com/appleboy/gorush/issues/386#issuecomment-479191179

- https://github.com/mercari/gaurun/issues/115


## Usage

### Running

Start the server:

    $ shove \
        -api-addr localhost:8322 \
        -queue-redis redis://redis:6379 \
        -fcm-api-key $FCM_API_KEY \
        -apns-certificate-path /etc/shove/apns/production/bundle.pem -apns-sandbox-certificate-path /etc/shove/apns/sandbox/bundle.pem \
        -webpush-vapid-public-key=abc123 -webpush-vapid-private-key=secret \
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

    2021/03/23 21:15:57 Using Redis queue at redis://localhost:6379
    2021/03/23 21:15:57 Initializing Email service
    2021/03/23 21:15:57 Serving on localhost:8322
    2021/03/23 21:15:57 Shove server started
    2021/03/23 21:15:57 Email worker started
    2021/03/23 21:15:57 Email digester started
    2021/03/23 21:15:58 Email Sending email
    2021/03/23 21:15:59 Email Sending email
    2021/03/23 21:15:59 Email Sending email
    2021/03/23 21:16:00 Email rate to john@doe.org too high, digested
    2021/03/23 21:16:12 Email rate to john@doe.org too high, digested
    2021/03/23 21:16:18 Sending digest email
