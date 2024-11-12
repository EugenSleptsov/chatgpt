package manager

import (
	"GPTBot/api/log"
	"GPTBot/api/telegram"
	"GPTBot/storage"
	"fmt"
	"strings"
)

type ChatLogger struct {
	LogClient *log.Log
}

func NewChatLogger(logClient *log.Log) *ChatLogger {
	return &ChatLogger{LogClient: logClient}
}

func (cl *ChatLogger) LogMessage(update telegram.Update, chat *storage.Chat) {
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

	cl.LogClient.LogToFile(fmt.Sprintf("log/%d.log", chat.ChatID), lines)
}
