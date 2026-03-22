package service

import (
	"GPTBot/domain/ai"
	"GPTBot/domain/chat"
	"GPTBot/infrastructure/logger"
	"GPTBot/infrastructure/util"
	"GPTBot/pipeline"
	"fmt"
	"strings"
	"time"
)

// ChatDefaults holds the values used when creating a brand-new chat.
type ChatDefaults struct {
	MaxMessages     int
	SummarizePrompt string
	SystemPrompt    string
	LogDir          string
}

// ChatService handles chat lifecycle:
// creating/finding chat objects in storage and writing chat message logs.
type ChatService struct {
	storage  chat.Storage
	defaults ChatDefaults
	fileLog  logger.FileLog
}

func NewChatService(storageClient chat.Storage, defaults ChatDefaults, fileLog logger.FileLog) *ChatService {
	return &ChatService{
		storage:  storageClient,
		defaults: defaults,
		fileLog:  fileLog,
	}
}

func (cs *ChatService) Save() {
	cs.storage.Save()
}

func (cs *ChatService) MarkDirty(chatID int64) {
	cs.storage.MarkDirty(chatID)
}

func (cs *ChatService) GetOrCreateChat(ctx *pipeline.RequestContext) *chat.Chat {
	c, ok := cs.storage.Get(ctx.ChatID)
	if !ok {
		c = &chat.Chat{
			ChatID: ctx.ChatID,
			Settings: chat.ChatSettings{
				MaxMessages:     cs.defaults.MaxMessages,
				UseMarkdown:     true,
				SummarizePrompt: cs.defaults.SummarizePrompt,
			},
			Sessions: []*chat.Session{{
				ID:           chat.DefaultSessionID,
				Topic:        chat.DefaultSessionTopic,
				History:      make([]*chat.ConversationEntry, 0),
				SystemPrompt: cs.defaults.SystemPrompt,
				Model:        ai.DefaultTierID,
			}},
			ActiveSessionID:  chat.DefaultSessionID,
			NextSessionID:    chat.DefaultNextSessionID,
			ImageGenNextTime: time.Now(),
			Title:            ctx.ChatTitle,
		}
		_ = cs.storage.Set(ctx.ChatID, c)
	}
	c.Title = ctx.ChatTitle
	return c
}

func (cs *ChatService) LogMessage(ctx *pipeline.RequestContext, c *chat.Chat) {
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

	cs.fileLog.LogToFile(fmt.Sprintf("%s/%d.log", cs.defaults.LogDir, ctx.ChatID), lines)
}

// ReadChatLog returns the last N lines from a chat's log file.
func (cs *ChatService) ReadChatLog(chatID int64, count int) ([]string, error) {
	return util.ReadLastLines(fmt.Sprintf("%s/%d.log", cs.defaults.LogDir, chatID), count)
}
