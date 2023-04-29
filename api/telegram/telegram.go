package telegram

import (
	"GPTBot/util"
	"github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"io/ioutil"
	"log"
	"net/http"
	"strings"
)

type Bot struct {
	api      *tgbotapi.BotAPI
	Username string
}

type UpdatesChannel <-chan Update
type Update tgbotapi.Update

type Command string

const (
	CommandHelp     Command = "help"
	CommandHistory  Command = "history"
	CommandRollback Command = "rollback"
	CommandClear    Command = "clear"
)

var CommandDescriptions = map[Command]string{
	CommandHelp:     "Справка по командам",
	CommandHistory:  "Показать историю переписки",
	CommandRollback: "Отменить последнее сообщение",
	CommandClear:    "Очистить историю переписки",
}

var DefaultCommandList = []Command{
	CommandHelp,
	CommandHistory,
	CommandRollback,
	CommandClear,
}

func NewBot(token string) (*Bot, error) {
	api, err := tgbotapi.NewBotAPI(token)
	if err != nil {
		return nil, err
	}

	bot := &Bot{
		api:      api,
		Username: api.Self.UserName,
	}

	log.Printf("Authorized on account %s", bot.api.Self.UserName)
	return bot, nil
}

func (botInstance *Bot) SetCommandList(commands ...Command) error {
	var tgCommands []tgbotapi.BotCommand
	for _, command := range commands {
		tgCommands = append(tgCommands, tgbotapi.BotCommand{Command: string(command), Description: CommandDescriptions[command]})
	}

	_, err := botInstance.api.Request(tgbotapi.NewSetMyCommands(tgCommands...))
	return err
}

func (botInstance *Bot) GetUpdateChannel(timeout int) UpdatesChannel {
	botInstance.api.Debug = false

	updateConfig := tgbotapi.NewUpdate(0)
	updateConfig.Timeout = timeout

	updates := botInstance.api.GetUpdatesChan(updateConfig)

	ourChannel := make(chan Update)
	go func(channel tgbotapi.UpdatesChannel) {
		defer close(ourChannel)
		for update := range channel {
			ourChannel <- Update(update)
		}
	}(updates)

	return ourChannel
}

func (botInstance *Bot) Reply(chatID int64, replyTo int, text string, isMarkdown bool) {
	msg := tgbotapi.NewMessage(chatID, text)
	if isMarkdown {
		msg.ParseMode = "MarkdownV2"
		msg.Text = util.FixMarkdown(escapeMarkdownV2(msg.Text))
	}
	msg.ReplyToMessageID = replyTo
	_, err := botInstance.api.Send(msg)
	if err != nil {
		log.Printf("Error sending message: %v", err)
	}
}

func (botInstance *Bot) Message(message string, adminId int64, isMarkdown bool) {
	msg := tgbotapi.NewMessage(adminId, message)
	if isMarkdown {
		msg.ParseMode = "MarkdownV2"
		msg.Text = util.FixMarkdown(escapeMarkdownV2(msg.Text))
	}
	_, err := botInstance.api.Send(msg)
	if err != nil {
		log.Printf("Error sending message: %v", err)
	}
}

func (botInstance *Bot) SendImage(chatID int64, imageUrl string, caption string) error {
	response, err := http.Get(imageUrl)
	if err != nil {
		return err
	}
	defer response.Body.Close()

	imageData, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return err
	}

	photoMsg := tgbotapi.NewPhoto(chatID, tgbotapi.FileBytes{Name: "image.png", Bytes: imageData})
	photoMsg.Caption = caption
	_, err = botInstance.api.Send(photoMsg)
	if err != nil {
		return err
	}

	return nil
}

func escapeMarkdownV2(text string) string {
	charsToEscape := []string{"_", "*", "[", "]", "(", ")", "~", ">", "#", "+", "-", "=", "|", "{", "}", ".", "!"}
	for _, char := range charsToEscape {
		text = strings.ReplaceAll(text, char, "\\"+char)
	}
	return text
}
