package telegram

import tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

// FileInfo is a transport-agnostic wrapper for a remote file descriptor.
type FileInfo struct {
	FilePath string
}

// BotAPI is the interface used by handlers and commands to interact with
// the Telegram transport. Implementing this interface allows mocking in tests.
type BotAPI interface {
	Reply(chatID int64, replyTo int, text string)
	ReplyMarkdown(chatID int64, replyTo int, text string, isMarkdown bool)
	Message(message string, chatID int64, isMarkdown bool)
	SendImage(chatID int64, imageUrl string, caption string) error
	SendImageData(chatID int64, data []byte, caption string) error
	AudioUpload(chatID int64, bytes []byte) error
	GetFile(fileID string) (FileInfo, error)
	FileURL(filePath string) string
	GetUsername() string
}

// compile-time check: Bot implements BotAPI
var _ BotAPI = (*Bot)(nil)

// GetUsername returns the bot's Telegram username.
func (botInstance *Bot) GetUsername() string {
	return botInstance.Username
}

// GetFileInfo wraps tgbotapi.File into transport-agnostic FileInfo.
func toFileInfo(f tgbotapi.File) FileInfo {
	return FileInfo{FilePath: f.FilePath}
}
