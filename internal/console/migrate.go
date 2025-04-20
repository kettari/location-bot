package console

import (
	"github.com/kettari/location-bot/internal/config"
	"github.com/kettari/location-bot/internal/entity"
	"github.com/kettari/location-bot/internal/storage"
	"log/slog"
)

type MigrateCommand struct {
}

func NewMigrateCommand() *MigrateCommand {
	cmd := MigrateCommand{}
	return &cmd
}

func (cmd *MigrateCommand) Name() string {
	return "migrate"
}

func (cmd *MigrateCommand) Description() string {
	return "migrates GORM database scheme"
}

func (cmd *MigrateCommand) Run() error {
	slog.Info("migrating GORM database scheme")

	conf := config.GetConfig()
	manager := storage.NewManager(conf.DbConnectionString)
	if err := manager.Connect(); err != nil {
		return err
	}
	if err := manager.DB().AutoMigrate(&entity.Game{}); err != nil {
		return err
	}

	slog.Info("successfully migrated GORM database scheme")

	return nil
}
