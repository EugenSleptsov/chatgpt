package telegram

import (
	"GPTBot/api/logger"
	conf "GPTBot/config"
	"fmt"
	"io"
	"net/http"

	"github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// Bot is a low-level Telegram transport layer.
// It handles message delivery, file operations and update polling.
// Authorization, admin notifications and business logic live elsewhere.
type Bot struct {
	api       *tgbotapi.BotAPI
	Config    *conf.Config
	Username  string
	Token     string
	AdminId   int64
	LogClient logger.Log
}

type UpdatesChannel <-chan Update
type Update tgbotapi.Update

func NewInstance(config *conf.Config, logClient logger.Log) (*Bot, error) {
	api, err := tgbotapi.NewBotAPI(config.TelegramToken)
	if err != nil {
		return nil, err
	}

	bot := &Bot{
		api:       api,
		Config:    config,
		Username:  api.Self.UserName,
		Token:     config.TelegramToken,
		AdminId:   config.AdminId,
		LogClient: logClient,
	}

	bot.SetCommandList(config.CommandMenu)

	bot.LogClient.Logf("Authorized on account %s", bot.api.Self.UserName)

	return bot, nil
}

func (botInstance *Bot) GetUpdateChannel(timeout int) UpdatesChannel {
	botInstance.api.Debug = false

	updateConfig := tgbotapi.NewUpdate(0)
	updateConfig.Timeout = timeout

	updates := botInstance.api.GetUpdatesChan(updateConfig)

	ourChannel := make(chan Update)
	go func(channel tgbotapi.UpdatesChannel) {
		defer close(ourChannel)
		for update := range channel {
			ourChannel <- Update(update)
		}
	}(updates)

	return ourChannel
}

// --- Message delivery ---

func (botInstance *Bot) ReplyMarkdown(chatID int64, replyTo int, text string, isMarkdown bool) {
	botInstance.send(chatID, replyTo, text, isMarkdown)
}

func (botInstance *Bot) Reply(chatID int64, replyTo int, text string) {
	botInstance.send(chatID, replyTo, text, false)
}

func (botInstance *Bot) Message(message string, chatID int64, isMarkdown bool) {
	botInstance.send(chatID, 0, message, isMarkdown)
}

// send splits a message into chunks and delivers each one.
func (botInstance *Bot) send(chatID int64, replyTo int, text string, isMarkdown bool) {
	chunks := splitMessage(text)
	for _, chunk := range chunks {
		botInstance.sendChunk(chatID, replyTo, chunk, isMarkdown)
	}
}

// sendChunk tries to send a single chunk. If markdown fails, falls back to plain text.
func (botInstance *Bot) sendChunk(chatID int64, replyTo int, text string, isMarkdown bool) {
	if isMarkdown {
		msg := tgbotapi.NewMessage(chatID, formatMarkdownV2(text))
		msg.ParseMode = "MarkdownV2"
		if replyTo != 0 {
			msg.ReplyToMessageID = replyTo
		}
		if _, err := botInstance.api.Send(msg); err == nil {
			return
		}
		botInstance.LogClient.Logf("Markdown failed, falling back to plain text")
	}

	msg := tgbotapi.NewMessage(chatID, text)
	if replyTo != 0 {
		msg.ReplyToMessageID = replyTo
	}
	if _, err := botInstance.api.Send(msg); err != nil {
		botInstance.LogClient.Logf("Error sending message: %v", err)
	}
}

// --- File operations ---

func (botInstance *Bot) SendImage(chatID int64, imageUrl string, caption string) error {
	response, err := http.Get(imageUrl)
	if err != nil {
		return err
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			return
		}
	}(response.Body)

	imageData, err := io.ReadAll(response.Body)
	if err != nil {
		return err
	}

	photoMsg := tgbotapi.NewPhoto(chatID, tgbotapi.FileBytes{Name: "image.png", Bytes: imageData})
	photoMsg.Caption = caption
	_, err = botInstance.api.Send(photoMsg)
	return err
}

func (botInstance *Bot) GetFile(fileId string) (tgbotapi.File, error) {
	return botInstance.api.GetFile(tgbotapi.FileConfig{FileID: fileId})
}

func (botInstance *Bot) AudioUpload(chatID int64, bytes []byte) error {
	audioMsg := tgbotapi.NewAudio(chatID, tgbotapi.FileBytes{Name: "audio.ogg", Bytes: bytes})
	_, err := botInstance.api.Send(audioMsg)
	return err
}

// --- Telegram API helpers ---

func (botInstance *Bot) GetUserCount(chatID int64) (int, error) {
	return botInstance.api.GetChatMembersCount(tgbotapi.ChatMemberCountConfig{ChatConfig: tgbotapi.ChatConfig{ChatID: chatID}})
}

// --- Helpers ---

func GetChatTitle(update Update) string {
	if update.Message.Chat.ID > 0 {
		return fmt.Sprintf("%s %s [@%s / %d]", update.Message.Chat.FirstName, update.Message.Chat.LastName, update.Message.Chat.UserName, update.Message.Chat.ID)
	}

	return fmt.Sprintf("Chat %d [%s]", update.Message.Chat.ID, update.Message.Chat.Title)
}
