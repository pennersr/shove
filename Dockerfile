FROM golang:1.26-alpine AS build
WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/shove ./cmd/shove

FROM alpine:3.20
RUN apk add --no-cache ca-certificates
COPY --from=build /out/shove /usr/local/bin/shove
ENTRYPOINT ["shove"]
