package history

import (
	"context"
	"database/sql"
	"time"

	"github.com/chapmanjacobd/discoteca/internal/db"
)

// HistoryEntry represents a single playback session entry
type HistoryEntry struct {
	Playhead   int
	TimePlayed int64
	MarkDone   bool
}

// UpdateHistoryWithTime updates playback history in database with a specific timestamp
func UpdateHistoryWithTime(
	ctx context.Context,
	dbPath string,
	paths []string,
	entry HistoryEntry,
) error {
	sqlDB, err := db.Connect(ctx, dbPath)
	if err != nil {
		return err
	}
	defer sqlDB.Close()

	tx, err := sqlDB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	queries := db.New(sqlDB).WithTx(tx)
	done := int64(0)
	if entry.MarkDone {
		done = 1
	}

	for _, path := range paths {
		// Update media aggregate. UpdatePlayHistory only moves
		// time_last_played forward and time_first_played backward.
		if err := queries.UpdatePlayHistory(ctx, db.UpdatePlayHistoryParams{
			TimeLastPlayed:  sql.NullInt64{Int64: entry.TimePlayed, Valid: true},
			TimeFirstPlayed: sql.NullInt64{Int64: entry.TimePlayed, Valid: true},
			Playhead:        sql.NullInt64{Int64: int64(entry.Playhead), Valid: true},
			Path:            path,
		}); err != nil {
			continue
		}

		// Insert granular history record
		if err := queries.InsertHistory(ctx, db.InsertHistoryParams{
			MediaPath:  path,
			TimePlayed: sql.NullInt64{Int64: entry.TimePlayed, Valid: true},
			Playhead:   sql.NullInt64{Int64: int64(entry.Playhead), Valid: true},
			Done:       sql.NullInt64{Int64: done, Valid: true},
		}); err != nil {
			return err
		}
	}

	return tx.Commit()
}

// UpdateHistorySimple updates playback history in database without needing a Tracker
func UpdateHistorySimple(ctx context.Context, dbPath string, paths []string, playhead int, markDone bool) error {
	return UpdateHistoryWithTime(ctx, dbPath, paths, HistoryEntry{
		Playhead:   playhead,
		TimePlayed: time.Now().Unix(),
		MarkDone:   markDone,
	})
}

// DeleteHistoryByPaths removes history records for specified paths
func DeleteHistoryByPaths(ctx context.Context, dbPath string, paths []string) error {
	sqlDB, err := db.Connect(ctx, dbPath)
	if err != nil {
		return err
	}
	defer sqlDB.Close()

	tx, err := sqlDB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	for _, path := range paths {
		if _, err := tx.ExecContext(ctx, "DELETE FROM history WHERE media_path = ?", path); err != nil {
			return err
		}
		// Also reset playhead/play_count in media table?
		// Python remove logic does that too.
		if _, err := tx.ExecContext(
			ctx,
			"UPDATE media SET playhead=0, play_count=0, time_last_played=0, time_first_played=0 WHERE path = ?",
			path,
		); err != nil {
			return err
		}
	}

	return tx.Commit()
}
