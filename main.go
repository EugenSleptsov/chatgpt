package main

import (
	"GPTBot/api/gpt"
	"GPTBot/api/telegram"
	"GPTBot/util"
	"fmt"
	"log"
	"time"
)

var chats = make(map[int64]*Chat)

type Chat struct {
	ChatID           int64
	Settings         ChatSettings
	History          []*ConversationEntry
	ImageGenNextTime time.Time
}

type ChatSettings struct {
	Temperature float32
	Model       string
	MaxMessages int
}

type ConversationEntry struct {
	Prompt   gpt.Message
	Response gpt.Message
}

type Config struct {
	TelegramToken     string
	GPTToken          string
	TimeoutValue      int
	MaxMessages       int
	AdminId           int64
	IgnoreReportIds   []int64
	AuthorizedUserIds []int64
}

func (c *Config) String() string {
	return fmt.Sprintf("Config{\n  TelegramToken: %s,\n  GPTToken: %s,\n  TimeoutValue: %d,\n  MaxMessages: %d,\n  AdminId: %d,\n  IgnoreReportIds: %v,\n  AuthorizedUserIds: %v,\n}",
		c.TelegramToken, c.GPTToken, c.TimeoutValue, c.MaxMessages, c.AdminId, c.IgnoreReportIds, c.AuthorizedUserIds)
}

func main() {
	config, err := readConfig("bot.conf")
	if err != nil {
		log.Fatalf("Error reading bot.conf: %v", err)
	}

	bot, err := telegram.NewBot(config.TelegramToken)
	if err != nil {
		log.Fatal(err)
	}

	gptClient := &gpt.GPTClient{
		ApiKey: config.GPTToken,
	}

	// buffer up to 100 update messages
	updateChan := make(chan telegram.Update, 100)

	// create a pool of worker goroutines
	numWorkers := 10
	for i := 0; i < numWorkers; i++ {
		go worker(updateChan, bot, gptClient, config)
	}

	for update := range bot.GetUpdateChannel(config.TimeoutValue) {
		// Ignore any non-Message Updates
		if update.Message == nil {
			continue
		}

		chat, ok := chats[update.Message.Chat.ID]
		if !ok {
			chat = &Chat{
				ChatID: update.Message.Chat.ID,
				Settings: ChatSettings{
					Temperature: 0.6,
					Model:       "gpt-3.5-turbo",
					MaxMessages: config.MaxMessages,
				},
				History:          make([]*ConversationEntry, 0),
				ImageGenNextTime: time.Now(),
			}
			chats[chat.ChatID] = chat
		}

		// If no authorized users are provided, make the bot public
		if len(config.AuthorizedUserIds) > 0 {
			if !util.IsIdInList(update.Message.From.ID, config.AuthorizedUserIds) {
				bot.Reply(chat.ChatID, update.Message.MessageID, "Sorry, you do not have access to this bot.")
				log.Printf("Unauthorized access attempt by user %d: %s %s (%s)", update.Message.From.ID, update.Message.From.FirstName, update.Message.From.LastName, update.Message.From.UserName)

				// Notify the admin
				if config.AdminId > 0 {
					adminMessage := fmt.Sprintf("Unauthorized access attempt by user %d: %s %s (%s)", update.Message.From.ID, update.Message.From.FirstName, update.Message.From.LastName, update.Message.From.UserName)
					bot.Message(adminMessage, config.AdminId)
				}
				continue
			}
		}

		// Send the Update to the worker goroutines via the channel
		updateChan <- update
	}
}

func formatHistory(history []gpt.Message) []string {
	if len(history) == 0 {
		return []string{"История разговоров пуста."}
	}

	var historyMessage string
	var historyMessages []string
	characterCount := 0

	for i, message := range history {
		formattedLine := fmt.Sprintf("%d. %s: %s\n", i+1, util.Title(message.Role), message.Content)
		lineLength := len(formattedLine)

		if characterCount+lineLength > 4096 {
			historyMessages = append(historyMessages, historyMessage)
			historyMessage = ""
			characterCount = 0
		}

		historyMessage += formattedLine
		characterCount += lineLength
	}

	if len(historyMessage) > 0 {
		historyMessages = append(historyMessages, historyMessage)
	}

	return historyMessages
}

func messagesFromHistory(history []*ConversationEntry) []gpt.Message {
	var messages []gpt.Message
	for _, entry := range history {
		messages = append(messages, entry.Prompt)
		if entry.Response != (gpt.Message{}) {
			messages = append(messages, entry.Response)
		}
	}
	return messages
}
