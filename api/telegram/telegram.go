package telegram

import (
	"GPTBot/util"
	"fmt"
	"github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"io"
	"log"
	"net/http"
	"strings"
)

type Bot struct {
	api      *tgbotapi.BotAPI
	Username string
	Token    string
	AdminId  int64
	LogBot   *LogBot
}

type UpdatesChannel <-chan Update
type Update tgbotapi.Update

type Command string

const (
	CommandHelp      Command = "help"
	CommandHistory   Command = "history"
	CommandRollback  Command = "rollback"
	CommandClear     Command = "clear"
	CommandSummarize Command = "summarize"
)

var CommandDescriptions = map[Command]string{
	CommandHelp:      "Справка по командам",
	CommandHistory:   "Показать историю переписки",
	CommandRollback:  "Отменить последнее сообщение",
	CommandClear:     "Очистить историю переписки",
	CommandSummarize: "Суммаризировать историю переписки",
}

var DefaultCommandList = []Command{
	CommandHelp,
	CommandHistory,
	CommandRollback,
	CommandClear,
	CommandSummarize,
}

func NewInstance(token string, commandMenu []string, logbottoken string) (*Bot, error) {
	api, err := tgbotapi.NewBotAPI(token)
	if err != nil {
		return nil, err
	}

	bot := &Bot{
		api:      api,
		Username: api.Self.UserName,
		Token:    token,
	}

	bot.SetCommandList(commandMenu)

	log.Printf("Authorized on account %s", bot.api.Self.UserName)

	if logbottoken != "" {
		logBot, err := NewLogBot(logbottoken)
		if err != nil {
			return nil, err
		}
		bot.LogBot = logBot
	}

	return bot, nil
}

func (botInstance *Bot) SetCommandList(rawCommandMenu []string) {
	var commandMenu []Command
	for _, command := range rawCommandMenu {
		if _, ok := CommandDescriptions[Command(command)]; ok {
			commandMenu = append(commandMenu, Command(command))
		}
	}

	if len(commandMenu) > 0 {
		_ = botInstance._setCommandList(commandMenu...)
	} else {
		_ = botInstance._setCommandList(DefaultCommandList...)
	}
}

func (botInstance *Bot) _setCommandList(commands ...Command) error {
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

func (botInstance *Bot) ReplyMarkdown(chatID int64, replyTo int, text string) {
	botInstance.reply(chatID, replyTo, text, true)
}

func (botInstance *Bot) Reply(chatID int64, replyTo int, text string) {
	botInstance.reply(chatID, replyTo, text, false)
}

func (botInstance *Bot) Message(message string, chatID int64, isMarkdown bool) {
	botInstance._message(chatID, 0, message, isMarkdown)
}

func (botInstance *Bot) reply(chatID int64, replyTo int, text string, isMarkdown bool) {
	botInstance.message(chatID, replyTo, text, isMarkdown)
}

func (botInstance *Bot) message(chatID int64, replyTo int, text string, isMarkdown bool) {
	// split long messages
	for len(text) > 4096 {
		botInstance._message(chatID, replyTo, text[:4096], isMarkdown)
		text = text[4096:]
	}
	botInstance._message(chatID, replyTo, text, isMarkdown)
}

func (botInstance *Bot) _message(chatID int64, replyTo int, text string, isMarkdown bool) {
	msg := tgbotapi.NewMessage(chatID, text)
	if isMarkdown {
		msg.ParseMode = "MarkdownV2"
		msg.Text = util.FixMarkdown(escapeMarkdownV2(msg.Text))
	}
	if replyTo != 0 {
		msg.ReplyToMessageID = replyTo
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
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			return
		}
	}(response.Body)

	imageData, err := io.ReadAll(response.Body)
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

func (botInstance *Bot) GetUserCount(chatID int64) (int, error) {
	return botInstance.api.GetChatMembersCount(tgbotapi.ChatMemberCountConfig{ChatConfig: tgbotapi.ChatConfig{ChatID: chatID}})
}

func (botInstance *Bot) SetAdminId(adminId int64) {
	botInstance.AdminId = adminId
}

func (botInstance *Bot) GetFile(fileId string) (tgbotapi.File, error) {
	return botInstance.api.GetFile(tgbotapi.FileConfig{FileID: fileId})
}

func (botInstance *Bot) AudioUpload(chatID int64, bytes []byte) error {
	audioMsg := tgbotapi.NewAudio(chatID, tgbotapi.FileBytes{Name: "audio.ogg", Bytes: bytes})
	_, err := botInstance.api.Send(audioMsg)
	if err != nil {
		return err
	}

	return nil
}

func (botInstance *Bot) Log(message string) {
	log.Print(message)

	if botInstance.LogBot != nil && botInstance.AdminId != 0 {
		err := botInstance.LogBot.SendMessage(botInstance.AdminId, message)
		if err != nil {
			log.Printf("Error sending log message: %v", err)
		}
	}
}

func escapeMarkdownV2(text string) string {
	charsToEscape := []string{"_", "*", "[", "]", "(", ")", "~", ">", "#", "+", "-", "=", "|", "{", "}", ".", "!"}
	for _, char := range charsToEscape {
		text = strings.ReplaceAll(text, char, "\\"+char)
	}
	return text
}

func GetChatTitle(update Update) string {
	if update.Message.Chat.ID > 0 {
		return fmt.Sprintf("%s %s [@%s]", update.Message.Chat.FirstName, update.Message.Chat.LastName, update.Message.Chat.UserName)
	}

	return fmt.Sprintf("Chat %d [%s]", update.Message.Chat.ID, update.Message.Chat.Title)
}
