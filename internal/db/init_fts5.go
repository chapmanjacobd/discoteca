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

	return Migrate(sqlDB)
}
