package main

import (
	"github.com/jmoiron/sqlx"
	_ "github.com/mattn/go-sqlite3"
)

func ConnectDB() (*sqlx.DB, error) {
	db, err := sqlx.Connect("sqlite3", "data.db")
	if err != nil {
		return nil, err
	}
	return db, nil
}
