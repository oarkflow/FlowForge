package db

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/jmoiron/sqlx"
	_ "github.com/mattn/go-sqlite3"
)

func Connect(dsn string) (*sqlx.DB, error) {
	dir := filepath.Dir(dsn)
	if dir != "" && dir != "." {
		_ = os.MkdirAll(dir, 0755)
	}

	sqliteDSN := fmt.Sprintf(
		"file:%s?_journal_mode=WAL&_synchronous=NORMAL&_foreign_keys=on&_busy_timeout=5000&cache=shared",
		dsn,
	)
	db, err := sqlx.Open("sqlite3", sqliteDSN)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)
	db.SetConnMaxLifetime(0)
	return db, nil
}
