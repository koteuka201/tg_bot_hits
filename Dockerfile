# syntax=docker/dockerfile:1
FROM golang:1.21-alpine AS build

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -o /telegram-doctor-bot

FROM alpine:3.18
RUN apk add --no-cache ca-certificates
COPY --from=build /telegram-doctor-bot /telegram-doctor-bot

# default data path
VOLUME /data
ENV PERSISTENCE_FILE=/data/conversationbot.json

EXPOSE 8080
ENTRYPOINT ["/telegram-doctor-bot"]
