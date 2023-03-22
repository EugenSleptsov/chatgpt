package main

import (
	"GPTBot/api/gpt"
	"GPTBot/api/telegram"
	"bufio"
	"fmt"
	"log"
	"os"
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
		go processUpdate(bot, update, gptClient, config) // Launch a goroutine for each update
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
		formattedLine := fmt.Sprintf("%d. %s: %s\n", i+1, Title(message.Role), message.Content)
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

func Title(s string) string {
	if len(s) == 0 {
		return s
	}

	r := []rune(s)
	if r[0] >= 'a' && r[0] <= 'z' {
		r[0] = r[0] - 'a' + 'A'
	}

	return string(r)
}

func translateText(bot *telegram.Bot, chatID int64, messageID int, gptClient *gpt.GPTClient, prompt string) {
	translationPrompt := fmt.Sprintf("Translate the following text to English: \"%s\". You should answer only with translated text without explanations and quotation marks	", prompt)

	responsePayload, err := gptClient.CallGPT35([]gpt.Message{
		{Role: "system", Content: "You are a helpful assistant that translates."},
		{Role: "user", Content: translationPrompt},
	})

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
	if update.Message == nil {
		return
	}

	chatID := update.Message.Chat.ID
	fromID := update.Message.From.ID

	if !isUserAuthorized(fromID, config.AuthorizedUserIds) {
		bot.Answer(chatID, update.Message.MessageID, "Sorry, you do not have access to this bot.")
		log.Printf("Unauthorized access attempt by user %d: %s %s (%s)", fromID, update.Message.From.FirstName, update.Message.From.LastName, update.Message.From.UserName)

		// Notify the admin
		if config.AdminId > 0 {
			adminMessage := fmt.Sprintf("Unauthorized access attempt by user %d: %s %s (%s)", fromID, update.Message.From.FirstName, update.Message.From.LastName, update.Message.From.UserName)
			bot.Admin(adminMessage, config.AdminId)
		}
		return
	}

	// Check for commands
	if update.Message.IsCommand() {
		command := update.Message.Command()
		switch command {
		case "start":
			bot.Answer(chatID, update.Message.MessageID, "Здравствуйте! Я помощник GPT-3.5 Turbo, и я здесь, чтобы помочь вам с любыми вопросами или задачами. Просто напишите ваш вопрос или запрос, и я сделаю все возможное, чтобы помочь вам! Для справки наберите /help")
		case "clear":
			chatHistory[chatID] = nil
			bot.Answer(chatID, update.Message.MessageID, "История разговоров была очищена.")
		case "history":
			historyMessages := formatHistory(chatHistory[chatID])
			for _, message := range historyMessages {
				bot.Answer(chatID, update.Message.MessageID, message)
			}
		case "help":
			helpText := `Список доступных команд и их описание:
/help - Показывает список доступных команд и их описание.
/start - Отправляет приветственное сообщение, описывающее цель бота.
/clear - Очищает историю разговоров для текущего чата.
/history - Показывает всю сохраненную на данный момент историю разговоров в красивом форматировании.
/translate <text> - Переводит <text> на любом языке на английский язык`
			bot.Answer(chatID, update.Message.MessageID, helpText)

		case "translate":
			if len(update.Message.CommandArguments()) == 0 {
				bot.Answer(chatID, update.Message.MessageID, "Please provide a text to translate. Usage: /translate <text>")
			} else {
				translateText(bot, chatID, update.Message.MessageID, gptClient, update.Message.CommandArguments())
			}

		default:
			if fromID == config.AdminId {
				switch command {
				case "reload":
					config, err := readConfig("bot.conf")
					if err != nil {
						log.Fatalf("Error reading bot.conf: %v", err)
					}

					bot.Answer(chatID, update.Message.MessageID, fmt.Sprintf("Config updated: %s", fmt.Sprint(config)))

				case "adduser":
					if len(update.Message.CommandArguments()) == 0 {
						bot.Answer(chatID, update.Message.MessageID, "Please provide a user id to add")
					} else {
						userId, err := strconv.Atoi(update.Message.CommandArguments())
						if err != nil {
							bot.Answer(chatID, update.Message.MessageID, fmt.Sprintf("Invalid user id: %s", update.Message.CommandArguments()))
							break
						}

						for _, auth := range config.AuthorizedUserIds {
							if auth == userId {
								bot.Answer(chatID, update.Message.MessageID, fmt.Sprintf("User already added: %d", userId))
								break
							}
						}

						config.AuthorizedUserIds = append(config.AuthorizedUserIds, userId)
						err = updateConfig("bot.conf", config)
						if err != nil {
							log.Fatalf("Error updating bot.conf: %v", err)
						}

						bot.Answer(chatID, update.Message.MessageID, fmt.Sprintf("User successfully added: %d", userId))
					}

				case "removeuser":
					if len(update.Message.CommandArguments()) == 0 {
						bot.Answer(chatID, update.Message.MessageID, "Please provide a user id to remove")
					} else {
						userId, err := strconv.Atoi(update.Message.CommandArguments())
						if err != nil {
							bot.Answer(chatID, update.Message.MessageID, fmt.Sprintf("Invalid user id: %s", update.Message.CommandArguments()))
							break
						}

						newList := make([]int, 0)
						for _, auth := range config.AuthorizedUserIds {
							if auth == userId {
								bot.Answer(chatID, update.Message.MessageID, fmt.Sprintf("User will be removed: %d", userId))
								break
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
			} else {
				bot.Answer(chatID, update.Message.MessageID, fmt.Sprintf("Неизвестная команда /%s", command))
			}
		}

		return
	}

	log.Printf("[%s] %s", update.Message.From.UserName, update.Message.Text)

	// Maintain conversation history
	chatHistory[chatID] = append(chatHistory[chatID], gpt.Message{Role: "user", Content: update.Message.Text})
	if len(chatHistory[chatID]) > config.MaxMessages {
		excessMessages := len(chatHistory[chatID]) - config.MaxMessages
		chatHistory[chatID] = chatHistory[chatID][excessMessages:]
	}

	responsePayload, err := gptClient.CallGPT35(chatHistory[chatID])
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
	if config.AdminId > 0 {
		if fromID != config.AdminId {
			var adminMessage string
			if !isIDInList(fromID, config.IgnoreReportIds) {
				adminMessage = fmt.Sprintf("[User: %s %s (%s, ID: %d)] %s\n[ChatGPT] %s\n",
					update.Message.From.FirstName, update.Message.From.LastName, update.Message.From.UserName, update.Message.From.ID, update.Message.Text,
					response)
			} else {
				adminMessage = fmt.Sprintf("[User: %s %s (%s, ID: %d)] asked ChatGPT",
					update.Message.From.FirstName, update.Message.From.LastName, update.Message.From.UserName, update.Message.From.ID)
			}
			bot.Admin(adminMessage, config.AdminId)
		}
	}
}

// Helper function to check if an ID is in a list of IDs
func isIDInList(id int, idList []int) bool {
	for _, listID := range idList {
		if id == listID {
			return true
		}
	}
	return false
}

func isUserAuthorized(userID int, authorizedUsers []int) bool {
	// If no authorized users are provided, make the bot public
	if len(authorizedUsers) == 0 {
		return true
	}

	// Check if the user is in the list of authorized users
	for _, authorizedUser := range authorizedUsers {
		if userID == authorizedUser {
			return true
		}
	}
	return false
}

func readConfig(filename string) (*Config, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	config := make(map[string]string)
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
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
	}, scanner.Err()
}

func updateConfig(filename string, config *Config) error {
	file, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	var lines []string
	authorizedUsersLine := fmt.Sprintf("authorized_user_ids=%s", strings.Join(strings.Split(strings.Trim(strings.Trim(fmt.Sprint(config.AuthorizedUserIds), "[]"), " "), " "), ","))

	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(strings.TrimSpace(line), "authorized_user_ids") {
			lines = append(lines, authorizedUsersLine)
		} else {
			lines = append(lines, line)
		}
	}

	if scanner.Err() != nil {
		return scanner.Err()
	}

	outputFile, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer outputFile.Close()

	writer := bufio.NewWriter(outputFile)
	for _, line := range lines {
		_, err := writer.WriteString(line + "\n")
		if err != nil {
			return err
		}
	}

	err = writer.Flush()
	if err != nil {
		return err
	}

	return nil
}
