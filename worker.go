package main

import (
	"GPTBot/api/gpt"
	"GPTBot/api/telegram"
	"GPTBot/util"
	"fmt"
	"log"
	"strings"
)

// worker function that processes updates
func worker(updateChan <-chan telegram.Update, bot *telegram.Bot, gptClient *gpt.GPTClient, config *Config) {
	for update := range updateChan {
		processUpdate(bot, update, gptClient, config)
	}
}

func processUpdate(bot *telegram.Bot, update telegram.Update, gptClient *gpt.GPTClient, config *Config) {
	chatID := update.Message.Chat.ID
	fromID := update.Message.From.ID

	// Check for command
	if update.Message.IsCommand() {
		command := update.Message.Command()
		switch command {
		case "start":
			commandStart(bot, update, chatID)
		case "clear":
			commandClear(bot, update, chatID)
		case "history":
			commandHistory(bot, update, chatID)
		case "rollback":
			commandRollback(bot, update, chatID)
		case "help":
			commandHelp(bot, update, chatID)
		case "translate":
			commandTranslate(bot, update, gptClient, chatID)
		case "grammar":
			commandGrammar(bot, update, gptClient, chatID)
		case "enhance":
			commandEnhance(bot, update, gptClient, chatID)
		case "imagine":
			commandImagine(bot, update, gptClient, chatID, config)
		default:
			if fromID != config.AdminId {
				bot.Reply(chatID, update.Message.MessageID, fmt.Sprintf("Неизвестная команда /%s", command))
				break
			}

			switch command {
			case "reload":
				commandReload(bot, update, chatID)
			case "adduser":
				commandAddUser(bot, update, chatID, config)
			case "removeuser":
				commandRemoveUser(bot, update, chatID, config)
			}
		}

		return
	}

	gptChat(bot, update, gptClient, config, chatID, fromID)
}

func gptText(bot *telegram.Bot, chatID int64, messageID int, gptClient *gpt.GPTClient, systemPrompt, userPrompt string) {
	responsePayload, err := gptClient.CallGPT35([]gpt.Message{
		{Role: "system", Content: systemPrompt},
		{Role: "user", Content: userPrompt},
	}, "gpt-3.5-turbo", 0.6)

	if err != nil {
		log.Printf("Error: %v", err)
		return
	}

	response := "I'm sorry, there was a problem. You can try again."
	if len(responsePayload.Choices) > 0 {
		response = strings.TrimSpace(responsePayload.Choices[0].Message.Content)
	}

	log.Printf("[%s] %s", "ChatGPT", response)
	bot.Reply(chatID, messageID, response)
}

func gptImage(bot *telegram.Bot, chatID int64, gptClient *gpt.GPTClient, prompt string, config *Config) {
	imageUrl, err := gptClient.GenerateImage(prompt, gpt.ImageSize512)
	if err != nil {
		log.Printf("Error generating image: %v", err)
		return
	}

	enhancedCaption := prompt
	responsePayload, err := gptClient.CallGPT35([]gpt.Message{
		{Role: "system", Content: "You are an assistant that generates natural language description (prompt) for an artificial intelligence (AI) that generates images"},
		{Role: "user", Content: fmt.Sprintf("Please improve this prompt: \"%s\". Answer with improved prompt only. Keep prompt at most 200 characters long. Your prompt must be in one sentence.", prompt)},
	}, "gpt-3.5-turbo", 0.7)
	if err == nil {
		enhancedCaption = strings.TrimSpace(responsePayload.Choices[0].Message.Content)
	}

	err = bot.SendImage(chatID, imageUrl, enhancedCaption)
	if err != nil {
		log.Printf("Error sending image: %v", err)
		return
	}

	log.Printf("[ChatGPT] sent image %s", imageUrl)
	if config.AdminId > 0 {
		if chatID != config.AdminId {
			bot.Message(fmt.Sprintf("Image with prompt \"%s\" sent to chat %d", prompt, chatID), config.AdminId)
		}
	}
}

func gptChat(bot *telegram.Bot, update telegram.Update, gptClient *gpt.GPTClient, config *Config, chatID int64, fromID int64) {
	log.Printf("[%s] %s", update.Message.From.UserName, update.Message.Text)

	if update.Message.Chat.IsGroup() {
		isReplyToBot := update.Message.ReplyToMessage != nil && update.Message.ReplyToMessage.From.UserName == bot.Username
		if !strings.Contains(update.Message.Text, "@"+bot.Username) && !isReplyToBot {
			return
		}

		if strings.Contains(update.Message.Text, "@"+bot.Username) {
			update.Message.Text = strings.Replace(update.Message.Text, "@"+bot.Username, "", -1)
		}
	}

	// Maintain conversation history
	userMessage := gpt.Message{Role: "user", Content: update.Message.Text}
	historyEntry := &ConversationEntry{Prompt: userMessage, Response: gpt.Message{}}
	chatHistory[chatID] = append(chatHistory[chatID], historyEntry)
	if len(chatHistory[chatID]) > config.MaxMessages {
		excessMessages := len(chatHistory[chatID]) - config.MaxMessages
		chatHistory[chatID] = chatHistory[chatID][excessMessages:]
	}

	responsePayload, err := gptClient.CallGPT35(messagesFromHistory(chatHistory[chatID]), "gpt-3.5-turbo", 0.8)
	if err != nil {
		log.Printf("Error: %v", err)
		return
	}

	response := "I'm sorry, there was a problem in answering. You can try again"
	if len(responsePayload.Choices) > 0 {
		response = strings.TrimSpace(responsePayload.Choices[0].Message.Content)
	}

	// Add the assistant's response to the conversation history
	historyEntry.Response = gpt.Message{Role: "assistant", Content: response}

	log.Printf("[%s] %s", "ChatGPT", response)
	bot.Reply(chatID, update.Message.MessageID, response)

	if config.AdminId == 0 {
		return
	}

	if fromID == config.AdminId {
		return
	}

	if util.IsIdInList(fromID, config.IgnoreReportIds) {
		return
	}

	bot.Message(fmt.Sprintf("[User: %s %s (%s, ID: %d)] %s\n[ChatGPT] %s\n", update.Message.From.FirstName, update.Message.From.LastName, update.Message.From.UserName, update.Message.From.ID, update.Message.Text, response), config.AdminId)
}
