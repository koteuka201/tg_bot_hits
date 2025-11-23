package main

import (
	"log"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type App struct {
	bot     *tgbotapi.BotAPI
	storage *Storage
}

func NewApp(token, persistencePath string) (*App, error) {
	bot, err := tgbotapi.NewBotAPI(token)
	if err != nil {
		return nil, err
	}
	bot.Debug = true // Добавили дебаг
	
	// Логируем успешную авторизацию
	log.Printf("Authorized on account %s", bot.Self.UserName)
	
	st, err := NewStorage(persistencePath)
	if err != nil {
		return nil, err
	}
	return &App{bot: bot, storage: st}, nil
}

func (a *App) Run() {
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 30

	updates := a.bot.GetUpdatesChan(u)
	log.Println("Bot started and waiting for updates...")

	for update := range updates {
		go func(update tgbotapi.Update) {
			defer func() {
				if r := recover(); r != nil {
					log.Printf("PANIC in handler: %v", r)
				}
			}()

			if update.Message == nil {
				return
			}

			// Проверяем, что From не nil
			if update.Message.From == nil {
				log.Println("Message has no From field")
				return
			}

			chatID := update.Message.Chat.ID
			userID := update.Message.From.ID
			text := update.Message.Text

			log.Printf("Received message from user %d in chat %d: %s", userID, chatID, text)

			if update.Message.IsCommand() {
				switch update.Message.Command() {
				case "start":
					a.handleStart(chatID, userID)
				case "show_data":
					a.handleShowData(chatID, userID)
				default:
					a.sendText(chatID, "Unknown command")
				}
				return
			}

			state := a.storage.GetUserField(userID, "state")
			log.Printf("User %d state: %s", userID, state)
			
			switch state {
			case "":
				a.sendText(chatID, "Send /start to begin the conversation")
			case "CHOOSING":
				a.handleChoosing(chatID, userID, text)
			case "TYPING_CHOICE":
				a.handleTypingChoice(chatID, userID, text)
			case "TYPING_REPLY":
				a.handleTypingReply(chatID, userID, text)
			default:
				a.sendText(chatID, "Unknown state, send /start")
				a.storage.SetUserField(userID, "state", "")
			}
		}(update)
	}
}

func (a *App) sendText(chatID int64, text string) {
	msg := tgbotapi.NewMessage(chatID, text)
	log.Printf("Sending message to chat %d: %s", chatID, text)
	resp, err := a.bot.Send(msg)
	if err != nil {
		log.Printf("failed to send message: %v", err)
	} else {
		log.Printf("Message sent successfully: %v", resp.MessageID)
	}
}

func (a *App) handleStart(chatID int64, userID int64) {
	log.Printf("handleStart called with chatID=%d, userID=%d", chatID, userID)
	userData := a.storage.GetUserMap(userID)
	log.Printf("Got userData: %v", userData)
	replyText := "Hi! My name is Doctor Botter."
	if len(userData) > 0 {
		keys := make([]string, 0, len(userData))
		for k := range userData {
			if k == "state" || k == "choice" {
				continue
			}
			keys = append(keys, k)
		}
		if len(keys) > 0 {
			replyText += " You already told me your " + strings.Join(keys, ", ") + ". Why don't you tell me something more?"
		} else {
			replyText += " Tell me about yourself."
		}
	} else {
		replyText += " I will hold a more complex conversation with you. Why don't you tell me something about yourself?"
	}

	log.Printf("Setting user state to CHOOSING for user %d", userID)
	a.storage.SetUserField(userID, "state", "CHOOSING")
	log.Printf("State set successfully")

	keyboard := tgbotapi.NewReplyKeyboard(
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("Age"),
			tgbotapi.NewKeyboardButton("Favourite colour"),
		),
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("Number of siblings"),
			tgbotapi.NewKeyboardButton("Something else..."),
		),
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("Done"),
		),
	)

	msg := tgbotapi.NewMessage(chatID, replyText)
	msg.ReplyMarkup = keyboard
	log.Printf("About to send start message to chat %d: %s", chatID, replyText)
	resp, err := a.bot.Send(msg)
	if err != nil {
		log.Printf("SEND START FAILED: %v", err)
	} else {
		log.Printf("start message sent successfully: %v", resp.MessageID)
	}
	log.Printf("handleStart finished")
}

func (a *App) handleShowData(chatID int64, userID int64) {
	userData := a.storage.GetUserMap(userID)
	a.sendText(chatID, "This is what you already told me: "+factsToStr(userData))
}

func (a *App) handleChoosing(chatID int64, userID int64, text string) {
	lower := strings.ToLower(text)
	if text == "Done" {
		if a.storage.HasUserField(userID, "choice") {
			a.storage.DeleteUserField(userID, "choice")
		}
		userData := a.storage.GetUserMap(userID)
		a.sendText(chatID, "I learned these facts about you: "+factsToStr(userData)+" Until next time!")
		msg := tgbotapi.NewMessage(chatID, "Conversation ended")
		msg.ReplyMarkup = tgbotapi.NewRemoveKeyboard(true)
		a.bot.Send(msg)
		a.storage.SetUserField(userID, "state", "")
		return
	}

	switch text {
	case "Age", "Favourite colour", "Number of siblings":
		a.storage.SetUserField(userID, "choice", lower)
		if val := a.storage.GetUserField(userID, lower); val != "" {
			a.sendText(chatID, "Your "+lower+"? I already know the following about that: "+val)
		} else {
			a.sendText(chatID, "Your "+lower+"? Yes, I would love to hear about that!")
		}
		a.storage.SetUserField(userID, "state", "TYPING_REPLY")
	case "Something else...":
		a.sendText(chatID, `Alright, please send me the category first, for example "Most impressive skill"`)
		a.storage.SetUserField(userID, "state", "TYPING_CHOICE")
	default:
		a.sendText(chatID, "Please choose one of the keyboard options or send Done")
	}
}

func (a *App) handleTypingChoice(chatID int64, userID int64, text string) {
	k := strings.ToLower(text)
	a.storage.SetUserField(userID, "choice", k)
	a.sendText(chatID, "Got it. Now send me the value for "+k)
	a.storage.SetUserField(userID, "state", "TYPING_REPLY")
}

func (a *App) handleTypingReply(chatID int64, userID int64, text string) {
	category := a.storage.GetUserField(userID, "choice")
	if category == "" {
		a.sendText(chatID, "No category chosen. Send /start")
		a.storage.SetUserField(userID, "state", "")
		return
	}
	a.storage.SetUserField(userID, category, strings.ToLower(text))
	a.storage.DeleteUserField(userID, "choice")
	a.sendText(chatID, "Neat! Here's what you already told me:\n"+factsToStr(a.storage.GetUserMap(userID))+"\nYou can tell me more, or change your opinion on something.")
	a.storage.SetUserField(userID, "state", "CHOOSING")
}