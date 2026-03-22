package telegram

import (
	"fmt"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// Transport is the lowest-level Telegram wrapper.
// It owns a single *tgbotapi.BotAPI and token and exposes only
// primitive operations: send, get file, poll updates.
// No formatting, no business logic, no splitting — just the wire.
type Transport struct {
	api      *tgbotapi.BotAPI
	token    string
	username string
}

// NewTransport creates a Transport by authenticating with the given token.
func NewTransport(token string) (*Transport, error) {
	api, err := tgbotapi.NewBotAPI(token)
	if err != nil {
		return nil, err
	}
	return &Transport{
		api:      api,
		token:    token,
		username: api.Self.UserName,
	}, nil
}

// Username returns the bot account name obtained at authentication time.
func (t *Transport) Username() string { return t.username }

// Send delivers a Chattable (message, photo, audio, etc.) and returns the resulting Message.
func (t *Transport) Send(c tgbotapi.Chattable) (tgbotapi.Message, error) {
	return t.api.Send(c)
}

// Request performs a Telegram API request that does not return a Message
// (e.g. setMyCommands). Returns the raw API response.
func (t *Transport) Request(c tgbotapi.Chattable) (*tgbotapi.APIResponse, error) {
	return t.api.Request(c)
}

// GetFile retrieves file metadata by file ID.
func (t *Transport) GetFile(fileID string) (tgbotapi.File, error) {
	return t.api.GetFile(tgbotapi.FileConfig{FileID: fileID})
}

// FileURL returns the full download URL for a Telegram file path.
func (t *Transport) FileURL(filePath string) string {
	return fmt.Sprintf("https://api.telegram.org/file/bot%s/%s", t.token, filePath)
}

// GetUpdatesChan starts long-polling and returns a channel of raw updates.
func (t *Transport) GetUpdatesChan(config tgbotapi.UpdateConfig) tgbotapi.UpdatesChannel {
	t.api.Debug = false
	return t.api.GetUpdatesChan(config)
}
