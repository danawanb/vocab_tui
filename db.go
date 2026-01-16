package main

import (
	"github.com/jmoiron/sqlx"
	_ "github.com/mattn/go-sqlite3"
	"log"
	"os"
	"path/filepath"
)

func ConnectDB() (*sqlx.DB, error) {

	home, err := os.UserHomeDir()
	if err != nil {
		log.Fatal(err)
	}

	dbDir := filepath.Join(home, "Code", "gox", "vocab_tui")
	if err := os.MkdirAll(dbDir, 0755); err != nil {
		log.Fatal(err)
	}

	dbPath := filepath.Join(dbDir, "data.db")
	log.Println("DB path:", dbPath)

	db, err := sqlx.Connect("sqlite3", dbPath)
	if err != nil {
		return nil, err
	}
	return db, nil
}
