package db_test

import (
	"database/sql"
	"path/filepath"
	"testing"

	"github.com/chapmanjacobd/discoteca/internal/db"
)

func TestMedia_ToModel(t *testing.T) {
	dbMedia := db.Media{
		Path:     filepath.FromSlash("/test/movie.mp4"),
		Title:    sql.NullString{String: "Test Movie", Valid: true},
		Size:     sql.NullInt64{Int64: 1024, Valid: true},
		Duration: sql.NullInt64{Valid: false},
	}

	media := dbMedia.ToModel()

	if media.Path != filepath.FromSlash("/test/movie.mp4") {
		t.Errorf("Expected path %s, got %s", filepath.FromSlash("/test/movie.mp4"), media.Path)
	}

	if media.Title == nil || *media.Title != "Test Movie" {
		t.Errorf("Expected title Test Movie, got %v", media.Title)
	}

	if media.Size == nil || *media.Size != 1024 {
		t.Errorf("Expected size 1024, got %v", media.Size)
	}

	if media.Duration != nil {
		t.Errorf("Expected duration nil, got %v", media.Duration)
	}

	// Test nullFloat64Ptr
	dbMedia.Fps = sql.NullFloat64{Float64: 24.0, Valid: true}
	media = dbMedia.ToModel()
	if media.Fps == nil || *media.Fps != 24.0 {
		t.Errorf("Expected Fps 24.0, got %v", media.Fps)
	}
}
