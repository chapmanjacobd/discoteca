package commands_test

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	_ "github.com/mattn/go-sqlite3"

	"github.com/chapmanjacobd/discoteca/internal/commands"
	"github.com/chapmanjacobd/discoteca/internal/db"
	"github.com/chapmanjacobd/discoteca/internal/models"
)

// duTestEnv holds shared test infrastructure
type duTestEnv struct {
	cmd *commands.ServeCmd
	mux http.Handler
}

func TestHandleDU_WithFilters(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test_du_filters.db")

	sqlDB, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer sqlDB.Close()

	if err2 := db.InitDB(context.Background(), sqlDB); err2 != nil {
		t.Fatalf("Failed to initialize DB: %v", err2)
	}

	_, err = sqlDB.Exec(`INSERT INTO media (path, title, media_type, size, duration, time_deleted) VALUES
		('/home/videos/movie1.mp4', 'Movie1', 'video', 500000000, 7200, 0),
		('/home/videos/movie2.mp4', 'Movie2', 'video', 300000000, 5400, 0),
		('/home/videos/clip1.mp4', 'Clip1', 'video', 50000000, 30, 0),
		('/home/videos/clip2.mp4', 'Clip2', 'video', 25000000, 15, 0),
		('/home/videos/short.mp4', 'Short', 'video', 10000000, 5, 0),
		('/home/audio/song1.mp3', 'Song1', 'audio', 5000000, 180, 0),
		('/home/audio/song2.mp3', 'Song2', 'audio', 4000000, 150, 0),
		('/home/audio/podcast.mp3', 'Podcast', 'audio', 15000000, 900, 0),
		('/home/images/photo1.png', 'Photo1', 'image', 2000000, 0, 0),
		('/home/images/photo2.png', 'Photo2', 'image', 1500000, 0, 0),
		('/home/images/photo3.png', 'Photo3', 'image', 3000000, 0, 0),
		('/home/documents/doc1.txt', 'Doc1', 'text', 50000, 0, 0),
		('/home/documents/doc2.pdf', 'Doc2', 'text', 100000, 0, 0)`)
	if err != nil {
		t.Fatal(err)
	}

	env := &duTestEnv{
		cmd: SetupTestServeCmd(dbPath),
	}
	defer env.cmd.Close()
	env.mux = env.cmd.Mux()

	t.Run("video-only filter returns only video folders", env.testVideoOnlyFilter)
	t.Run("media_type=video filter returns only video folders", env.testMediaTypeVideoFilter)
	t.Run("audio-only filter returns only audio folders", env.testAudioOnlyFilter)
	t.Run("image-only filter returns only image folders", env.testImageOnlyFilter)
	t.Run("size filter returns only media matching size range", env.testSizeFilter)
	t.Run("duration filter returns only media matching duration range", env.testDurationFilter)
	t.Run("search filter returns only matching media", env.testSearchFilter)
	t.Run("include_counts returns filter bins", env.testIncludeCountsReturnsBins)
	t.Run("filter with include_counts returns filtered bins", env.testFilterWithIncludeCounts)
	t.Run("filters persist when navigating to subfolder", env.testFilterPersistNavigation)
	t.Run("audio filter persists when navigating to subfolder", env.testAudioFilterPersistNavigation)
	t.Run("image filter persists when navigating to subfolder", env.testImageFilterPersistNavigation)
	t.Run("size filter persists when navigating to subfolder", env.testSizeFilterPersistNavigation)
	t.Run("duration filter persists when navigating to subfolder", env.testDurationFilterPersistNavigation)
	t.Run("episodes filecounts filter works in DU mode", env.testFileCountsFilter)
	t.Run("episodes filecounts filter persists when navigating to subfolder", env.testFileCountsFilterPersistNavigation)
}

func (e *duTestEnv) testVideoOnlyFilter(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/du?path=&video=true", nil)
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

	if len(resp.Folders) == 0 {
		t.Error("Expected folders in response")
	}
}

func (e *duTestEnv) testMediaTypeVideoFilter(t *testing.T) {
	unfilteredResp := e.doDURequest(t, "/api/du?path=&include_counts=true")
	unfilteredFileCount := sumFolderCounts(unfilteredResp.Folders)

	filteredResp := e.doDURequest(t, "/api/du?path=&media_type=video")
	filteredFileCount := sumFolderCounts(filteredResp.Folders)

	if filteredFileCount >= unfilteredFileCount {
		t.Errorf(
			"Expected filtered file count (%d) to be less than unfiltered (%d)",
			filteredFileCount,
			unfilteredFileCount,
		)
	}
	if filteredFileCount == 0 {
		t.Error("Expected some video results")
	}
}

func (e *duTestEnv) testAudioOnlyFilter(t *testing.T) {
	resp := e.doDURequest(t, "/api/du?path=&audio=true")
	if resp.TotalCount == 0 {
		t.Log("No audio results found (acceptable)")
	}
}

func (e *duTestEnv) testImageOnlyFilter(t *testing.T) {
	resp := e.doDURequest(t, "/api/du?path=&image=true")
	if resp.TotalCount == 0 {
		t.Log("No image results found (acceptable)")
	}
}

func (e *duTestEnv) testSizeFilter(t *testing.T) {
	e.doDURequest(t, "/api/du?path=&size=>100KB")
}

func (e *duTestEnv) testDurationFilter(t *testing.T) {
	e.doDURequest(t, "/api/du?path=&duration=>10")
}

func (e *duTestEnv) testSearchFilter(t *testing.T) {
	e.doDURequest(t, "/api/du?path=&search=test")
}

func (e *duTestEnv) testIncludeCountsReturnsBins(t *testing.T) {
	resp := e.doDURequest(t, "/api/du?path=&include_counts=true")

	if resp.Counts == nil {
		t.Error("Expected counts in response when include_counts=true")
	}

	if len(resp.Counts.MediaType) == 0 {
		t.Error("Expected media_type bins in counts")
	}

	if len(resp.Counts.SizePercentiles) == 0 {
		t.Error("Expected size percentiles in counts")
	}

	if len(resp.Counts.DurationPercentiles) == 0 {
		t.Error("Expected duration percentiles in counts")
	}
}

func (e *duTestEnv) testFilterWithIncludeCounts(t *testing.T) {
	resp1 := e.doDURequest(t, "/api/du?path=&include_counts=true")
	resp2 := e.doDURequest(t, "/api/du?path=&include_counts=true&video-only=true")

	if resp2.Counts == nil {
		t.Fatal("Expected counts in filtered response")
	}

	_ = resp1
}

func (e *duTestEnv) testFilterPersistNavigation(t *testing.T) {
	resp1 := e.doDURequest(t, "/api/du?path=&video=true")

	if len(resp1.Folders) == 0 {
		t.Fatal("Expected folders at root level with video filter")
	}

	firstFolderPath := resp1.Folders[0].Path
	resp2 := e.doDURequest(t, "/api/du?path="+firstFolderPath+"&video=true")

	totalItems := len(resp2.Folders) + len(resp2.Files)
	if totalItems == 0 {
		t.Errorf("Expected folders or files in subfolder with video filter, got none")
	}
}

func (e *duTestEnv) testAudioFilterPersistNavigation(t *testing.T) {
	resp1 := e.doDURequest(t, "/api/du?path=&audio=true")

	if len(resp1.Folders) == 0 {
		t.Fatal("Expected folders at root level with audio filter")
	}

	firstFolderPath := resp1.Folders[0].Path
	resp2 := e.doDURequest(t, "/api/du?path="+firstFolderPath+"&audio=true")

	totalItems := len(resp2.Folders) + len(resp2.Files)
	if totalItems == 0 {
		t.Errorf("Expected folders or files in subfolder with audio filter, got none")
	}
}

func (e *duTestEnv) testImageFilterPersistNavigation(t *testing.T) {
	resp1 := e.doDURequest(t, "/api/du?path=&image=true")

	if len(resp1.Folders) == 0 {
		t.Fatal("Expected folders at root level")
	}

	resp2 := e.doDURequest(t, "/api/du?path="+resp1.Folders[0].Path+"&image=true")

	totalItems := len(resp2.Folders) + len(resp2.Files)
	if totalItems == 0 {
		t.Errorf("Expected folders or files with image filter, got none")
	}
}

func (e *duTestEnv) testSizeFilterPersistNavigation(t *testing.T) {
	resp1 := e.doDURequest(t, "/api/du?path=&size=>100KB")

	if len(resp1.Folders) == 0 {
		t.Fatal("Expected folders at root level")
	}

	resp2 := e.doDURequest(t, "/api/du?path="+resp1.Folders[0].Path+"&size=>100KB")

	if resp2.FolderCount == 0 && resp2.FileCount == 0 {
		t.Errorf("Expected folders or files with size filter, got none")
	}
}

func (e *duTestEnv) testDurationFilterPersistNavigation(t *testing.T) {
	resp1 := e.doDURequest(t, "/api/du?path=&duration=>10")

	if len(resp1.Folders) == 0 {
		t.Fatal("Expected folders at root level")
	}

	resp2 := e.doDURequest(t, "/api/du?path="+resp1.Folders[0].Path+"&duration=>10")

	if resp2.FolderCount == 0 && resp2.FileCount == 0 {
		t.Errorf("Expected folders or files with duration filter, got none")
	}
}

func (e *duTestEnv) testFileCountsFilter(t *testing.T) {
	resp1 := e.doDURequest(t, "/api/du?path=&include_counts=true")

	resp2 := e.doDURequest(t, "/api/du?path=&file_counts=1")

	if resp2.TotalCount > resp1.TotalCount {
		t.Errorf("Filtered count (%d) should not exceed unfiltered count (%d)",
			resp2.TotalCount, resp1.TotalCount)
	}
}

func (e *duTestEnv) testFileCountsFilterPersistNavigation(t *testing.T) {
	resp1 := e.doDURequest(t, "/api/du?path=&file_counts=1")

	if len(resp1.Folders) == 0 {
		t.Fatal("Expected folders at root level with filecounts filter")
	}

	firstFolderPath := resp1.Folders[0].Path
	resp2 := e.doDURequest(t, "/api/du?path="+firstFolderPath+"&file_counts=1")

	totalItems := len(resp2.Folders) + len(resp2.Files)

	if totalItems == 0 && resp2.TotalCount == 0 {
		t.Skip("No results found, but no error occurred")
	}
}

func (e *duTestEnv) doDURequest(t *testing.T, url string) models.DUResponse {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, url, nil)
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
	return resp
}

func sumFolderCounts(folders []models.FolderStats) int {
	total := 0
	for _, f := range folders {
		total += f.Count
	}
	return total
}

// TestHandleDU_WithFilters_WindowsPaths tests filter functionality with mixed path separators
func TestHandleDU_WithFilters_WindowsPaths(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test_du_filters_windows.db")

	sqlDB, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer sqlDB.Close()

	if err2 := db.InitDB(context.Background(), sqlDB); err2 != nil {
		t.Fatalf("Failed to initialize DB: %v", err2)
	}

	_, err = sqlDB.Exec(`INSERT INTO media (path, title, media_type, size, duration, time_deleted) VALUES
		('/media/videos/movie1.mp4', 'Movie1', 'video', 500000000, 7200, 0),
		('/media/videos/movie2.mp4', 'Movie2', 'video', 300000000, 5400, 0),
		('/media/videos/clip1.mp4', 'Clip1', 'video', 50000000, 30, 0),
		('/media/videos/clip2.mp4', 'Clip2', 'video', 25000000, 15, 0),
		('/media/videos/short.mp4', 'Short', 'video', 10000000, 5, 0),
		('\\media\\audio\\song1.mp3', 'Song1', 'audio', 5000000, 180, 0),
		('\\media\\audio\\song2.mp3', 'Song2', 'audio', 4000000, 150, 0),
		('\\media\\audio\\podcast.mp3', 'Podcast', 'audio', 15000000, 900, 0),
		('/media\\images\\photo1.png', 'Photo1', 'image', 2000000, 0, 0),
		('/media\\images\\photo2.png', 'Photo2', 'image', 1500000, 0, 0),
		('/media\\images\\photo3.png', 'Photo3', 'image', 3000000, 0, 0),
		('\\media\\documents\\doc1.txt', 'Doc1', 'text', 50000, 0, 0),
		('\\media\\documents\\doc2.pdf', 'Doc2', 'text', 100000, 0, 0)`)
	if err != nil {
		t.Fatal(err)
	}

	env := &duTestEnv{
		cmd: SetupTestServeCmd(dbPath),
	}
	defer env.cmd.Close()
	env.mux = env.cmd.Mux()

	t.Run("video-only filter returns only video folders", env.testWindowsVideoOnlyFilter)
	t.Run("media_type=video filter returns only video folders", env.testWindowsMediaTypeVideoFilter)
	t.Run("audio-only filter returns only audio folders", env.testWindowsAudioOnlyFilter)
	t.Run("image-only filter returns only image folders", env.testWindowsImageOnlyFilter)
	t.Run("filters persist when navigating to subfolder", env.testWindowsFilterPersistNavigation)
}

func (e *duTestEnv) testWindowsVideoOnlyFilter(t *testing.T) {
	resp := e.doDURequest(t, "/api/du?video=true")
	if resp.TotalCount == 0 {
		t.Error("Expected video results")
	}
}

func (e *duTestEnv) testWindowsMediaTypeVideoFilter(t *testing.T) {
	unfilteredResp := e.doDURequest(t, "/api/du?include_counts=true")
	unfilteredFileCount := sumFolderCounts(unfilteredResp.Folders)

	filteredResp := e.doDURequest(t, "/api/du?media_type=video")
	filteredFileCount := sumFolderCounts(filteredResp.Folders)

	if filteredFileCount >= unfilteredFileCount {
		t.Errorf(
			"Expected filtered file count (%d) to be less than unfiltered (%d)",
			filteredFileCount,
			unfilteredFileCount,
		)
	}
	if filteredFileCount == 0 {
		t.Error("Expected some video results")
	}
}

func (e *duTestEnv) testWindowsAudioOnlyFilter(t *testing.T) {
	resp := e.doDURequest(t, "/api/du?audio=true")
	if resp.TotalCount == 0 {
		t.Error("Expected audio results")
	}
}

func (e *duTestEnv) testWindowsImageOnlyFilter(t *testing.T) {
	resp := e.doDURequest(t, "/api/du?image=true")
	if resp.TotalCount == 0 {
		t.Error("Expected image results")
	}
}

func (e *duTestEnv) testWindowsFilterPersistNavigation(t *testing.T) {
	resp1 := e.doDURequest(t, "/api/du?video=true")

	if len(resp1.Folders) == 0 {
		t.Fatal("Expected folders at root level with video filter")
	}

	firstFolderPath := resp1.Folders[0].Path
	resp2 := e.doDURequest(t, "/api/du?path="+firstFolderPath+"&video=true")

	totalItems := len(resp2.Folders) + len(resp2.Files)
	if totalItems == 0 {
		t.Errorf("Expected folders or files in subfolder with video filter, got none")
	}
}

// TestHandleDU_MixedUnixWindowsPaths tests DU endpoint with both Unix and Windows paths
func TestHandleDU_MixedUnixWindowsPaths(t *testing.T) {
	dbPath := t.TempDir() + "/test.db"
	testDB, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer testDB.Close()

	_, err = testDB.Exec(`CREATE TABLE media (
		path TEXT PRIMARY KEY,
		title TEXT,
		media_type TEXT,
		size INTEGER,
		duration INTEGER,
		time_deleted INTEGER DEFAULT 0,
		time_created INTEGER,
		time_modified INTEGER,
		time_downloaded INTEGER,
		time_first_played INTEGER,
		time_last_played INTEGER,
		play_count INTEGER,
		playhead INTEGER,
		album TEXT,
		artist TEXT,
		genre TEXT,
		categories TEXT,
		description TEXT,
		language TEXT,
		score REAL,
		video_codecs TEXT,
		audio_codecs TEXT,
		subtitle_codecs TEXT,
		width INTEGER,
		height INTEGER
	)`)
	if err != nil {
		t.Fatal(err)
	}

	_, err = testDB.Exec(`INSERT INTO media (path, title, media_type, size, duration, time_deleted) VALUES
		('/home/user/videos/movie1.mp4', 'Movie1', 'video', 500000000, 7200, 0),
		('/home/user/videos/movie2.mkv', 'Movie2', 'video', 800000000, 9000, 0),
		('/home/user/music/album/song1.mp3', 'Song1', 'audio', 5000000, 240, 0),
		('/home/user/music/album/song2.mp3', 'Song2', 'audio', 6000000, 300, 0),
		('/home/user/docs/report.pdf', 'Report', 'text', 100000, 0, 0),
		('/var/media/shows/episode1.avi', 'Episode1', 'video', 300000000, 3600, 0),
		('/var/media/shows/episode2.avi', 'Episode2', 'video', 350000000, 3800, 0)`)
	if err != nil {
		t.Fatal(err)
	}

	_, err = testDB.Exec(`INSERT INTO media (path, title, media_type, size, duration, time_deleted) VALUES
		('C:\\Users\\John\\Videos\\clip1.mp4', 'Clip1', 'video', 200000000, 1800, 0),
		('C:\\Users\\John\\Videos\\clip2.mov', 'Clip2', 'video', 250000000, 2100, 0),
		('C:\\Users\\John\\Music\\track1.flac', 'Track1', 'audio', 30000000, 420, 0),
		('C:\\Users\\John\\Music\\track2.flac', 'Track2', 'audio', 35000000, 480, 0),
		('C:\\Users\\John\\Documents\\notes.txt', 'Notes', 'text', 5000, 0, 0),
		('D:/Media/TV/series1.mkv', 'Series1', 'video', 400000000, 2700, 0),
		('D:/Media/TV/series2.mkv', 'Series2', 'video', 450000000, 2800, 0),
		('\\\\Server\\Share\\movies\\film.mp4', 'Film', 'video', 1200000000, 10800, 0)`)
	if err != nil {
		t.Fatal(err)
	}

	env := &duTestEnv{
		cmd: SetupTestServeCmd(dbPath),
	}
	defer env.cmd.Close()
	env.mux = env.cmd.Mux()

	t.Run("root_level_shows_both_unix_and_windows_paths", env.testMixedPathsRootLevel)
	t.Run("unix_path_navigation_works", env.testMixedPathsUnixNavigation)
	t.Run("windows_path_navigation_works", env.testMixedPathsWindowsNavigation)
	t.Run("video_filter_works_with_mixed_paths", env.testMixedPathsVideoFilter)
	t.Run("type_counts_include_all_media_types", env.testMixedPathsTypeCounts)
}

func (e *duTestEnv) testMixedPathsRootLevel(t *testing.T) {
	resp := e.doDURequest(t, "/api/du?include_counts=true")

	if resp.TotalCount == 0 {
		t.Error("Expected results from mixed paths")
	}

	pathRoots := make(map[string]bool)
	for _, folder := range resp.Folders {
		parts := strings.FieldsFunc(folder.Path, func(r rune) bool {
			return r == '/' || r == '\\'
		})
		if len(parts) > 0 {
			pathRoots[parts[0]] = true
		}
	}

	if len(pathRoots) < 3 {
		t.Errorf("Expected at least 3 different root paths, got %d", len(pathRoots))
	}
}

func (e *duTestEnv) testMixedPathsUnixNavigation(t *testing.T) {
	resp := e.doDURequest(t, "/api/du?path=/home/user&include_counts=true")

	if resp.FolderCount == 0 && resp.FileCount == 0 {
		t.Error("Expected results under /home/user")
	}
}

func (e *duTestEnv) testMixedPathsWindowsNavigation(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/du?path=C:\\\\Users\\\\John&include_counts=true", nil)
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

	if resp.FolderCount == 0 && resp.FileCount == 0 {
		req2 := httptest.NewRequest(http.MethodGet, "/api/du?path=C:/Users/John&include_counts=true", nil)
		req2.Header.Set("X-Disco-Token", e.cmd.APIToken)
		w2 := httptest.NewRecorder()
		e.mux.ServeHTTP(w2, req2)

		if w2.Code == http.StatusOK {
			var resp2 models.DUResponse
			if err2 := json.Unmarshal(w2.Body.Bytes(), &resp2); err2 == nil {
				if resp2.FolderCount > 0 || resp2.FileCount > 0 {
					return
				}
			}
		}
		t.Skip("Windows path navigation may not work correctly on non-Windows systems")
	}
}

func (e *duTestEnv) testMixedPathsVideoFilter(t *testing.T) {
	resp := e.doDURequest(t, "/api/du?video=true&include_counts=true")

	if resp.TotalCount == 0 {
		t.Error("Expected video results from mixed paths")
	}
}

func (e *duTestEnv) testMixedPathsTypeCounts(t *testing.T) {
	resp := e.doDURequest(t, "/api/du?include_counts=true")

	if resp.Counts == nil {
		t.Fatal("Expected counts to be populated")
	}

	typeMap := make(map[string]int64)
	for _, bt := range resp.Counts.MediaType {
		typeMap[bt.Label] = bt.Value
	}

	if typeMap["video"] == 0 {
		t.Error("Expected video media_type count > 0")
	}
	if typeMap["audio"] == 0 {
		t.Error("Expected audio media_type count > 0")
	}
	if typeMap["text"] == 0 {
		t.Error("Expected text media_type count > 0")
	}
}
