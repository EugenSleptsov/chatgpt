package main

import (
	"GPTBot/util"
	"fmt"
	"log"
	"strconv"
	"strings"
)

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
