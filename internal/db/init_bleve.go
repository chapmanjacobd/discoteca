//go:build bleve

package db

import (
	"database/sql"
)

func InitDB(sqlDB *sql.DB) error {
	schema := GetSchema()

	tx, err := sqlDB.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.Exec(schema); err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return err
	}

	// Note: Bleve index is managed separately from SQLite
	// Index initialization happens in the Bleve package

	return Migrate(sqlDB)
}
