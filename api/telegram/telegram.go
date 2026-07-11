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

	if err := bot.SetCommandList(commandMenu); err != nil {
		return nil, err
	}

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

// ReplyWithButtons sends a (short) text reply with an inline keyboard attached
// and returns the sent message's ID. Button messages are not split; they are
// expected to be small control panels.
func (botInstance *Bot) ReplyWithButtons(chatID int64, replyTo int, text string, markdown bool, buttons [][]sender.Button) (int, error) {
	kb := inlineKeyboard(buttons)

	if markdown {
		msg := tgbotapi.NewMessage(chatID, markdownToHTML(text))
		msg.ParseMode = "HTML"
		msg.ReplyMarkup = kb
		if replyTo != 0 {
			msg.ReplyToMessageID = replyTo
		}
		if sent, err := botInstance.transport.Send(msg); err == nil {
			return sent.MessageID, nil
		}
		botInstance.LogClient.Logf("HTML formatting failed, falling back to plain text")
	}

	msg := tgbotapi.NewMessage(chatID, text)
	msg.ReplyMarkup = kb
	if replyTo != 0 {
		msg.ReplyToMessageID = replyTo
	}
	sent, err := botInstance.transport.Send(msg)
	if err != nil {
		return 0, err
	}
	return sent.MessageID, nil
}

// DeleteMessage removes a bot message. Telegram only allows deleting messages
// younger than 48 hours; older ones return an error the caller may ignore.
func (botInstance *Bot) DeleteMessage(chatID int64, messageID int) error {
	_, err := botInstance.transport.Request(tgbotapi.NewDeleteMessage(chatID, messageID))
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

// --- Progress / verbose reporting (service.ProgressReporter) ---

// StartProgress posts a transient status message ("Идет …") and returns a
// function that deletes it. Best-effort: send/delete failures are only logged,
// the surrounding work must not depend on the status message.
func (botInstance *Bot) StartProgress(chatID int64, text string) func() {
	msg := tgbotapi.NewMessage(chatID, text)
	sent, err := botInstance.transport.Send(msg)
	if err != nil {
		botInstance.LogClient.Logf("Error sending progress message: %v", err)
		return func() {}
	}
	return func() {
		if err := botInstance.DeleteMessage(chatID, sent.MessageID); err != nil {
			botInstance.LogClient.Logf("Error deleting progress message: %v", err)
		}
	}
}

// Announce posts a permanent informational message (verbose tool logging).
func (botInstance *Bot) Announce(chatID int64, text string) {
	botInstance.Message(text, chatID, false)
}

// --- File operations ---

// SendForceReply posts a message with a force-reply prompt so the user's next
// message is a reply to it. Selective keeps the prompt targeted at the user.
func (botInstance *Bot) SendForceReply(chatID int64, text string) error {
	msg := tgbotapi.NewMessage(chatID, text)
	msg.ReplyMarkup = tgbotapi.ForceReply{ForceReply: true, Selective: true}
	_, err := botInstance.transport.Send(msg)
	return err
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
