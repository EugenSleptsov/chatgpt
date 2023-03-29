package main

import (
	"GPTBot/api/gpt"
	"GPTBot/api/telegram"
	"GPTBot/util"
	"fmt"
	"log"
	"strconv"
	"strings"
)

var chatHistory = make(map[int64][]gpt.Message)

type Config struct {
	TelegramToken     string
	GPTToken          string
	TimeoutValue      int
	MaxMessages       int
	AdminId           int
	IgnoreReportIds   []int
	AuthorizedUserIds []int
}

func (c *Config) String() string {
	return fmt.Sprintf("Config{\n  TelegramToken: %s,\n  GPTToken: %s,\n  TimeoutValue: %d,\n  MaxMessages: %d,\n  AdminId: %d,\n  IgnoreReportIds: %v,\n  AuthorizedUserIds: %v,\n}",
		c.TelegramToken, c.GPTToken, c.TimeoutValue, c.MaxMessages, c.AdminId, c.IgnoreReportIds, c.AuthorizedUserIds)
}

func main() {
	config, err := readConfig("bot.conf")
	if err != nil {
		log.Fatalf("Error reading bot.conf: %v", err)
	}

	bot, err := telegram.NewBot(config.TelegramToken)
	if err != nil {
		log.Fatal(err)
	}

	gptClient := &gpt.GPTClient{
		ApiKey: config.GPTToken,
	}

	for update := range bot.GetUpdateChannel(config.TimeoutValue) {
		// Ignore any non-Message Updates
		if update.Message == nil {
			continue
		}

		// If no authorized users are provided, make the bot public
		if len(config.AuthorizedUserIds) > 0 {
			if !util.IsIdInList(update.Message.From.ID, config.AuthorizedUserIds) {
				bot.Answer(update.Message.Chat.ID, update.Message.MessageID, "Sorry, you do not have access to this bot.")
				log.Printf("Unauthorized access attempt by user %d: %s %s (%s)", update.Message.From.ID, update.Message.From.FirstName, update.Message.From.LastName, update.Message.From.UserName)

				// Notify the admin
				if config.AdminId > 0 {
					adminMessage := fmt.Sprintf("Unauthorized access attempt by user %d: %s %s (%s)", update.Message.From.ID, update.Message.From.FirstName, update.Message.From.LastName, update.Message.From.UserName)
					bot.Admin(adminMessage, config.AdminId)
				}
				continue
			}
		}

		// Launch a goroutine for each update
		go processUpdate(bot, update, gptClient, config)
	}
}

func formatHistory(history []gpt.Message) []string {
	if len(history) == 0 {
		return []string{"История разговоров пуста."}
	}

	var historyMessage string
	var historyMessages []string
	characterCount := 0

	for i, message := range history {
		formattedLine := fmt.Sprintf("%d. %s: %s\n", i+1, util.Title(message.Role), message.Content)
		lineLength := len(formattedLine)

		if characterCount+lineLength > 4096 {
			historyMessages = append(historyMessages, historyMessage)
			historyMessage = ""
			characterCount = 0
		}

		historyMessage += formattedLine
		characterCount += lineLength
	}

	if len(historyMessage) > 0 {
		historyMessages = append(historyMessages, historyMessage)
	}

	return historyMessages
}

func translateText(bot *telegram.Bot, chatID int64, messageID int, gptClient *gpt.GPTClient, prompt string) {
	translationPrompt := fmt.Sprintf("Translate the following text to English: \"%s\". You should answer only with translated text without explanations and quotation marks	", prompt)

	responsePayload, err := gptClient.CallGPT35([]gpt.Message{
		{Role: "system", Content: "You are a helpful assistant that translates."},
		{Role: "user", Content: translationPrompt},
	}, "gpt-3.5-turbo", 0.6)

	if err != nil {
		log.Printf("Error: %v", err)
		return
	}

	response := "I'm sorry, there was a problem translating your text. You can try again."
	if len(responsePayload.Choices) > 0 {
		response = strings.TrimSpace(responsePayload.Choices[0].Message.Content)
	}

	log.Printf("[%s] %s", "ChatGPT", response)
	bot.Answer(chatID, messageID, response)
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
		case "help":
			commandHelp(bot, update, chatID)
		case "translate":
			commandTranslate(bot, update, gptClient, chatID)
		default:
			if fromID != config.AdminId {
				bot.Answer(chatID, update.Message.MessageID, fmt.Sprintf("Неизвестная команда /%s", command))
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

	handleMessage(bot, update, gptClient, config, chatID, fromID)
}

func handleMessage(bot *telegram.Bot, update telegram.Update, gptClient *gpt.GPTClient, config *Config, chatID int64, fromID int) {
	log.Printf("[%s] %s", update.Message.From.UserName, update.Message.Text)

	// Maintain conversation history
	chatHistory[chatID] = append(chatHistory[chatID], gpt.Message{Role: "user", Content: update.Message.Text})
	if len(chatHistory[chatID]) > config.MaxMessages {
		excessMessages := len(chatHistory[chatID]) - config.MaxMessages
		chatHistory[chatID] = chatHistory[chatID][excessMessages:]
	}

	responsePayload, err := gptClient.CallGPT35(chatHistory[chatID], "gpt-3.5-turbo", 0.8)
	if err != nil {
		log.Printf("Error: %v", err)
		return
	}

	response := "I'm sorry, there was a problem in answering. You can try again"
	if len(responsePayload.Choices) > 0 {
		response = strings.TrimSpace(responsePayload.Choices[0].Message.Content)
	}

	// Add the assistant's response to the conversation history
	chatHistory[chatID] = append(chatHistory[chatID], gpt.Message{Role: "assistant", Content: response})

	log.Printf("[%s] %s", "ChatGPT", response)
	bot.Answer(chatID, update.Message.MessageID, response)

	if config.AdminId == 0 {
		return
	}

	if fromID == config.AdminId {
		return
	}

	var adminMessage string
	if !util.IsIdInList(fromID, config.IgnoreReportIds) {
		adminMessage = fmt.Sprintf("[User: %s %s (%s, ID: %d)] %s\n[ChatGPT] %s\n",
			update.Message.From.FirstName, update.Message.From.LastName, update.Message.From.UserName, update.Message.From.ID, update.Message.Text,
			response)
	} else {
		adminMessage = fmt.Sprintf("[User: %s %s (%s, ID: %d)] asked ChatGPT",
			update.Message.From.FirstName, update.Message.From.LastName, update.Message.From.UserName, update.Message.From.ID)
	}
	bot.Admin(adminMessage, config.AdminId)
}

func commandRemoveUser(bot *telegram.Bot, update telegram.Update, chatID int64, config *Config) {
	if len(update.Message.CommandArguments()) == 0 {
		bot.Answer(chatID, update.Message.MessageID, "Please provide a user id to remove")
	} else {
		userId, err := strconv.Atoi(update.Message.CommandArguments())
		if err != nil {
			bot.Answer(chatID, update.Message.MessageID, fmt.Sprintf("Invalid user id: %s", update.Message.CommandArguments()))
			return
		}

		newList := make([]int, 0)
		for _, auth := range config.AuthorizedUserIds {
			if auth == userId {
				bot.Answer(chatID, update.Message.MessageID, fmt.Sprintf("User will be removed: %d", userId))
			} else {
				newList = append(newList, auth)
			}
		}

		config.AuthorizedUserIds = newList
		err = updateConfig("bot.conf", config)
		if err != nil {
			log.Fatalf("Error updating bot.conf: %v", err)
		}

		bot.Answer(chatID, update.Message.MessageID, "Command successfully ended")
	}
}

func commandAddUser(bot *telegram.Bot, update telegram.Update, chatID int64, config *Config) {
	if len(update.Message.CommandArguments()) == 0 {
		bot.Answer(chatID, update.Message.MessageID, "Please provide a user id to add")
	} else {
		userId, err := strconv.Atoi(update.Message.CommandArguments())
		if err != nil {
			bot.Answer(chatID, update.Message.MessageID, fmt.Sprintf("Invalid user id: %s", update.Message.CommandArguments()))
			return
		}

		for _, auth := range config.AuthorizedUserIds {
			if auth == userId {
				bot.Answer(chatID, update.Message.MessageID, fmt.Sprintf("User already added: %d", userId))
				return
			}
		}

		config.AuthorizedUserIds = append(config.AuthorizedUserIds, userId)
		err = updateConfig("bot.conf", config)
		if err != nil {
			log.Fatalf("Error updating bot.conf: %v", err)
		}

		bot.Answer(chatID, update.Message.MessageID, fmt.Sprintf("User successfully added: %d", userId))
	}
}

func commandReload(bot *telegram.Bot, update telegram.Update, chatID int64) {
	config, err := readConfig("bot.conf")
	if err != nil {
		log.Fatalf("Error reading bot.conf: %v", err)
	}

	bot.Answer(chatID, update.Message.MessageID, fmt.Sprintf("Config updated: %s", fmt.Sprint(config)))
}

func commandTranslate(bot *telegram.Bot, update telegram.Update, gptClient *gpt.GPTClient, chatID int64) {
	if len(update.Message.CommandArguments()) == 0 {
		bot.Answer(chatID, update.Message.MessageID, "Please provide a text to translate. Usage: /translate <text>")
	} else {
		translateText(bot, chatID, update.Message.MessageID, gptClient, update.Message.CommandArguments())
	}
}

func commandHelp(bot *telegram.Bot, update telegram.Update, chatID int64) {
	helpText := `Список доступных команд и их описание:
/help - Показывает список доступных команд и их описание.
/start - Отправляет приветственное сообщение, описывающее цель бота.
/clear - Очищает историю разговоров для текущего чата.
/history - Показывает всю сохраненную на данный момент историю разговоров в красивом форматировании.
/translate <text> - Переводит <text> на любом языке на английский язык`
	bot.Answer(chatID, update.Message.MessageID, helpText)
}

func commandHistory(bot *telegram.Bot, update telegram.Update, chatID int64) {
	historyMessages := formatHistory(chatHistory[chatID])
	for _, message := range historyMessages {
		bot.Answer(chatID, update.Message.MessageID, message)
	}
}

func commandStart(bot *telegram.Bot, update telegram.Update, chatID int64) {
	bot.Answer(chatID, update.Message.MessageID, "Здравствуйте! Я помощник GPT-3.5 Turbo, и я здесь, чтобы помочь вам с любыми вопросами или задачами. Просто напишите ваш вопрос или запрос, и я сделаю все возможное, чтобы помочь вам! Для справки наберите /help")
}

func commandClear(bot *telegram.Bot, update telegram.Update, chatID int64) {
	chatHistory[chatID] = nil
	bot.Answer(chatID, update.Message.MessageID, "История разговоров была очищена.")
}

func readConfig(filename string) (*Config, error) {
	config := make(map[string]string)
	lines, err := util.ReadLines(filename)
	if err != nil {
		return nil, err
	}

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "#") {
			continue // Ignore comment lines
		}

		parts := strings.SplitN(line, "=", 2)
		if len(parts) == 2 {
			config[strings.TrimSpace(parts[0])] = strings.TrimSpace(parts[1])
		}
	}

	timeoutValue, err := strconv.Atoi(config["timeout_value"])
	if err != nil {
		log.Fatalf("Error converting timeout_value to integer: %v", err)
	}
	maxMessages, err := strconv.Atoi(config["max_messages"])
	if err != nil {
		log.Fatalf("Error converting max_messages to integer: %v", err)
	}

	var adminID int
	if config["admin_id"] != "" {
		adminID, err = strconv.Atoi(config["admin_id"])
		if err != nil {
			log.Fatalf("Error converting admin_id to integer: %v", err)
		}
	}

	ignoreReportIds := make([]int, 0)
	if config["ignore_report_ids"] != "" {
		ids := strings.Split(config["ignore_report_ids"], ",")
		for _, id := range ids {
			parsedID, err := strconv.Atoi(strings.TrimSpace(id))
			if err != nil {
				log.Fatalf("Error converting ignore_report_ids to integer: %v", err)
			}
			ignoreReportIds = append(ignoreReportIds, parsedID)
		}
	}

	authorizedUsersRaw := strings.Split(config["authorized_user_ids"], ",")
	var authorizedUserIDs []int
	for _, idStr := range authorizedUsersRaw {
		id, err := strconv.Atoi(strings.TrimSpace(idStr))
		if err == nil {
			authorizedUserIDs = append(authorizedUserIDs, id)
		}
	}

	return &Config{
		TelegramToken:     config["telegram_token"],
		GPTToken:          config["gpt_token"],
		TimeoutValue:      timeoutValue,
		MaxMessages:       maxMessages,
		AdminId:           adminID,
		IgnoreReportIds:   ignoreReportIds,
		AuthorizedUserIds: authorizedUserIDs,
	}, nil
}

func updateConfig(filename string, config *Config) error {
	oldLines, err := util.ReadLines(filename)
	if err != nil {
		return err
	}

	var lines []string
	authorizedUsersLine := fmt.Sprintf("authorized_user_ids=%s", strings.Join(strings.Split(strings.Trim(strings.Trim(fmt.Sprint(config.AuthorizedUserIds), "[]"), " "), " "), ","))

	for _, line := range oldLines {
		if strings.HasPrefix(strings.TrimSpace(line), "authorized_user_ids") {
			lines = append(lines, authorizedUsersLine)
		} else {
			lines = append(lines, line)
		}
	}

	return util.WriteLines(filename, lines)
}
