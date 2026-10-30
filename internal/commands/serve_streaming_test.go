package commands_test

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	_ "github.com/mattn/go-sqlite3"

	"github.com/chapmanjacobd/discoteca/internal/commands"
	"github.com/chapmanjacobd/discoteca/internal/db"
	"github.com/chapmanjacobd/discoteca/internal/models"
)

// TestHandleSubtitles tests the subtitles endpoint
func TestHandleSubtitles(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test_subtitles.db")

	sqlDB, _ := sql.Open("sqlite3", dbPath)
	db.InitDB(context.Background(), sqlDB)

	subPath := filepath.Join(tempDir, "test.vtt")
	subContent := `WEBVTT

00:00:01.000 --> 00:00:04.000
Test subtitle
`
	if err := os.WriteFile(subPath, []byte(subContent), 0o644); err != nil {
		t.Fatal(err)
	}

	_, err := sqlDB.Exec(
		`INSERT INTO media (path, title, media_type, time_deleted) VALUES (?, 'Test', 'video', 0)`,
		subPath,
	)
	if err != nil {
		t.Fatal(err)
	}
	sqlDB.Close()

	cmd := &commands.ServeCmd{
		Databases: []string{dbPath},
	}
	defer cmd.Close()
	mux := cmd.Mux()

	t.Run("ValidVTTSubtitle", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/subtitles?path="+subPath, nil)
		req.Header.Set("X-Disco-Token", cmd.APIToken)
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected 200, got %d - Body: %s", w.Code, w.Body.String())
		}

		if w.Header().Get("Content-Type") != "text/vtt" {
			t.Errorf("Expected Content-Type text/vtt, got %s", w.Header().Get("Content-Type"))
		}
	})

	t.Run("MissingPath", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/subtitles", nil)
		req.Header.Set("X-Disco-Token", cmd.APIToken)
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected 400, got %d", w.Code)
		}
	})

	t.Run("UnauthorizedPath", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/subtitles?path=/etc/passwd", nil)
		req.Header.Set("X-Disco-Token", cmd.APIToken)
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		if w.Code != http.StatusForbidden {
			t.Errorf("Expected 403 for unauthorized path, got %d", w.Code)
		}
	})
}

// streamingTestEnv holds shared test infrastructure for streaming tests
type streamingTestEnv struct {
	cmd *commands.ServeCmd
	mux http.Handler
}

// TestHandleDU tests the disk usage endpoint
func TestHandleDU(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test_du.db")

	sqlDB, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer sqlDB.Close()

	if err2 := db.InitDB(context.Background(), sqlDB); err2 != nil {
		t.Fatalf("Failed to initialize DB: %v", err2)
	}

	_, err = sqlDB.Exec(`INSERT INTO media (path, title, media_type, size, duration, time_deleted) VALUES
		('/videos/movies/movie1.mp4', 'Movie1', 'video', 1073741824, 7200, 0),
		('/videos/movies/movie2.mp4', 'Movie2', 'video', 536870912, 3600, 0),
		('/videos/music/song1.mp4', 'Song1', 'video', 268435456, 300, 0),
		('\\videos\\movies\\movie3.mp4', 'Movie3', 'video', 800000000, 5400, 0),
		('\\videos\\music\\song2.mp4', 'Song2', 'video', 150000000, 240, 0),
		('/videos\\tv\\show1.mp4', 'Show1', 'video', 400000000, 1800, 0)`)
	if err != nil {
		t.Fatal(err)
	}

	env := &streamingTestEnv{
		cmd: &commands.ServeCmd{
			Databases: []string{dbPath},
		},
	}
	defer env.cmd.Close()
	env.mux = env.cmd.Mux()

	t.Run("RootLevel", env.testDURootLevel)
	t.Run("SpecificPath", env.testDUSpecificPath)
	t.Run("DirectFiles", env.testDUDirectFiles)
	t.Run("WindowsStylePath", env.testDUWindowsStylePath)
	t.Run("MixedStylePath", env.testDUMixedStylePath)
	t.Run("WindowsAbsolutePath", env.testDUWindowsAbsolutePath)
	t.Run("WindowsRootPath", env.testDUWindowsRootPath)
	t.Run("UNCPath", env.testDUNCPath)
	t.Run("PathWithDotComponents", env.testDUPathWithDotComponents)
	t.Run("PathWithDoubleSeparators", env.testDUPathWithDoubleSeparators)
	t.Run("WindowsPathWithDotDot", env.testDUWindowsPathWithDotDot)
}

func (e *streamingTestEnv) testDURootLevel(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/du", nil)
	req.Header.Set("X-Disco-Token", e.cmd.APIToken)
	w := httptest.NewRecorder()
	e.mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected 200, got %d - Body: %s", w.Code, w.Body.String())
	}

	if w.Header().Get("X-Total-Count") == "" {
		t.Error("Expected X-Total-Count header")
	}

	var resp models.DUResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if resp.Folders == nil {
		t.Error("Expected folders array in response")
	}

	for _, folder := range resp.Folders {
		if folder.Path == "" {
			t.Error("Expected folder to have path")
		}
	}
}

func (e *streamingTestEnv) testDUSpecificPath(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/du?path=/videos", nil)
	req.Header.Set("X-Disco-Token", e.cmd.APIToken)
	w := httptest.NewRecorder()
	e.mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected 200, got %d - Body: %s", w.Code, w.Body.String())
	}

	var resp models.DUResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if resp.Folders == nil {
		t.Error("Expected folders array in response")
	}
	if resp.Files == nil {
		t.Error("Expected files array in response (can be empty)")
	}
}

func (e *streamingTestEnv) testDUDirectFiles(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/du?path=/videos/movies", nil)
	req.Header.Set("X-Disco-Token", e.cmd.APIToken)
	w := httptest.NewRecorder()
	e.mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected 200, got %d - Body: %s", w.Code, w.Body.String())
	}

	var resp models.DUResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if len(resp.Files) == 0 && len(resp.Folders) == 0 {
		t.Error("Expected files or folders in response")
	}

	for _, folder := range resp.Folders {
		if folder.Count == 0 && len(folder.Files) == 1 {
			t.Error("Files should be in the files array, not wrapped as fake folders")
		}
	}
}

func (e *streamingTestEnv) testDUWindowsStylePath(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/du?path=\\videos\\movies", nil)
	req.Header.Set("X-Disco-Token", e.cmd.APIToken)
	w := httptest.NewRecorder()
	e.mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected 200, got %d - Body: %s", w.Code, w.Body.String())
	}

	var resp models.DUResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if len(resp.Files) == 0 && len(resp.Folders) == 0 {
		t.Error("Expected files or folders for Windows-style path")
	}
}

func (e *streamingTestEnv) testDUMixedStylePath(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/du?path=/videos\\movies", nil)
	req.Header.Set("X-Disco-Token", e.cmd.APIToken)
	w := httptest.NewRecorder()
	e.mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected 200, got %d - Body: %s", w.Code, w.Body.String())
	}

	var resp models.DUResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if len(resp.Files) == 0 && len(resp.Folders) == 0 {
		t.Error("Expected files or folders for mixed-style path")
	}
}

func (e *streamingTestEnv) testDUWindowsAbsolutePath(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/du?path=C:\\videos\\movies", nil)
	req.Header.Set("X-Disco-Token", e.cmd.APIToken)
	w := httptest.NewRecorder()
	e.mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected 200, got %d - Body: %s", w.Code, w.Body.String())
	}

	var resp models.DUResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if resp.Folders == nil {
		t.Error("Expected folders array (can be empty)")
	}
}

func (e *streamingTestEnv) testDUWindowsRootPath(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/du?path=C:\\", nil)
	req.Header.Set("X-Disco-Token", e.cmd.APIToken)
	w := httptest.NewRecorder()
	e.mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected 200, got %d - Body: %s", w.Code, w.Body.String())
	}

	var resp models.DUResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if resp.Folders == nil {
		t.Error("Expected folders array (can be empty)")
	}
}

func (e *streamingTestEnv) testDUNCPath(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/du?path=\\\\server\\share\\videos", nil)
	req.Header.Set("X-Disco-Token", e.cmd.APIToken)
	w := httptest.NewRecorder()
	e.mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected 200, got %d - Body: %s", w.Code, w.Body.String())
	}

	var resp models.DUResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if resp.Folders == nil {
		t.Error("Expected folders array (can be empty)")
	}
}

func (e *streamingTestEnv) testDUPathWithDotComponents(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/du?path=/videos/./movies/../music", nil)
	req.Header.Set("X-Disco-Token", e.cmd.APIToken)
	w := httptest.NewRecorder()
	e.mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected 200, got %d - Body: %s", w.Code, w.Body.String())
	}

	var resp models.DUResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if len(resp.Files) == 0 && len(resp.Folders) == 0 {
		t.Error("Expected files or folders for normalized path")
	}
}

func (e *streamingTestEnv) testDUPathWithDoubleSeparators(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/du?path=/videos//movies", nil)
	req.Header.Set("X-Disco-Token", e.cmd.APIToken)
	w := httptest.NewRecorder()
	e.mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected 200, got %d - Body: %s", w.Code, w.Body.String())
	}

	var resp models.DUResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if len(resp.Files) == 0 && len(resp.Folders) == 0 {
		t.Error("Expected files or folders for normalized path")
	}
}

func (e *streamingTestEnv) testDUWindowsPathWithDotDot(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/du?path=\\videos\\movies\\..\\music", nil)
	req.Header.Set("X-Disco-Token", e.cmd.APIToken)
	w := httptest.NewRecorder()
	e.mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected 200, got %d - Body: %s", w.Code, w.Body.String())
	}

	var resp models.DUResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if len(resp.Files) == 0 && len(resp.Folders) == 0 {
		t.Error("Expected files or folders for normalized path")
	}
}

// TestHandleEpisodes tests the episodes endpoint
func TestHandleEpisodes(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test_episodes.db")

	sqlDB, _ := sql.Open("sqlite3", dbPath)
	db.InitDB(context.Background(), sqlDB)

	_, err := sqlDB.Exec(`INSERT INTO media (path, title, media_type, size, time_deleted) VALUES
		('/shows/MyShow/MyShow.S01E01.mp4', 'Episode 1', 'video', 300, 0),
		('/shows/MyShow/MyShow.S01E02.mp4', 'Episode 2', 'video', 200, 0),
		('/shows/MyShow/MyShow.S01E03.mp4', 'Episode 3', 'video', 100, 0),
		('/shows/MyShow/MyShow.S01E01.en.vtt', 'Episode 1 captions', '', 50, 0),
		('/shows/Other/Other.mp4', 'Other', 'video', 10, 0)`)
	if err != nil {
		t.Fatal(err)
	}
	sqlDB.Close()

	cmd := &commands.ServeCmd{
		Databases: []string{dbPath},
	}
	defer cmd.Close()
	mux := cmd.Mux()

	t.Run("GroupEpisodes", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/episodes?all=true", nil)
		req.Header.Set("X-Disco-Token", cmd.APIToken)
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected 200, got %d - Body: %s", w.Code, w.Body.String())
		}

		if w.Header().Get("X-Total-Count") == "" {
			t.Error("Expected X-Total-Count header")
		}

		var groups []models.FolderStats
		if err := json.NewDecoder(w.Body).Decode(&groups); err != nil {
			t.Fatal(err)
		}
		totalFiles := 0
		for _, group := range groups {
			totalFiles += len(group.Files)
			for _, file := range group.Files {
				if filepath.Ext(file.Path) == ".vtt" {
					t.Errorf("sidecar subtitle leaked into group view: %s", file.Path)
				}
			}
		}
		if totalFiles != 4 {
			t.Errorf("expected 4 media files (subtitle excluded), got %d", totalFiles)
		}
	})

	t.Run("SortOrdersFolders", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/episodes?all=true&sort=size", nil)
		req.Header.Set("X-Disco-Token", cmd.APIToken)
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("Expected 200, got %d - Body: %s", w.Code, w.Body.String())
		}
		var groups []models.FolderStats
		if err := json.NewDecoder(w.Body).Decode(&groups); err != nil {
			t.Fatal(err)
		}
		if len(groups) < 2 || groups[0].Path != "/shows/Other" {
			t.Fatalf("expected /shows/Other first for size sort, got %+v", groups)
		}
	})
}

// TestHandleThumbnailShortVideo verifies that clips shorter than the preferred
// 25-second seek position still get a first-frame thumbnail instead of a 404.
func TestHandleThumbnailShortVideo(t *testing.T) {
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		t.Skip("ffmpeg not available")
	}

	tempDir := t.TempDir()
	videoPath := filepath.Join(tempDir, "short.mp4")
	ffmpegArgs := []string{
		"-y", "-hide_banner", "-loglevel", "error",
		"-f", "lavfi", "-i", "testsrc=duration=1:size=160x120:rate=10",
		"-c:v", "libx264", "-pix_fmt", "yuv420p", videoPath,
	}
	if out, err := exec.Command("ffmpeg", ffmpegArgs...).CombinedOutput(); err != nil {
		t.Skipf("unable to generate test video: %v - %s", err, out)
	}

	dbPath := filepath.Join(tempDir, "thumbnails.db")
	sqlDB, _ := sql.Open("sqlite3", dbPath)
	db.InitDB(context.Background(), sqlDB)
	if _, err := sqlDB.Exec(
		`INSERT INTO media (path, title, media_type, time_deleted) VALUES (?, 'Short', 'video', 0)`,
		videoPath,
	); err != nil {
		t.Fatal(err)
	}
	sqlDB.Close()

	cmd := &commands.ServeCmd{Databases: []string{dbPath}}
	defer cmd.Close()
	mux := cmd.Mux()

	req := httptest.NewRequest(http.MethodGet, "/api/thumbnail?path="+videoPath, nil)
	req.Header.Set("X-Disco-Token", cmd.APIToken)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected thumbnail for short video, got %d - %s", w.Code, w.Body.String())
	}
	if w.Body.Len() == 0 {
		t.Fatal("expected non-empty thumbnail")
	}
}
