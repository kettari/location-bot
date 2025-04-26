package handler

import (
	"fmt"
	"github.com/kettari/location-bot/internal/config"
	"github.com/kettari/location-bot/internal/schedule"
	"github.com/kettari/location-bot/internal/storage"
	tele "gopkg.in/telebot.v4"
	"log/slog"
)

func NewGamesHandler() tele.HandlerFunc {
	return func(c tele.Context) error {
		slog.Info("got command /games", "from", formatHumanName(c.Sender()), "chat", formatHumanName(c.Chat()))
		// Only in private chats
		if private, err := isPrivate(c); err != nil {
			return err
		} else if !private {
			return c.Reply("Команды работают только в личной переписке")
		}

		if c.Message() != nil && c.Message().Sender != nil {
			conf := config.GetConfig()
			manager := storage.NewManager(conf.DbConnectionString)
			sch := schedule.NewSchedule(manager)
			if err := sch.LoadJoinableEvents(); err != nil {
				return err
			}
			if err := sch.ExecuteFullReport(fmt.Sprintf("%d,0", c.Message().Sender.ID)); err != nil {
				return err
			}
		}

		return nil
	}
}
