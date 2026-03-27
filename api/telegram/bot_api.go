package telegram

import (
	"GPTBot/pipeline"
	"GPTBot/pipeline/sender"
)

// BotAPI is the full interface of the Telegram bot.
// Sub-interfaces (sender.MessageSender, pipeline.FileResolver) are defined
// in their consumer packages; BotAPI composes them all for convenience
// in the composition root.
type BotAPI interface {
	sender.MessageSender
	pipeline.FileResolver
	GetUsername() string
}

// compile-time checks
var _ BotAPI = (*Bot)(nil)
var _ sender.MessageSender = (*Bot)(nil)
var _ pipeline.FileResolver = (*Bot)(nil)

// GetUsername returns the bot's Telegram username.
func (botInstance *Bot) GetUsername() string {
	return botInstance.Username
}

// GetFile resolves a Telegram file ID into a pipeline.FileInfo.
func (botInstance *Bot) GetFile(fileID string) (pipeline.FileInfo, error) {
	f, err := botInstance.transport.GetFile(fileID)
	if err != nil {
		return pipeline.FileInfo{}, err
	}
	return pipeline.FileInfo{FilePath: f.FilePath}, nil
}
