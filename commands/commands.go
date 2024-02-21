package commands

import (
	"GPTBot/api/gpt"
	"GPTBot/api/telegram"
	"GPTBot/storage"
	"GPTBot/util"
	"fmt"
	"log"
	"strings"
)

var CommandList = map[string]Command{
	"help":             &CommandHelp{},
	"start":            &CommandStart{},
	"clear":            &CommandClear{},
	"history":          &CommandHistory{},
	"rollback":         &CommandRollback{},
	"translate":        &CommandTranslate{},
	"enhance":          &CommandEnhance{},
	"grammar":          &CommandGrammar{},
	"summarize":        &CommandSummarize{},
	"summarize_prompt": &CommandSummarizePrompt{},
	"analyze":          &CommandAnalyze{},
	"temperature":      &CommandTemperature{},
	"model":            &CommandModel{},
	"imagine":          &CommandImagine{},
	"system":           &CommandSystem{},
	"markdown":         &CommandMarkdown{},
}

type Command interface {
	Name() string
	Description() string
	Execute(bot *telegram.Bot, update telegram.Update, gptClient *gpt.GPTClient, chat *storage.Chat)
}

func gptText(bot *telegram.Bot, chat *storage.Chat, messageID int, gptClient *gpt.GPTClient, systemPrompt, userPrompt string) {
	responsePayload, err := gptClient.CallGPT([]gpt.Message{
		{Role: "system", Content: systemPrompt},
		{Role: "user", Content: userPrompt},
	}, chat.Settings.Model, 0.6)

	if err != nil {
		log.Printf("Error: %v", err)
		return
	}

	response := "Произошла ошибка с получением ответа, пожалуйста, попробуйте позднее"
	if len(responsePayload.Choices) > 0 {
		response = strings.TrimSpace(responsePayload.Choices[0].Message.Content)
	}

	log.Printf("[%s] %s", "ChatGPT", response)
	bot.Reply(chat.ChatID, messageID, response)
}

func summarizeText(bot *telegram.Bot, chat *storage.Chat, messageID int, gptClient *gpt.GPTClient, systemPrompt string, messageCount int) {
	// open log file
	lines, err := util.ReadLastLines(fmt.Sprintf("log/%d.log", chat.ChatID), messageCount)
	if err != nil {
		bot.Reply(chat.ChatID, messageID, "Произошла ошибка")
		return
	}

	if len(lines) == 0 {
		bot.Reply(chat.ChatID, messageID, "История чата пуста")
		return
	}

	bot.Reply(chat.ChatID, messageID, fmt.Sprintf("Обработка %d сообщений...", len(lines)))
	chatLog := strings.Join(lines, "\n")
	gptText(bot, chat, messageID, gptClient, systemPrompt, "Вот сообщения чата, которые ты должен обработать:\n\n"+chatLog)
}
