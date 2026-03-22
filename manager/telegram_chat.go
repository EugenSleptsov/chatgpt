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

func (cm *TelegramChatManager) Save() {
	cm.StorageClient.Save()
}

func (cm *TelegramChatManager) MarkDirty(chatID int64) {
	cm.StorageClient.MarkDirty(chatID)
}

func (cm *TelegramChatManager) GetOrCreateChat(ctx *telegram.UpdateContext) *storage.Chat {
	chat, ok := cm.StorageClient.Get(ctx.ChatID)
	if !ok {
		chat = &storage.Chat{
			ChatID: ctx.ChatID,
			Settings: storage.ChatSettings{
				MaxMessages:     cm.Config.MaxMessages,
				UseMarkdown:     true,
				SummarizePrompt: cm.Config.SummarizePrompt,
			},
			Sessions: []*storage.Session{{
				ID:           storage.DefaultSessionID,
				Topic:        storage.DefaultSessionTopic,
				History:      make([]*storage.ConversationEntry, 0),
				SystemPrompt: cm.Config.DefaultSystemPrompt,
				Model:        gpt.DefaultTierID,
			}},
			ActiveSessionID:  storage.DefaultSessionID,
			NextSessionID:    storage.DefaultNextSessionID,
			ImageGenNextTime: time.Now(),
			Title:            ctx.ChatTitle(),
		}
		_ = cm.StorageClient.Set(ctx.ChatID, chat)
	}
	chat.Title = ctx.ChatTitle()
	return chat
}

func (cm *TelegramChatManager) LogMessage(ctx *telegram.UpdateContext, chat *storage.Chat) {
	if ctx.SenderName == "" {
		return
	}

	var lines []string
	for _, v := range strings.Split(ctx.Text, "\n") {
		if v != "" {
			lines = append(lines, v)
		}
	}

	if ctx.IsGroup {
		for i := range lines {
			lines[i] = fmt.Sprintf("%s: %s", ctx.SenderName, lines[i])
		}
	}

	cm.FileLogClient.LogToFile(fmt.Sprintf("%s/%d.log", cm.Config.LogDir, ctx.ChatID), lines)
}
