FROM golang:1.22-alpine AS builder

RUN apk add --no-cache git ca-certificates

WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /noveltts ./cmd/server

FROM alpine:3.19

RUN apk add --no-cache ca-certificates tzdata

COPY --from=builder /noveltts /usr/local/bin/noveltts

RUN mkdir -p /data
VOLUME /data

ENV NOVELTTS_CONFIG=/data/config.json
EXPOSE 8080

ENTRYPOINT ["noveltts"]
