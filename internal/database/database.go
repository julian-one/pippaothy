package database

import (
	"github.com/jmoiron/sqlx"
)

// DB wraps sqlx.DB to provide application-specific methods
type DB struct {
	*sqlx.DB
}

// NewDB creates a new database connection wrapper
func NewDB() (*DB, error) {
	sqlxDB, err := Create()
	if err != nil {
		return nil, err
	}
	return &DB{DB: sqlxDB}, nil
}
