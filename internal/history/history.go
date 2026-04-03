package history

import (
	"context"
	"database/sql"
	"time"

	"github.com/chapmanjacobd/discoteca/internal/db"
)

type Tracker struct {
	queries *db.Queries
}

func NewTracker(database *sql.DB) *Tracker {
	return &Tracker{
		queries: db.New(database),
	}
}

func (t *Tracker) UpdatePlayback(ctx context.Context, path string, playhead int32) error {
	now := time.Now().Unix()
	// Update media record
	if err := t.queries.UpdatePlayHistory(ctx, db.UpdatePlayHistoryParams{
		TimeLastPlayed:  sql.NullInt64{Int64: now, Valid: true},
		TimeFirstPlayed: sql.NullInt64{Int64: now, Valid: true},
		Playhead:        sql.NullInt64{Int64: int64(playhead), Valid: true},
		Path:            path,
	}); err != nil {
		return err
	}

	// Insert into history table
	return t.queries.InsertHistory(ctx, db.InsertHistoryParams{
		MediaPath:  path,
		TimePlayed: sql.NullInt64{Int64: now, Valid: true},
		Playhead:   sql.NullInt64{Int64: int64(playhead), Valid: true},
		Done:       sql.NullInt64{Int64: 0, Valid: true},
	})
}

func (t *Tracker) MarkDeleted(ctx context.Context, path string) error {
	now := time.Now().Unix()
	return t.queries.MarkDeleted(ctx, db.MarkDeletedParams{
		TimeDeleted: sql.NullInt64{Int64: now, Valid: true},
		Path:        path,
	})
}

// UpdateHistoryWithTime updates playback history in database with a specific timestamp
func UpdateHistoryWithTime(
	ctx context.Context,
	dbPath string,
	paths []string,
	playhead int,
	timePlayed int64,
	markDone bool,
) error {
	sqlDB, err := db.Connect(dbPath)
	if err != nil {
		return err
	}
	defer sqlDB.Close()

	tx, err := sqlDB.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	queries := db.New(sqlDB).WithTx(tx)
	done := int64(0)
	if markDone {
		done = 1
	}

	for _, path := range paths {
		// Update media aggregate
		// Note: UpdatePlayHistory in queries.sql only updates time_last_played if it's newer
		if err := queries.UpdatePlayHistory(ctx, db.UpdatePlayHistoryParams{
			TimeLastPlayed:  sql.NullInt64{Int64: timePlayed, Valid: true},
			TimeFirstPlayed: sql.NullInt64{Int64: timePlayed, Valid: true},
			Playhead:        sql.NullInt64{Int64: int64(playhead), Valid: true},
			Path:            path,
		}); err != nil {
			continue
		}

		// Insert granular history record
		if err := queries.InsertHistory(ctx, db.InsertHistoryParams{
			MediaPath:  path,
			TimePlayed: sql.NullInt64{Int64: timePlayed, Valid: true},
			Playhead:   sql.NullInt64{Int64: int64(playhead), Valid: true},
			Done:       sql.NullInt64{Int64: done, Valid: true},
		}); err != nil {
			return err
		}
	}

	return tx.Commit()
}

// UpdateHistorySimple updates playback history in database without needing a Tracker
func UpdateHistorySimple(ctx context.Context, dbPath string, paths []string, playhead int, markDone bool) error {
	return UpdateHistoryWithTime(ctx, dbPath, paths, playhead, time.Now().Unix(), markDone)
}

// DeleteHistoryByPaths removes history records for specified paths
func DeleteHistoryByPaths(dbPath string, paths []string) error {
	sqlDB, err := db.Connect(dbPath)
	if err != nil {
		return err
	}
	defer sqlDB.Close()

	tx, err := sqlDB.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	for _, path := range paths {
		if _, err := tx.Exec("DELETE FROM history WHERE media_path = ?", path); err != nil {
			return err
		}
		// Also reset playhead/play_count in media table?
		// Python remove logic does that too.
		if _, err := tx.Exec(
			"UPDATE media SET playhead=0, play_count=0, time_last_played=0, time_first_played=0 WHERE path = ?",
			path,
		); err != nil {
			return err
		}
	}

	return tx.Commit()
}
