package handler

import (
	tele "gopkg.in/telebot.v4"
	"log/slog"
)

func NewStartHandler() tele.HandlerFunc {
	return func(c tele.Context) error {
		slog.Info("got command /start", "from", formatHumanName(c.Sender()), "chat", formatHumanName(c.Chat()))
		h := NewHelpHandler()
		return h(c)
	}
}
