package main

import (
	"GPTBot/api/gpt"
	"GPTBot/api/log"
	"GPTBot/api/telegram"
	"GPTBot/commands"
	"GPTBot/storage"
	"fmt"
	"strings"
	"time"
)

type Worker struct {
	TelegramClient *telegram.Bot
	GptClient      *gpt.GPTClient
	StorageClient  storage.Storage
	LogClient      *log.Log
	CommandFactory commands.CommandFactory
	HandlerFactory UpdateHandlerFactory
}

func NewWorker(telegramClient *telegram.Bot, gptClient *gpt.GPTClient, storageClient storage.Storage, logClient *log.Log, commandFactory commands.CommandFactory, handlerFactory UpdateHandlerFactory) *Worker {
	return &Worker{
		TelegramClient: telegramClient,
		GptClient:      gptClient,
		StorageClient:  storageClient,
		LogClient:      logClient,
		CommandFactory: commandFactory,
		HandlerFactory: handlerFactory,
	}
}

func (w *Worker) Start(updateChan <-chan telegram.Update) {
	for update := range updateChan {
		w.ProcessUpdate(update)
		w.StorageClient.Save()
	}
}

func (w *Worker) LogMessage(update telegram.Update, chat *storage.Chat) {
	// putting history to log file
	// every newline is a new message
	var lines []string
	name := update.Message.From.FirstName + " " + update.Message.From.LastName
	for _, v := range strings.Split(update.Message.Text, "\n") {
		if v != "" {
			lines = append(lines, v)
		}
	}

	// для групповых чатов указываем имя пользователя
	if chat.ChatID < 0 {
		for i := range lines {
			lines[i] = fmt.Sprintf("%s: %s", name, lines[i])
		}
	}

	w.LogClient.LogToFile(fmt.Sprintf("log/%d.log", chat.ChatID), lines)
}

func (w *Worker) ProcessUpdate(update telegram.Update) {
	// Ignore any non-Message Updates
	if update.Message == nil {
		return
	}

	chat := w.GetOrCreateChat(update)

	if !update.Message.IsCommand() {
		w.LogMessage(update, chat)
	}

	// If no authorized users are provided, make the bot public
	if !w.TelegramClient.IsAuthorizedUser(update.Message.From.ID) {
		if update.Message.Chat.Type == "private" {
			w.TelegramClient.Reply(chat.ChatID, update.Message.MessageID, "Sorry, you do not have access to this bot.")
			w.TelegramClient.Log(fmt.Sprintf("[%s]\nMessage: %s", chat.Title, update.Message.Text))
		}
		return
	}

	handler := w.HandlerFactory.GetHandler(update)
	_ = handler.Handle(update, chat)
}

func (w *Worker) GetOrCreateChat(update telegram.Update) *storage.Chat {
	chatID := update.Message.Chat.ID
	chat, ok := w.StorageClient.Get(chatID)
	if !ok {
		chat = &storage.Chat{
			ChatID: update.Message.Chat.ID,
			Settings: storage.ChatSettings{
				Temperature:     0.8,
				Model:           gpt.ModelGPT4OmniMini,
				MaxMessages:     w.TelegramClient.Config.MaxMessages,
				UseMarkdown:     true,
				SystemPrompt:    "You are a helpful ChatGPT bot based on OpenAI GPT Language model. You are a helpful assistant that always tries to help and answer with relevant information as possible.",
				SummarizePrompt: w.TelegramClient.Config.SummarizePrompt,
				Token:           w.TelegramClient.Config.GPTToken,
			},
			History:          make([]*storage.ConversationEntry, 0),
			ImageGenNextTime: time.Now(),
			Title:            telegram.GetChatTitle(update),
		}
		_ = w.StorageClient.Set(chatID, chat)
	}
	chat.Title = telegram.GetChatTitle(update)

	return chat
}
