package postgres

import (
	"database/sql"
	"embed"
	"fmt"

	_ "github.com/lib/pq"
)

//go:embed *.sql
var migrationsFS embed.FS

func NewDB(connStr string) (*sql.DB, error) {
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return nil, fmt.Errorf("open postgres: %w", err)
	}
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("ping postgres: %w", err)
	}
	return db, nil
}

func MigrateUp(db *sql.DB) error {
	// Прогон миграции через Go (без psql)
	sqlBytes, err := migrationsFS.ReadFile("001_init.up.sql")
	if err != nil {
		return fmt.Errorf("read migration: %w", err)
	}
	_, err = db.Exec(string(sqlBytes))
	if err != nil {
		return fmt.Errorf("exec migration: %w", err)
	}
	return nil
}

func MigrateDown(db *sql.DB) error {
	sqlBytes, err := migrationsFS.ReadFile("001_init.down.sql")
	if err != nil {
		return fmt.Errorf("read migration: %w", err)
	}
	_, err = db.Exec(string(sqlBytes))
	if err != nil {
		return fmt.Errorf("exec down: %w", err)
	}
	return nil
}