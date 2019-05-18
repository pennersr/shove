# When push comes to shove...

[![Go Report Card](https://goreportcard.com/badge/gitlab.com/pennersr/shove)](https://goreportcard.com/report/gitlab.com/pennersr/shove)

## Background

This is the replacement for [Pulsus](https://github.com/pennersr/pulsus) which has been steadily serving up to 100M push notifications. But, given that it was still using the binary APNS protocol it was due for an upgrade.

## Features

- Asynchronous: a push client can just fire & forget.
- Feedback: asynchronously receive information on invalid device tokens.
- Services: APNS, FCM, Web Push.
- Multiple workers per push service.
- Queueing: both in-memory and persistent via Redis.
- Exponential back-off in case of failure.
- Less moving parts: when using Redis, you can push directly to the queue, bypassing the need for the Shove server to be up and running.
- Prometheus support


## Why?

- https://github.com/appleboy/gorush/issues/386#issuecomment-479191179

- https://github.com/mercari/gaurun/issues/115


## Usage

Running:
```
shove -fcm-api-key $FCM_API_KEY -apns-certificate-path /etc/shove/apns/production/bundle.pem -apns-sandbox-certificate-path /etc/shove/apns/sandbox/bundle.pem -api-addr localhost:8322 -queue-redis redis://redis:6379 -webpush-vapid-public-key=abc123 -webpush-vapid-private-key=secret
```

Receive feedback:
```
$ curl -X POST 'http://localhost:8322/api/feedback'

{
  "feedback": [
    {"service":"apns-sandbox",
     "token":"881becff86cbd221544044d3b9aeaaf6314dfbef2abb2fe313f3725f4505cb47",
     "reason":"invalid"}
  ]
}
```

Push an APNS notification:
```
$ curl  -i  --data '{"service": "apns", "headers": {"apns-priority": 10, "apns-topic": "com.shove.app"}, "payload": {"aps": { "alert": "hi"}}, "token": "81b8ecff8cb6d22154404d43b9aeaaf6219dfbef2abb2fe313f3725f4505cb47"}' http://localhost:8322/api/push/apns

HTTP/1.1 202 Accepted
Date: Tue, 07 May 2019 19:00:15 GMT
Content-Length: 2
Content-Type: text/plain; charset=utf-8

OK
```

Push an FCM notification:
```
$ curl  -i  --data '{"to": "feE8R6apOdA:AA91PbGHMX5HUoB-tbcqBO_e75NbiOc2AiFbGL3rrYtc99Z5ejbGmCCvOhKW5liqfOzRGOXxto5l7y6b_0dCc-AQ2_bXOcDkcPZgsXGbZvmEjaZA72DfVkZ2pfRrcpcc_9IiiRT5NYC", "notification": {"title": "Hello"}}' http://localhost:8322/api/push/fcm

HTTP/1.1 202 Accepted
Date: Tue, 07 May 2019 19:00:15 GMT
Content-Length: 2
Content-Type: text/plain; charset=utf-8

OK
```

Push a WebPush notification:
```
$ curl  -i  --data '{"subscription": {"endpoint":"https://updates.push.services.mozilla.com/wpush/v2/gAAAAAc4BA....UrjGlg","keys":{"auth":"Hbj3ap...al9ew","p256dh":"BeKdTC3...KLGBJlgF"}}, "headers": {"ttl": 3600, "urgency": "high"}, "token": "use-this-for-feedback-instead-of-subscription", "payload": {"hello":"world"}}' http://localhost:8322/api/push/webpush
```

The subscription (serialized as a JSON string) is used for receiving
feedback. Alternatively, you can specify an optional `token` parameter as done
in the example above.