package schedule

import (
	"log/slog"

	"github.com/kettari/location-bot/internal/bot"
	"github.com/kettari/location-bot/internal/config"
)

// ExecuteFullReport and send notification to recipients
//
// Destination format: chat_id_1,thread_id_1;chat_id_2,thread_id_2
func (s *Schedule) ExecuteFullReport(destination string) error {
	slog.Info("executing joinable games full report")

	conf := config.GetConfig()
	b, err := bot.CreateBot(conf.BotToken, destination)
	if err != nil {
		slog.Error("unable to create bot processor object", "error", err)
		return err
	}
	notification, err := s.Format()
	if err != nil {
		slog.Error("unable to format notification", "error", err)
		return err
	}
	if err = b.Send(notification); err != nil {
		slog.Error("unable to send notification", "error", err)
		return err
	}

	slog.Info("full report sent")

	return nil
}
