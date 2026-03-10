//go:build fts5

package commands

import (
	"database/sql"

	"github.com/chapmanjacobd/discotheque/internal/db"
)

func InitDB(sqlDB *sql.DB) error {
	schema := db.GetSchema()

	tx, err := sqlDB.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.Exec(string(schema)); err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return err
	}

	return runMigrations(sqlDB)
}
