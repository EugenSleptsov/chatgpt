package manager

import (
	"GPTBot/api/gpt"
	"GPTBot/api/log"
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
	FileLogClient log.FileLog
}

func NewTelegramChatManager(storageClient storage.Storage, config *conf.Config, fileLogClient log.FileLog) *TelegramChatManager {
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
				Temperature:     0.8,
				Model:           gpt.ModelGPT4OmniMini,
				MaxMessages:     cm.Config.MaxMessages,
				UseMarkdown:     true,
				SystemPrompt:    "You are a helpful assistant...",
				SummarizePrompt: cm.Config.SummarizePrompt,
				Token:           cm.Config.GPTToken,
			},
			History:          make([]*storage.ConversationEntry, 0),
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
