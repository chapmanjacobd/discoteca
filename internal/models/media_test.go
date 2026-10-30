package models_test

import (
	"database/sql"
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	"github.com/chapmanjacobd/discoteca/internal/db"
	"github.com/chapmanjacobd/discoteca/internal/models"
)

func TestFromDB(t *testing.T) {
	dbMedia := db.Media{
		Path:     filepath.FromSlash("/test/movie.mp4"),
		Title:    sql.NullString{String: "Test Movie", Valid: true},
		Size:     sql.NullInt64{Int64: 1024, Valid: true},
		Duration: sql.NullInt64{Valid: false},
	}

	media := models.FromDB(dbMedia)

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
	media = models.FromDB(dbMedia)
	if media.Fps == nil || *media.Fps != 24.0 {
		t.Errorf("Expected Fps 24.0, got %v", media.Fps)
	}
}

func TestMedia_Parent(t *testing.T) {
	m := models.Media{Path: filepath.FromSlash("/dir/sub/file.mp4")}
	if got := m.Parent(); got != filepath.FromSlash("/dir/sub") {
		t.Errorf("Parent() = %v, want %v", got, filepath.FromSlash("/dir/sub"))
	}
}

func TestMedia_Stem(t *testing.T) {
	tests := []struct {
		path string
		want string
	}{
		{filepath.FromSlash("/dir/file.mp4"), "file"},
		{"file.mp4", "file"},
		{filepath.FromSlash("/dir/.hidden"), ".hidden"},
		{filepath.FromSlash("/dir/file.tar.gz"), "file.tar"},
	}
	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			m := &models.Media{Path: tt.path}
			if got := m.Stem(); got != tt.want {
				t.Errorf("models.Media.Stem() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMedia_ParentAtDepth(t *testing.T) {
	tests := []struct {
		path  string
		depth int
		want  string
	}{
		// Unix absolute paths
		{"/dir1/dir2/dir3/file.mp4", 0, "/"},
		{"/dir1/dir2/dir3/file.mp4", 1, "/dir1"},
		{"/dir1/dir2/dir3/file.mp4", 2, "/dir1/dir2"},
		{"/dir1/dir2/dir3/file.mp4", 3, "/dir1/dir2/dir3"},
		{"/dir1/dir2/dir3/file.mp4", 4, "/dir1/dir2/dir3"},
		{"/dir1/dir2/dir3/file.mp4", 10, "/dir1/dir2/dir3"},

		// Windows absolute paths (with drive letter)
		{"C:\\dir1\\dir2\\dir3\\file.mp4", 0, "C:\\"},
		{"C:\\dir1\\dir2\\dir3\\file.mp4", 1, "C:\\dir1"},
		{"C:\\dir1\\dir2\\dir3\\file.mp4", 2, "C:\\dir1\\dir2"},
		{"C:\\dir1\\dir2\\dir3\\file.mp4", 3, "C:\\dir1\\dir2\\dir3"},
	}
	for _, tt := range tests {
		t.Run(fmt.Sprintf("%s_depth_%d", tt.path, tt.depth), func(t *testing.T) {
			m := &models.Media{Path: tt.path}
			got := m.ParentAtDepth(tt.depth)
			// Normalize separators for cross-platform comparison
			gotNorm := strings.ReplaceAll(got, "\\", "/")
			wantNorm := strings.ReplaceAll(tt.want, "\\", "/")
			if gotNorm != wantNorm {
				t.Errorf(
					"models.Media.ParentAtDepth(%v) = %v (normalized: %q), want %v (normalized: %q)",
					tt.depth,
					got,
					gotNorm,
					tt.want,
					wantNorm,
				)
			}
		})
	}
}
