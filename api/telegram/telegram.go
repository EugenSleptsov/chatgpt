package telegram

import (
	"GPTBot/api/logger"
	conf "GPTBot/config"
	"io"
	"net/http"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// Bot is the high-level Telegram bot used for GPT interactions.
// It adds message formatting, splitting, file helpers and update
// polling on top of the shared Transport layer.
type Bot struct {
	Username  string
	LogClient logger.Log
	transport *Transport
}

// FileURL returns the full download URL for a Telegram file path.
func (botInstance *Bot) FileURL(filePath string) string {
	return botInstance.transport.FileURL(filePath)
}

type UpdatesChannel <-chan Update
type Update tgbotapi.Update

// Msg returns the effective message from an update.
// It looks through Message, EditedMessage, ChannelPost, EditedChannelPost
// and returns the first non-nil one. Returns nil if none is present.
func (u Update) Msg() *tgbotapi.Message {
	switch {
	case u.Message != nil:
		return u.Message
	case u.EditedMessage != nil:
		return u.EditedMessage
	case u.ChannelPost != nil:
		return u.ChannelPost
	case u.EditedChannelPost != nil:
		return u.EditedChannelPost
	default:
		return nil
	}
}

// IsEdited returns true when the update is an edit of an existing message.
func (u Update) IsEdited() bool {
	return u.EditedMessage != nil || u.EditedChannelPost != nil
}

func NewInstance(config *conf.Config, logClient logger.Log) (*Bot, error) {
	transport, err := NewTransport(config.TelegramToken)
	if err != nil {
		return nil, err
	}

	bot := &Bot{
		Username:  transport.Username(),
		LogClient: logClient,
		transport: transport,
	}

	bot.SetCommandList(config.CommandMenu)

	bot.LogClient.Logf("Authorized on account %s", bot.Username)

	return bot, nil
}

func (botInstance *Bot) GetUpdateChannel(timeout int) UpdatesChannel {
	updateConfig := tgbotapi.NewUpdate(0)
	updateConfig.Timeout = timeout

	updates := botInstance.transport.GetUpdatesChan(updateConfig)

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

// sendChunk tries to send a single chunk. If HTML formatting fails, falls back to plain text.
func (botInstance *Bot) sendChunk(chatID int64, replyTo int, text string, isMarkdown bool) {
	if isMarkdown {
		msg := tgbotapi.NewMessage(chatID, markdownToHTML(text))
		msg.ParseMode = "HTML"
		if replyTo != 0 {
			msg.ReplyToMessageID = replyTo
		}
		if _, err := botInstance.transport.Send(msg); err == nil {
			return
		}
		botInstance.LogClient.Logf("HTML formatting failed, falling back to plain text")
	}

	msg := tgbotapi.NewMessage(chatID, text)
	if replyTo != 0 {
		msg.ReplyToMessageID = replyTo
	}
	if _, err := botInstance.transport.Send(msg); err != nil {
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
	_, err = botInstance.transport.Send(photoMsg)
	return err
}

func (botInstance *Bot) GetFile(fileId string) (FileInfo, error) {
	f, err := botInstance.transport.GetFile(fileId)
	if err != nil {
		return FileInfo{}, err
	}
	return toFileInfo(f), nil
}

func (botInstance *Bot) AudioUpload(chatID int64, bytes []byte) error {
	audioMsg := tgbotapi.NewAudio(chatID, tgbotapi.FileBytes{Name: "audio.ogg", Bytes: bytes})
	_, err := botInstance.transport.Send(audioMsg)
	return err
}

// SendImageData sends raw image bytes (PNG) to a chat with an optional caption.
func (botInstance *Bot) SendImageData(chatID int64, data []byte, caption string) error {
	photoMsg := tgbotapi.NewPhoto(chatID, tgbotapi.FileBytes{Name: "image.png", Bytes: data})
	photoMsg.Caption = caption
	_, err := botInstance.transport.Send(photoMsg)
	return err
}
