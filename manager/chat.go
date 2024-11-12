package manager

import (
	"GPTBot/api/gpt"
	"GPTBot/api/telegram"
	conf "GPTBot/config"
	"GPTBot/storage"
	"time"
)

type ChatManager struct {
	StorageClient storage.Storage
	Config        *conf.Config
}

func NewChatManager(storageClient storage.Storage, config *conf.Config) *ChatManager {
	return &ChatManager{StorageClient: storageClient, Config: config}
}

func (cm *ChatManager) GetOrCreateChat(update telegram.Update) *storage.Chat {
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
