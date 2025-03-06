package database

import (
	"errors"
	"io"
	"os"

	"github.com/jmoiron/sqlx"
	_ "github.com/mattn/go-sqlite3"
)

func Create() (*sqlx.DB, error) {
	path := os.Getenv("DB_PATH")
	if path == "" {
		path = "./pippaothy.db"
	}
	db, err := sqlx.Open("sqlite3", path)
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
