package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// --- ReadConfig ---

func TestReadConfig_ValidFile(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "bot.yaml")
	yaml := `
telegram_token: "tok_telegram"
gpt_token: "tok_gpt"
timeout_value: 5
admin_id: 111
authorized_user_ids: [111, 222]
summarize_prompt: "sum it up"
data_dir: "mydata"
log_dir: "mylog"
`
	if err := os.WriteFile(file, []byte(yaml), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := ReadConfig(file)
	if err != nil {
		t.Fatalf("ReadConfig: %v", err)
	}

	if cfg.TelegramToken != "tok_telegram" {
		t.Errorf("TelegramToken = %q", cfg.TelegramToken)
	}
	if cfg.GPTToken != "tok_gpt" {
		t.Errorf("GPTToken = %q", cfg.GPTToken)
	}
	if cfg.TimeoutValue != 5 {
		t.Errorf("TimeoutValue = %d", cfg.TimeoutValue)
	}
	if cfg.AdminId != 111 {
		t.Errorf("AdminId = %d", cfg.AdminId)
	}
	if len(cfg.AuthorizedUserIds) != 2 || cfg.AuthorizedUserIds[0] != 111 || cfg.AuthorizedUserIds[1] != 222 {
		t.Errorf("AuthorizedUserIds = %v", cfg.AuthorizedUserIds)
	}
	if cfg.SummarizePrompt != "sum it up" {
		t.Errorf("SummarizePrompt = %q", cfg.SummarizePrompt)
	}
	if cfg.DataDir != "mydata" {
		t.Errorf("DataDir = %q", cfg.DataDir)
	}
	if cfg.LogDir != "mylog" {
		t.Errorf("LogDir = %q", cfg.LogDir)
	}
}

func TestReadConfig_MissingFile(t *testing.T) {
	_, err := ReadConfig("/nonexistent/path/bot.yaml")
	if err == nil {
		t.Fatal("expected error for missing file")
	}
}

func TestReadConfig_InvalidYAML(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "bad.yaml")
	if err := os.WriteFile(file, []byte(":::bad"), 0644); err != nil {
		t.Fatal(err)
	}
	_, err := ReadConfig(file)
	if err == nil {
		t.Fatal("expected error for invalid YAML")
	}
}

// --- ApplyDefaults ---

func TestApplyDefaults_FillsEmpty(t *testing.T) {
	cfg := &Config{}
	cfg.ApplyDefaults()
	if cfg.DataDir != "_var/data" {
		t.Errorf("DataDir = %q, want '_var/data'", cfg.DataDir)
	}
	if cfg.LogDir != "_var/log" {
		t.Errorf("LogDir = %q, want '_var/log'", cfg.LogDir)
	}
}

func TestApplyDefaults_PreservesExisting(t *testing.T) {
	cfg := &Config{DataDir: "custom_data", LogDir: "custom_log"}
	cfg.ApplyDefaults()
	if cfg.DataDir != "custom_data" {
		t.Errorf("DataDir = %q, want 'custom_data'", cfg.DataDir)
	}
	if cfg.LogDir != "custom_log" {
		t.Errorf("LogDir = %q, want 'custom_log'", cfg.LogDir)
	}
}

func TestReadConfig_AppliesDefaults(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "minimal.yaml")
	if err := os.WriteFile(file, []byte("timeout_value: 1\n"), 0644); err != nil {
		t.Fatal(err)
	}
	cfg, err := ReadConfig(file)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.DataDir != "_var/data" || cfg.LogDir != "_var/log" {
		t.Errorf("defaults not applied: DataDir=%q LogDir=%q", cfg.DataDir, cfg.LogDir)
	}
}

// --- UpdateConfig ---

func TestUpdateConfig_RoundTrip(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "out.yaml")

	original := &Config{
		TelegramToken:     "tg_token_123",
		GPTToken:          "gpt_token_456",
		TimeoutValue:      10,
		AdminId:           999,
		AuthorizedUserIds: []int64{100, 200},
		DataDir:           "d",
		LogDir:            "l",
	}

	if err := UpdateConfig(file, original); err != nil {
		t.Fatalf("UpdateConfig: %v", err)
	}

	loaded, err := ReadConfig(file)
	if err != nil {
		t.Fatalf("ReadConfig after UpdateConfig: %v", err)
	}

	if loaded.TelegramToken != original.TelegramToken {
		t.Errorf("TelegramToken mismatch: %q vs %q", loaded.TelegramToken, original.TelegramToken)
	}
	if loaded.GPTToken != original.GPTToken {
		t.Errorf("GPTToken mismatch")
	}
	if loaded.TimeoutValue != original.TimeoutValue {
		t.Errorf("TimeoutValue mismatch")
	}
	if loaded.AdminId != original.AdminId {
		t.Errorf("AdminId mismatch")
	}
	if len(loaded.AuthorizedUserIds) != 2 {
		t.Errorf("AuthorizedUserIds length mismatch")
	}
}

func TestUpdateConfig_InvalidPath(t *testing.T) {
	err := UpdateConfig("/nonexistent/dir/file.yaml", &Config{})
	if err == nil {
		t.Fatal("expected error for invalid path")
	}
}

// --- String (token masking) ---

func TestConfigString_MasksTokens(t *testing.T) {
	cfg := &Config{
		TelegramToken: "abcdef12345",
		GPTToken:      "sk-xxxxxxxxxxxxxxx",
	}
	s := cfg.String()
	// First 4 chars visible, rest masked.
	if !strings.Contains(s, "abcd****") {
		t.Errorf("TelegramToken not masked correctly in: %s", s)
	}
	if !strings.Contains(s, "sk-x****") {
		t.Errorf("GPTToken not masked correctly in: %s", s)
	}
	// Full tokens must NOT appear.
	if strings.Contains(s, "abcdef12345") {
		t.Error("full TelegramToken leaked")
	}
	if strings.Contains(s, "sk-xxxxxxxxxxxxxxx") {
		t.Error("full GPTToken leaked")
	}
}

func TestMaskToken_ShortToken(t *testing.T) {
	if got := maskToken("abc"); got != "****" {
		t.Errorf("maskToken(%q) = %q, want '****'", "abc", got)
	}
	if got := maskToken(""); got != "****" {
		t.Errorf("maskToken(%q) = %q, want '****'", "", got)
	}
}

func TestMaskToken_LongToken(t *testing.T) {
	if got := maskToken("hello_world"); got != "hell****" {
		t.Errorf("maskToken(%q) = %q, want 'hell****'", "hello_world", got)
	}
}
