package database

import (
	"errors"
	"fmt"
	"io"
	"os"

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
		return nil, errors.Join(errors.New("failed to open the database"), err)
	}

	if err = seed(db); err != nil {
		return nil, errors.Join(errors.New("failed to seed the database"), err)
	}
	return db, nil
}

func seed(db *sqlx.DB) error {
	f, err := os.Open("./schema/model.sql")
	if err != nil {
		return errors.Join(errors.New("failed to open model"), err)
	}
	defer f.Close()

	data, err := io.ReadAll(f)
	if err != nil {
		return errors.Join(errors.New(""), err)
	}
	model := string(data)

	if _, err := db.Exec(model); err != nil {
		return errors.Join(errors.New("failed to create the database model"), err)
	}
	return nil
}
