package notifier

import (
	"github.com/kettari/location-bot/internal/bot"
	"github.com/kettari/location-bot/internal/config"
	"log/slog"
)

type Report struct {
	conf     *config.Config
	schedule *Schedule
}

func NewReport(conf *config.Config, schedule *Schedule) *Report {
	return &Report{conf, schedule}
}

// ExecuteFullReport and send notification to recipients
//
// Destination format: chat_id_1,thread_id_1;chat_id_2,thread_id_2
func (r *Report) ExecuteFullReport(destination string) error {
	slog.Info("executing joinable games full report")

	b, err := bot.CreateBot(destination)
	if err != nil {
		slog.Error("unable to create bot processor object", "error", err)
		return err
	}
	notification, err := r.schedule.Format()
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
