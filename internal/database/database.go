package database

import (
	"fmt"
	"io"
	"os"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
)

type Config struct {
	Host     string
	Port     string
	User     string
	Password string
	Name     string
}

func New(cfg Config) (*sqlx.DB, error) {
	connStr := fmt.Sprintf(
		"postgres://%s:%s@%s:%s/%s?sslmode=disable",
		cfg.User,
		cfg.Password,
		cfg.Host,
		cfg.Port,
		cfg.Name,
	)

	db, err := sqlx.Open("postgres", connStr)
	if err != nil {
		return nil, fmt.Errorf("failed to open the database: %w", err)
	}

	// Test the connection
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	if err = seed(db); err != nil {
		return nil, fmt.Errorf("failed to seed the database: %w", err)
	}
	return db, nil
}

func seed(db *sqlx.DB) error {
	f, err := os.Open("./schema/model.sql")
	if err != nil {
		return fmt.Errorf("failed to open model: %w", err)
	}
	defer f.Close()

	data, err := io.ReadAll(f)
	if err != nil {
		return fmt.Errorf("failed to read schema file: %w", err)
	}
	model := string(data)

	if _, err := db.Exec(model); err != nil {
		return fmt.Errorf("failed to create the database model: %w", err)
	}
	return nil
}
