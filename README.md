# When push comes to shove...

## Background

This is the replacement for [Pulsus](https://github.com/pennersr/pulsus) which has been steadily serving up to 100M push notifications. But, given that it was still using the binary APNS protocol it was due for an upgrade.

## Design

- Asynchronous: a push client can just fire & forget.
- Feedback: receive information on invalid device tokens.

## Features

- APNS suppport.
- Multiple workers per push services.

## Roadmap

- Persist the queue in Redis.
- Add support for FCM.
- Add support for Web push.

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
$ curl -i --data '{"service": "apns-sandbox", "topic": "com.shove.app", "tokens": ["881becff86cbd215244044d3b9eaaaf6219dfbe2abfb2fe313f3725f4505cb47"]}p' http://localhost:8322/api/push

HTTP/1.1 202 Accepted
Date: Tue, 07 May 2019 19:00:15 GMT
Content-Length: 2
Content-Type: text/plain; charset=utf-8

OK
```