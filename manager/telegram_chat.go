package manager

import (
	"GPTBot/api/gpt"
	"GPTBot/api/logger"
	"GPTBot/api/telegram"
	conf "GPTBot/config"
	"GPTBot/storage"
	"fmt"
	"strings"
	"time"
)

type TelegramChatManager struct {
	StorageClient storage.Storage
	Config        *conf.Config
	FileLogClient logger.FileLog
}

func NewTelegramChatManager(storageClient storage.Storage, config *conf.Config, fileLogClient logger.FileLog) *TelegramChatManager {
	return &TelegramChatManager{
		StorageClient: storageClient,
		Config:        config,
		FileLogClient: fileLogClient,
	}
}

func (cm *TelegramChatManager) GetStorageClient() storage.Storage {
	return cm.StorageClient
}

func (cm *TelegramChatManager) GetOrCreateChat(update telegram.Update) *storage.Chat {
	chatID := update.Message.Chat.ID
	chat, ok := cm.StorageClient.Get(chatID)
	if !ok {
		chat = &storage.Chat{
			ChatID: chatID,
			Settings: storage.ChatSettings{
				MaxMessages:     cm.Config.MaxMessages,
				UseMarkdown:     true,
				SummarizePrompt: cm.Config.SummarizePrompt,
				Token:           cm.Config.GPTToken,
			},
			Sessions: []*storage.Session{{
				ID:           storage.DefaultSessionID,
				Topic:        storage.DefaultSessionTopic,
				History:      make([]*storage.ConversationEntry, 0),
				SystemPrompt: "You are a helpful assistant...",
				Model:        gpt.DefaultTierID,
				Temperature:  storage.DefaultSessionTemperature,
			}},
			ActiveSessionID:  storage.DefaultSessionID,
			NextSessionID:    storage.DefaultNextSessionID,
			ImageGenNextTime: time.Now(),
			Title:            telegram.GetChatTitle(update),
		}
		_ = cm.StorageClient.Set(chatID, chat)
	}
	chat.Title = telegram.GetChatTitle(update)
	return chat
}

func (cm *TelegramChatManager) LogMessage(update telegram.Update, chat *storage.Chat) {
	var lines []string
	name := update.Message.From.FirstName + " " + update.Message.From.LastName
	for _, v := range strings.Split(update.Message.Text, "\n") {
		if v != "" {
			lines = append(lines, v)
		}
	}

	if chat.ChatID < 0 {
		for i := range lines {
			lines[i] = fmt.Sprintf("%s: %s", name, lines[i])
		}
	}

	cm.FileLogClient.LogToFile(fmt.Sprintf("log/%d.log", chat.ChatID), lines)
}
