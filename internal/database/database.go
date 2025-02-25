package database

import (
	"database/sql"
	"errors"
	_ "github.com/mattn/go-sqlite3"
	"io"
	"os"
)

func Create() (*sql.DB, error) {
	db, err := sql.Open("sqlite3", "./pippaothy.db")
	if err != nil {
		panic(err)
	}

	if err = seed(db); err != nil {
		return nil, errors.Join(errors.New("failed to seed the db"), err)
	}
	return db, nil
}

func seed(db *sql.DB) error {
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
		return errors.Join(errors.New("failed to create the requests table"), err)
	}
	return nil
}
