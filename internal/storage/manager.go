package storage

import (
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/schema"
)

type Manager struct {
	connectionString string
	db               *gorm.DB
}

func NewManager(connectionString string) *Manager {
	return &Manager{connectionString: connectionString}
}

func (m *Manager) Connect() error {
	var err error

	if m.db != nil {
		return nil
	}

	m.db, err = gorm.Open(postgres.Open(m.connectionString), &gorm.Config{
		NamingStrategy: schema.NamingStrategy{
			TablePrefix: "loc_", // table name prefix, table for `User` would be `t_users`
		},
	})
	if err != nil {
		return err
	}

	return nil
}

func (m *Manager) DB() *gorm.DB {
	return m.db
}
