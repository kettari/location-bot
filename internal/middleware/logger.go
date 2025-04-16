package middle

import (
	tele "gopkg.in/telebot.v4"
	"log/slog"
)

// Logger returns a middle that logs incoming updates.
func Logger(logger *slog.Logger) tele.MiddlewareFunc {
	return func(next tele.HandlerFunc) tele.HandlerFunc {
		return func(c tele.Context) error {
			/*data, _ := json.MarshalIndent(c.Update(), "", "  ")
			logger.Debug(string(data))*/
			return next(c)
		}
	}
}
