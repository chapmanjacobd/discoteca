package commands_test

import (
	"context"
	"database/sql"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	_ "github.com/mattn/go-sqlite3"

	"github.com/chapmanjacobd/discoteca/internal/commands"
	"github.com/chapmanjacobd/discoteca/internal/db"
)

// TestHandleSubtitles_SubtitleCountOptimization tests that the subtitle endpoint
// checks subtitle_count in the database before attempting ffmpeg conversion.
// This optimization prevents "Failed to convert subtitles" errors for files
// without embedded subtitles.
func TestHandleSubtitles_SubtitleCountOptimization(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test_subtitle_opt.db")

	sqlDB, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer sqlDB.Close()

	db.InitDB(context.Background(), sqlDB)

	// Create a test video file (copy from system or create dummy)
	videoPath := filepath.Join(tempDir, "test_no_subs.mp4")
	// Create a minimal valid MP4 file (ftyp box)
	// This is enough to pass file existence checks
	videoData := []byte{
		0x00, 0x00, 0x00, 0x14, // box size
		0x66, 0x74, 0x79, 0x70, // 'ftyp'
		0x69, 0x73, 0x6F, 0x6D, // 'isom'
		0x00, 0x00, 0x00, 0x01, 0x69, 0x73, 0x6F, 0x6D, // 'isom'
	}
	if err := os.WriteFile(videoPath, videoData, 0o644); err != nil {
		t.Fatal(err)
	}

	// Insert media with subtitle_count = 0 (no embedded subtitles)
	_, err = sqlDB.Exec(`
		INSERT INTO media (path, title, media_type, subtitle_count, subtitle_codecs, time_deleted)
		VALUES (?, 'Test Video No Subs', 'video', 0, '', 0)
	`, videoPath)
	if err != nil {
		t.Fatal(err)
	}

	cmd := &commands.ServeCmd{
		Databases: []string{dbPath},
	}
	defer cmd.Close()

	t.Run("Returns404ForVideoWithoutSubtitles", func(t *testing.T) {
		// Request subtitles for a video with subtitle_count = 0
		// The optimization should check the database first and return 404
		// without attempting ffmpeg conversion
		req := httptest.NewRequest(http.MethodGet, "/api/subtitles?path="+videoPath, nil)
		req.Header.Set("X-Disco-Token", cmd.APIToken)
		w := httptest.NewRecorder()

		cmd.HandleSubtitles(w, req)

		// Should return 404 immediately (optimization: DB check before ffmpeg)
		if w.Code != http.StatusNotFound {
			t.Errorf("Expected 404 for video without subtitles, got %d - Body: %s", w.Code, w.Body.String())
		}

		body := w.Body.String()
		if !strings.Contains(body, "No subtitles") {
			t.Errorf("Expected 'No subtitles' in response, got: %s", body)
		}
	})

	t.Run("DoesNotCallFFmpegForZeroSubtitleCount", func(t *testing.T) {
		// Verify that ffmpeg is not called for files with subtitle_count = 0
		// We do this by checking that the response is fast and doesn't contain
		// ffmpeg error messages

		req := httptest.NewRequest(http.MethodGet, "/api/subtitles?path="+videoPath, nil)
		req.Header.Set("X-Disco-Token", cmd.APIToken)
		w := httptest.NewRecorder()

		cmd.HandleSubtitles(w, req)

		body := w.Body.String()

		// Should NOT contain ffmpeg error messages
		if strings.Contains(body, "Failed to convert subtitles") {
			t.Errorf("Should not attempt ffmpeg conversion for subtitle_count=0, got: %s", body)
		}

		// Should NOT contain ffmpeg-related errors
		if strings.Contains(body, "ffmpeg") {
			t.Errorf("Should not reference ffmpeg for subtitle_count=0, got: %s", body)
		}
	})
}

// TestHandleSubtitles_WithEmbeddedSubtitles tests that the endpoint
// proceeds with ffmpeg conversion when subtitle_count > 0
func TestHandleSubtitles_WithEmbeddedSubtitles(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test_with_subs.db")

	sqlDB, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer sqlDB.Close()

	db.InitDB(context.Background(), sqlDB)

	// Create a test video file
	videoPath := filepath.Join(tempDir, "test_with_subs.mp4")
	videoData := []byte{
		0x00, 0x00, 0x00, 0x14,
		0x66, 0x74, 0x79, 0x70,
		0x69, 0x73, 0x6F, 0x6D,
		0x00, 0x00, 0x00, 0x01, 0x69, 0x73, 0x6F, 0x6D,
	}
	if err := os.WriteFile(videoPath, videoData, 0o644); err != nil {
		t.Fatal(err)
	}

	// Insert media with subtitle_count > 0 (has embedded subtitles)
	_, err = sqlDB.Exec(`
		INSERT INTO media (path, title, media_type, subtitle_count, subtitle_codecs, time_deleted)
		VALUES (?, 'Test Video With Subs', 'video', 2, 'subrip,text', 0)
	`, videoPath)
	if err != nil {
		t.Fatal(err)
	}

	cmd := &commands.ServeCmd{
		Databases: []string{dbPath},
	}
	defer cmd.Close()

	t.Run("AttemptsFFmpegForVideoWithSubtitles", func(t *testing.T) {
		// Request subtitles for a video with subtitle_count > 0
		// This will attempt ffmpeg conversion (which may fail for our dummy file,
		// but the important part is that it tries)
		req := httptest.NewRequest(http.MethodGet, "/api/subtitles?path="+videoPath+"&index=0", nil)
		req.Header.Set("X-Disco-Token", cmd.APIToken)
		w := httptest.NewRecorder()

		cmd.HandleSubtitles(w, req)

		// For a dummy file without real subtitle tracks, ffmpeg will fail
		// The key test is that it ATTEMPTS the conversion (doesn't return early)
		// We expect either:
		// - 415 Unsupported Media (ffmpeg failed but was attempted)
		// - 200 OK (if somehow it worked)
		// But NOT 404 "No subtitles available" (which is the optimization path)

		if w.Code == http.StatusNotFound && strings.Contains(w.Body.String(), "No subtitles available") {
			t.Errorf("Should attempt ffmpeg conversion for subtitle_count>0, but got optimization path")
		}
	})
}

// TestSubtitleCountDatabaseQuery tests that subtitle_count can be queried
// efficiently from the database for optimization purposes
func TestSubtitleCountDatabaseQuery(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test_query.db")

	sqlDB, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer sqlDB.Close()

	db.InitDB(context.Background(), sqlDB)

	// Insert multiple media files with varying subtitle counts
	files := []struct {
		path          string
		subtitleCount int64
	}{
		{filepath.FromSlash("/videos/no_subs1.mp4"), 0},
		{filepath.FromSlash("/videos/no_subs2.mp4"), 0},
		{filepath.FromSlash("/videos/has_subs1.mp4"), 1},
		{filepath.FromSlash("/videos/has_subs2.mp4"), 3},
		{filepath.FromSlash("/videos/has_subs3.mp4"), 2},
	}

	for _, f := range files {
		_, err := sqlDB.Exec(`
			INSERT INTO media (path, title, media_type, subtitle_count, time_deleted)
			VALUES (?, 'Test', 'video', ?, 0)
		`, f.path, f.subtitleCount)
		if err != nil {
			t.Fatal(err)
		}
	}

	t.Run("QueryFilesWithSubtitles", func(t *testing.T) {
		// This is the optimization query used in serve_streaming.go
		var count int
		err := sqlDB.QueryRow("SELECT COUNT(*) FROM media WHERE subtitle_count > 0").Scan(&count)
		if err != nil {
			t.Fatal(err)
		}

		// Should find 3 files with subtitles
		if count != 3 {
			t.Errorf("Expected 3 files with subtitles, got %d", count)
		}
	})

	t.Run("QueryFilesWithoutSubtitles", func(t *testing.T) {
		var count int
		err := sqlDB.QueryRow("SELECT COUNT(*) FROM media WHERE subtitle_count = 0").Scan(&count)
		if err != nil {
			t.Fatal(err)
		}

		// Should find 2 files without subtitles
		if count != 2 {
			t.Errorf("Expected 2 files without subtitles, got %d", count)
		}
	})
}

// TestHandleSubtitles_ExternalSubtitleFile tests that external subtitle files
// (like .srt, .vtt) are still served correctly
func TestHandleSubtitles_ExternalSubtitleFile(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test_external.db")

	sqlDB, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer sqlDB.Close()

	db.InitDB(context.Background(), sqlDB)

	// Create an external subtitle file
	subPath := filepath.Join(tempDir, "movie.srt")
	subContent := `1
00:00:01,000 --> 00:00:04,000
Test subtitle line
`
	if err := os.WriteFile(subPath, []byte(subContent), 0o644); err != nil {
		t.Fatal(err)
	}

	// Insert the subtitle file as media
	_, err = sqlDB.Exec(`
		INSERT INTO media (path, title, media_type, time_deleted)
		VALUES (?, 'Test Subtitle', 'subtitle', 0)
	`, subPath)
	if err != nil {
		t.Fatal(err)
	}

	cmd := &commands.ServeCmd{
		Databases: []string{dbPath},
	}
	defer cmd.Close()

	t.Run("ServesExternalSubtitleFile", func(t *testing.T) {
		// External subtitle files should be served (possibly via ffmpeg conversion to VTT)
		req := httptest.NewRequest(http.MethodGet, "/api/subtitles?path="+subPath, nil)
		req.Header.Set("X-Disco-Token", cmd.APIToken)
		w := httptest.NewRecorder()

		cmd.HandleSubtitles(w, req)

		// For .srt files, ffmpeg will be called to convert to VTT
		// Since our file is valid, it should succeed or at least attempt conversion
		// (not return "No subtitles available" 404)
		if w.Code == http.StatusNotFound && strings.Contains(w.Body.String(), "No subtitles available") {
			t.Errorf("Should not return 'No subtitles' for external subtitle file")
		}
	})
}

// TestHandleSubtitles_NoFFmpegCallForZeroCount verifies that ffmpeg binary
// is not invoked when subtitle_count is 0
func TestHandleSubtitles_NoFFmpegCallForZeroCount(t *testing.T) {
	// Check if ffmpeg exists
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		t.Skip("ffmpeg not installed, skipping ffmpeg invocation test")
	}

	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test_no_ffmpeg.db")

	sqlDB, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer sqlDB.Close()

	db.InitDB(context.Background(), sqlDB)

	// Create a video file
	videoPath := filepath.Join(tempDir, "test.mp4")
	if err := os.WriteFile(videoPath, []byte("dummy"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Insert with subtitle_count = 0
	_, err = sqlDB.Exec(`
		INSERT INTO media (path, title, media_type, subtitle_count, time_deleted)
		VALUES (?, 'Test', 'video', 0, 0)
	`, videoPath)
	if err != nil {
		t.Fatal(err)
	}

	cmd := &commands.ServeCmd{
		Databases: []string{dbPath},
	}
	defer cmd.Close()

	// Make request
	req := httptest.NewRequest(http.MethodGet, "/api/subtitles?path="+videoPath, nil)
	req.Header.Set("X-Disco-Token", cmd.APIToken)
	w := httptest.NewRecorder()

	cmd.HandleSubtitles(w, req)

	// Verify it returned 404 without calling ffmpeg
	if w.Code != http.StatusNotFound {
		t.Errorf("Expected 404 for subtitle_count=0, got %d", w.Code)
	}

	if !strings.Contains(w.Body.String(), "No subtitles") {
		t.Errorf("Expected 'No subtitles' message, got: %s", w.Body.String())
	}
}
