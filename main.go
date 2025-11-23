package main

import (
    "log"
    "os"
)

func main() {
    token := os.Getenv("BOT_TOKEN")
    if token == "" {
        log.Fatal("BOT_TOKEN is not set")
    }

    dataFile := os.Getenv("PERSISTENCE_FILE")
    if dataFile == "" {
        dataFile = "conversationbot.json"
    }

    app, err := NewApp(token, dataFile)
    if err != nil {
        log.Fatalf("failed to create app: %v", err)
    }

    log.Println("Bot started")
    app.Run()
}
