package handler

import (
	tele "gopkg.in/telebot.v4"
	"log/slog"
)

const commonHelp = `Этот бот умеет высылать список игр, на которые <a href="https://rolecon.ru/">можно записаться в клубе «Локация»</a>, г. Москва. 

Команды:

/games — список игр в Локации, на которые можно записаться
/help — эта справка`

func NewHelpHandler() tele.HandlerFunc {
	return func(c tele.Context) error {
		slog.Info("Got command /help", "from", formatHumanName(c.Sender()), "chat", formatHumanName(c.Chat()))
		// Only in private chats
		if private, err := isPrivate(c); err != nil {
			return err
		} else if !private {
			return c.Reply("Команды работают только в личной переписке")
		}
		return c.Send(commonHelp)
	}
}
