package console

import (
	"github.com/kettari/location-bot/internal/config"
	"github.com/kettari/location-bot/internal/handler"
	middle "github.com/kettari/location-bot/internal/middleware"
	tele "gopkg.in/telebot.v4"
	"log/slog"
	"time"
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

	slog.Info("Starting the bot")
	pref := tele.Settings{
		Token:  conf.BotToken,
		Poller: &tele.LongPoller{Timeout: 1 * time.Second},
	}
	b, err := tele.NewBot(pref)
	if err != nil {
		slog.Error("Unable to create bot processor object", "error", err)
		return err
	}
	// Add middleware
	b.Use(middle.Logger(slog.Default()))

	// List bot commands
	b.Handle("/help", handler.NewHelpHandler())
	b.Handle("/start", handler.NewStartHandler())

	// Gracefully shutdown the bot after timeout
	c := make(chan int)
	go stopPoll(b, c)
	// Start poll
	b.Start()

	slog.Info("Bot stopped, exiting")

	return nil
}

// stopPoll after timeout
func stopPoll(bot *tele.Bot, c chan int) {
	stop := time.After(pollTimeout * time.Second)
	slog.Info("Timeout for shutdown started", "timeout_seconds", pollTimeout)
	for {
		select {
		case <-stop:
			slog.Info("Stopping the poll")
			bot.Stop()
			return
		}
	}

}
