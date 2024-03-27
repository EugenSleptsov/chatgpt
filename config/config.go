package config

import (
	"GPTBot/util"
	"errors"
	"fmt"
	"log"
	"strconv"
	"strings"
)

type Config struct {
	TelegramToken       string
	GPTToken            string
	SummarizePrompt     string
	TimeoutValue        int
	MaxMessages         int
	AdminId             int64
	IgnoreReportIds     []int64
	AuthorizedUserIds   []int64
	CommandMenu         []string
	TelegramTokenLogBot string
}

func (c *Config) String() string {
	return fmt.Sprintf("Config{\n  TelegramToken: %s,\n  GPTToken: %s,\n  TimeoutValue: %d,\n  MaxMessages: %d,\n  AdminId: %d,\n  IgnoreReportIds: %v,\n  AuthorizedUserIds: %v,\n  CommandMenu: %v\n  SummarizePrompt: %s\n}", c.TelegramToken, c.GPTToken, c.TimeoutValue, c.MaxMessages, c.AdminId, c.IgnoreReportIds, c.AuthorizedUserIds, c.CommandMenu, c.SummarizePrompt)
}

func ReadConfig(filename string) (*Config, error) {
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
		return nil, errors.New(fmt.Sprintf("Error converting timeout_value to integer: %v", err))
	}
	maxMessages, err := strconv.Atoi(config["max_messages"])
	if err != nil {
		return nil, errors.New(fmt.Sprintf("Error converting max_messages to integer: %v", err))
	}

	var adminID int64
	adminID, err = strconv.ParseInt(config["admin_id"], 10, 64)
	if err != nil {
		adminID = 0
		log.Printf("Error converting admin_id to integer: %v", err)
	}

	ids := strings.Split(config["ignore_report_ids"], ",")
	var ignoreReportIds []int64
	for _, id := range ids {
		parsedID, err := strconv.ParseInt(strings.TrimSpace(id), 10, 64)
		if err == nil {
			ignoreReportIds = append(ignoreReportIds, parsedID)
		}
	}

	authorizedUsersRaw := strings.Split(config["authorized_user_ids"], ",")
	var authorizedUserIDs []int64
	for _, idStr := range authorizedUsersRaw {
		id, err := strconv.ParseInt(strings.TrimSpace(idStr), 10, 64)
		if err == nil {
			authorizedUserIDs = append(authorizedUserIDs, id)
		}
	}

	commandMenuRaw := strings.Split(config["command_menu"], ",")
	var commandMenu []string
	for _, command := range commandMenuRaw {
		commandMenu = append(commandMenu, command)
	}

	return &Config{
		TelegramToken:       config["telegram_token"],
		GPTToken:            config["gpt_token"],
		SummarizePrompt:     config["summarize_prompt"],
		TimeoutValue:        timeoutValue,
		MaxMessages:         maxMessages,
		AdminId:             adminID,
		IgnoreReportIds:     ignoreReportIds,
		AuthorizedUserIds:   authorizedUserIDs,
		CommandMenu:         commandMenu,
		TelegramTokenLogBot: config["telegram_token_log_bot"],
	}, nil
}

func UpdateConfig(filename string, config *Config) error {
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
