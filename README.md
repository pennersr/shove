# When push comes to shove...

## Background

This is the replacement for [Pulsus](https://github.com/pennersr/pulsus) which has been steadily serving up to 100M push notifications. But, given that it was still using the binary APNS protocol it was due for an upgrade.

## Features

- Asynchronous: a push client can just fire & forget.
- Feedback: asynchronously receive information on invalid device tokens.
- Services: APNS.
- Multiple workers per push service.
- Queueing: both in-memory and persistent via Redis.
- Exponential back-off in case of failure.

## Roadmap

- Add support for FCM.
- Monitoring via Prometheus statistics.
- Add support for Web push.
- Push while shove is offline: offer client code to push to Redis directly instead of going via the API.


## Why?

- https://github.com/appleboy/gorush/issues/386#issuecomment-479191179

- https://github.com/mercari/gaurun/issues/115


## API

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

Push a notification:
```
$ curl -i --data '{"service": "apns-sandbox", "headers": {"apns-topic": "com.shove.app", "apns-priority": 10}, "tokens": ["881becff86cbd215244044d3b9eaaaf6219dfbe2abfb2fe313f3725f4505cb47"]}' http://localhost:8322/api/push

HTTP/1.1 202 Accepted
Date: Tue, 07 May 2019 19:00:15 GMT
Content-Length: 2
Content-Type: text/plain; charset=utf-8

OK
```