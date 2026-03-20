package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

type Config struct {
	TelegramToken       string   `yaml:"telegram_token"`
	GPTToken            string   `yaml:"gpt_token"`
	SummarizePrompt     string   `yaml:"summarize_prompt"`
	DefaultSystemPrompt string   `yaml:"default_system_prompt"`
	TimeoutValue        int      `yaml:"timeout_value"`
	MaxMessages         int      `yaml:"max_messages"`
	AdminId             int64    `yaml:"admin_id"`
	IgnoreReportIds     []int64  `yaml:"ignore_report_ids"`
	AuthorizedUserIds   []int64  `yaml:"authorized_user_ids"`
	CommandMenu         []string `yaml:"command_menu"`
	TelegramTokenLogBot string   `yaml:"telegram_token_log_bot"`
	DataDir             string   `yaml:"data_dir"`
	LogDir              string   `yaml:"log_dir"`
}

// Defaults fills zero-valued fields with sensible defaults.
func (c *Config) ApplyDefaults() {
	if c.DataDir == "" {
		c.DataDir = "data"
	}
	if c.LogDir == "" {
		c.LogDir = "log"
	}
}

// maskToken hides all but the first 4 characters of a token.
func maskToken(token string) string {
	if len(token) <= 4 {
		return "****"
	}
	return token[:4] + "****"
}

func (c *Config) String() string {
	return fmt.Sprintf(
		"Config{\n  TelegramToken: %s,\n  GPTToken: %s,\n  TimeoutValue: %d,\n  MaxMessages: %d,\n  AdminId: %d,\n  IgnoreReportIds: %v,\n  AuthorizedUserIds: %v,\n  CommandMenu: %v\n  SummarizePrompt: %s\n}",
		maskToken(c.TelegramToken), maskToken(c.GPTToken), c.TimeoutValue, c.MaxMessages, c.AdminId, c.IgnoreReportIds, c.AuthorizedUserIds, c.CommandMenu, c.SummarizePrompt,
	)
}

func ReadConfig(filename string) (*Config, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("reading config file: %w", err)
	}

	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("parsing config file: %w", err)
	}

	config.ApplyDefaults()
	return &config, nil
}

func UpdateConfig(filename string, config *Config) error {
	data, err := yaml.Marshal(config)
	if err != nil {
		return fmt.Errorf("marshaling config: %w", err)
	}

	return os.WriteFile(filename, data, 0644)
}
