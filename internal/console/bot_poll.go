package console

import (
	"log/slog"
	"time"

	"github.com/kettari/location-bot/internal/config"
	"github.com/kettari/location-bot/internal/handler"
	tele "gopkg.in/telebot.v4"
)

const pollTimeout = 58

type BotPollCommand struct {
}

func NewBotPollCommand() *BotPollCommand {
	cmd := BotPollCommand{}
	return &cmd
}

func (cmd *BotPollCommand) Name() string {
	return "bot:poll"
}

func (cmd *BotPollCommand) Description() string {
	return "polls Telegram Bot API for messages and processes them"
}

func (cmd *BotPollCommand) Run() error {
	conf := config.GetConfig()

	slog.Info("starting the bot")
	pref := tele.Settings{
		Token:  conf.BotToken,
		Poller: &tele.LongPoller{Timeout: 1 * time.Second},
	}
	b, err := tele.NewBot(pref)
	if err != nil {
		slog.Error("unable to create bot processor object", "error", err)
		return err
	}

	// List bot commands
	b.Handle("/help", handler.NewHelpHandler())
	b.Handle("/start", handler.NewStartHandler())
	b.Handle("/games", handler.NewGamesHandler())

	// Gracefully shutdown the bot after timeout
	go stopPoll(b)
	// Start poll
	b.Start()

	slog.Info("bot stopped, exiting")

	return nil
}

// stopPoll after timeout
func stopPoll(bot *tele.Bot) {
	stop := time.After(pollTimeout * time.Second)
	slog.Info("timeout for shutdown started", "timeout_seconds", pollTimeout)
	<-stop
	slog.Info("stopping the poll")
	bot.Stop()
}
