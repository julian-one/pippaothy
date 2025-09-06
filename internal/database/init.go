package database

import (
	"fmt"
	"io"
	"os"
	"strconv"
	"time"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
)

func GetPostgresConnectionString() string {
	host := os.Getenv("DB_HOST")
	port := os.Getenv("DB_PORT")
	user := os.Getenv("DB_USER")
	password := os.Getenv("DB_PASSWORD")
	dbname := os.Getenv("DB_NAME")
	return fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable", user, password, host, port, dbname)
}

func Create() (*sqlx.DB, error) {
	connStr := GetPostgresConnectionString()
	db, err := sqlx.Open("postgres", connStr)
	if err != nil {
		return nil, fmt.Errorf("failed to open the database: %w", err)
	}

	// Configure connection pool settings
	configureConnectionPool(db)

	// Test the connection
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	if err = seed(db); err != nil {
		return nil, fmt.Errorf("failed to seed the database: %w", err)
	}
	return db, nil
}

func configureConnectionPool(db *sqlx.DB) {
	// Set maximum number of open connections to the database
	maxOpenConns := getEnvInt("DB_MAX_OPEN_CONNS", 25)
	db.SetMaxOpenConns(maxOpenConns)

	// Set maximum number of idle connections in the pool
	maxIdleConns := getEnvInt("DB_MAX_IDLE_CONNS", 5)
	db.SetMaxIdleConns(maxIdleConns)

	// Set maximum amount of time a connection may be reused
	connMaxLifetime := getEnvDuration("DB_CONN_MAX_LIFETIME", 5*time.Minute)
	db.SetConnMaxLifetime(connMaxLifetime)

	// Set maximum amount of time a connection may be idle
	connMaxIdleTime := getEnvDuration("DB_CONN_MAX_IDLE_TIME", 1*time.Minute)
	db.SetConnMaxIdleTime(connMaxIdleTime)
}

// Helper function to get integer environment variable with default
func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

// Helper function to get duration environment variable with default
func getEnvDuration(key string, defaultValue time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		if duration, err := time.ParseDuration(value); err == nil {
			return duration
		}
	}
	return defaultValue
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
