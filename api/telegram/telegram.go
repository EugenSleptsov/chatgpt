package telegram

import (
	"GPTBot/infrastructure/logger"
	"GPTBot/pipeline/sender"
	"strings"

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

func NewInstance(token string, commandMenu []string, logClient logger.Log) (*Bot, error) {
	transport, err := NewTransport(token)
	if err != nil {
		return nil, err
	}

	bot := &Bot{
		Username:  transport.Username(),
		LogClient: logClient,
		transport: transport,
	}

	bot.SetCommandList(commandMenu)

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

// --- Inline keyboards / callbacks ---

// inlineKeyboard converts transport-agnostic button rows into a Telegram markup.
func inlineKeyboard(rows [][]sender.Button) tgbotapi.InlineKeyboardMarkup {
	kbRows := make([][]tgbotapi.InlineKeyboardButton, 0, len(rows))
	for _, row := range rows {
		kbRow := make([]tgbotapi.InlineKeyboardButton, 0, len(row))
		for _, b := range row {
			kbRow = append(kbRow, tgbotapi.NewInlineKeyboardButtonData(b.Text, b.Data))
		}
		kbRows = append(kbRows, kbRow)
	}
	return tgbotapi.NewInlineKeyboardMarkup(kbRows...)
}

// ReplyWithButtons sends a (short) text reply with an inline keyboard attached.
// Button messages are not split; they are expected to be small control panels.
func (botInstance *Bot) ReplyWithButtons(chatID int64, replyTo int, text string, markdown bool, buttons [][]sender.Button) error {
	kb := inlineKeyboard(buttons)

	if markdown {
		msg := tgbotapi.NewMessage(chatID, markdownToHTML(text))
		msg.ParseMode = "HTML"
		msg.ReplyMarkup = kb
		if replyTo != 0 {
			msg.ReplyToMessageID = replyTo
		}
		if _, err := botInstance.transport.Send(msg); err == nil {
			return nil
		}
		botInstance.LogClient.Logf("HTML formatting failed, falling back to plain text")
	}

	msg := tgbotapi.NewMessage(chatID, text)
	msg.ReplyMarkup = kb
	if replyTo != 0 {
		msg.ReplyToMessageID = replyTo
	}
	_, err := botInstance.transport.Send(msg)
	return err
}

// EditMessage replaces the text and inline keyboard of an existing message.
// A "message is not modified" response (tapping the already-selected option) is
// treated as success.
func (botInstance *Bot) EditMessage(chatID int64, messageID int, text string, markdown bool, buttons [][]sender.Button) error {
	kb := inlineKeyboard(buttons)

	if markdown {
		edit := tgbotapi.NewEditMessageTextAndMarkup(chatID, messageID, markdownToHTML(text), kb)
		edit.ParseMode = "HTML"
		if _, err := botInstance.transport.Send(edit); err == nil || isNotModified(err) {
			return nil
		}
		botInstance.LogClient.Logf("HTML formatting failed on edit, falling back to plain text")
	}

	edit := tgbotapi.NewEditMessageTextAndMarkup(chatID, messageID, text, kb)
	_, err := botInstance.transport.Send(edit)
	if isNotModified(err) {
		return nil
	}
	return err
}

// AnswerCallback acknowledges a button tap (stops Telegram's loading spinner).
func (botInstance *Bot) AnswerCallback(callbackID string, text string) error {
	_, err := botInstance.transport.Request(tgbotapi.NewCallback(callbackID, text))
	return err
}

// isNotModified reports whether err is Telegram's "message is not modified"
// error, which is benign for edits that produce identical content.
func isNotModified(err error) bool {
	return err != nil && strings.Contains(err.Error(), "message is not modified")
}

// --- File operations ---

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
