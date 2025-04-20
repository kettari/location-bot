package config

import (
	"log/slog"
	"os"
	"strings"
)

type Config struct {
	Debug               bool
	BotToken            string
	BotUsername         string
	OpenAIApiKey        string
	OpenAILanguageModel string
	DbConnectionString  string
	NotificationChatID  string
}

var config *Config

func GetConfig() *Config {
	if config != nil {
		return config
	}
	config = &Config{}

	// Debug mode
	debug := os.Getenv("BOT_DEBUG")
	if strings.ToLower(debug) == "true" || debug == "1" {
		config.Debug = true
	}
	if config.Debug {
		slog.SetLogLoggerLevel(slog.LevelDebug)
	}

	// Bot token
	config.BotToken = os.Getenv("BOT_TELEGRAM_TOKEN")
	if len(config.BotToken) == 0 {
		slog.Error("bot token not found in the environment (BOT_TELEGRAM_TOKEN)")
		os.Exit(1)
	}

	// Bot username
	config.BotUsername = os.Getenv("BOT_TELEGRAM_NAME")
	if len(config.BotUsername) == 0 {
		slog.Error("bot username not found in the environment (BOT_TELEGRAM_NAME)")
		os.Exit(1)
	}

	// Open AI API key
	config.OpenAIApiKey = os.Getenv("BOT_OPENAI_API_KEY")
	if len(config.OpenAIApiKey) == 0 {
		slog.Error("open AI API key not found in the environment (BOT_OPENAI_API_KEY)")
		os.Exit(1)
	}

	// Database connection string
	config.DbConnectionString = os.Getenv("BOT_DB_STRING")
	if len(config.DbConnectionString) == 0 {
		slog.Error("database connection string is not set in the environment (BOT_DB_STRING)")
		os.Exit(1)
	}

	// Bot LLM
	config.OpenAILanguageModel = os.Getenv("BOT_OPENAI_LANGUAGE_MODEL")
	if len(config.OpenAILanguageModel) == 0 {
		slog.Error("bot language model not found in the environment (BOT_OPENAI_LANGUAGE_MODEL)")
		os.Exit(1)
	}

	// Bot chats to notify
	config.NotificationChatID = os.Getenv("BOT_NOTIFICATION_CHAT_ID")
	if len(config.NotificationChatID) == 0 {
		slog.Error("bot notification chat IDs not found in the environment (BOT_NOTIFICATION_CHAT_ID)")
		os.Exit(1)
	}

	slog.Debug("configuration parameters",
		"BOT_DEBUG", config.Debug,
		"BOT_TELEGRAM_TOKEN", config.BotToken,
		"BOT_TELEGRAM_NAME", config.BotUsername,
		"BOT_OPENAI_API_KEY", config.OpenAIApiKey,
		"BOT_DB_STRING", config.DbConnectionString,
		"BOT_OPENAI_LANGUAGE_MODEL", config.OpenAILanguageModel,
		"BOT_NOTIFICATION_CHAT_ID", config.NotificationChatID)

	return config
}
