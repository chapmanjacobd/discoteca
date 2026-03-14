//go:build bleve

package bleve

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/chapmanjacobd/discoteca/internal/models"
	"github.com/chapmanjacobd/discoteca/internal/testutils"
	_ "github.com/mattn/go-sqlite3"
)

func TestMediaDocument(t *testing.T) {
	var size int64 = 1024
	var duration int64 = 3600
	title := "Test Title"
	description := "Test Description"
	mediaType := "video"
	pathTokenized := filepath.FromSlash("/test/path")

	media := models.Media{
		Path:          filepath.FromSlash("/test/path/video.mp4"),
		Size:          &size,
		Duration:      &duration,
		Title:         &title,
		Description:   &description,
		Type:          &mediaType,
		PathTokenized: &pathTokenized,
	}

	doc := ToBleveDoc(media)

	if doc.ID != filepath.FromSlash("/test/path/video.mp4") {
		t.Errorf("Expected ID %s, got %s", filepath.FromSlash("/test/path/video.mp4"), doc.ID)
	}
	if doc.Path != filepath.FromSlash("/test/path/video.mp4") {
		t.Errorf("Expected Path %s, got %s", filepath.FromSlash("/test/path/video.mp4"), doc.Path)
	}
	if doc.PathTokenized != filepath.FromSlash("/test/path") {
		t.Errorf("Expected PathTokenized %s, got %s", filepath.FromSlash("/test/path"), doc.PathTokenized)
	}
	if doc.Title != "Test Title" {
		t.Errorf("Expected Title Test Title, got %s", doc.Title)
	}
	if doc.Description != "Test Description" {
		t.Errorf("Expected Description Test Description, got %s", doc.Description)
	}
	if doc.Type != "video" {
		t.Errorf("Expected Type video, got %s", doc.Type)
	}
	if doc.Size != 1024 {
		t.Errorf("Expected Size 1024, got %d", doc.Size)
	}
	if doc.Duration != 3600 {
		t.Errorf("Expected Duration 3600, got %d", doc.Duration)
	}
}

func TestMediaDocumentNilFields(t *testing.T) {
	media := models.Media{
		Path: filepath.FromSlash("/test/path/video.mp4"),
	}

	doc := ToBleveDoc(media)

	if doc.ID != filepath.FromSlash("/test/path/video.mp4") {
		t.Errorf("Expected ID %s, got %s", filepath.FromSlash("/test/path/video.mp4"), doc.ID)
	}
	if doc.Path != filepath.FromSlash("/test/path/video.mp4") {
		t.Errorf("Expected Path %s, got %s", filepath.FromSlash("/test/path/video.mp4"), doc.Path)
	}
	if doc.PathTokenized != "" {
		t.Errorf("Expected empty PathTokenized, got %s", doc.PathTokenized)
	}
	if doc.Title != "" {
		t.Errorf("Expected empty Title, got %s", doc.Title)
	}
	if doc.Description != "" {
		t.Errorf("Expected empty Description, got %s", doc.Description)
	}
	if doc.Type != "" {
		t.Errorf("Expected empty Type, got %s", doc.Type)
	}
	if doc.Size != 0 {
		t.Errorf("Expected Size 0, got %d", doc.Size)
	}
	if doc.Duration != 0 {
		t.Errorf("Expected Duration 0, got %d", doc.Duration)
	}
}

func TestInitIndex(t *testing.T) {
	fixture := testutils.Setup(t)
	defer fixture.Cleanup()

	dbPath := fixture.DBPath

	// Test index initialization
	err := InitIndex(dbPath)
	if err != nil {
		t.Fatalf("InitIndex failed: %v", err)
	}

	// Verify index was created
	index := GetIndex()
	if index == nil {
		t.Error("Expected index to be initialized")
	}

	// Test idempotency - calling InitIndex again should not fail
	err = InitIndex(dbPath)
	if err != nil {
		t.Fatalf("InitIndex second call failed: %v", err)
	}

	// Clean up
	CloseIndex()
}

func TestInitIndexCreatesBleveDirectory(t *testing.T) {
	fixture := testutils.Setup(t)
	defer fixture.Cleanup()

	dbPath := fixture.DBPath

	err := InitIndex(dbPath)
	if err != nil {
		t.Fatalf("InitIndex failed: %v", err)
	}
	defer CloseIndex()

	// Check that .bleve directory was created
	dbDir := filepath.Dir(dbPath)
	// Index is created with dbName.bleve pattern
	_, err = os.ReadDir(dbDir)
	if err != nil {
		t.Errorf("Failed to read db directory: %v", err)
	}

	// Verify index exists by checking GetIndex returns non-nil
	index := GetIndex()
	if index == nil {
		t.Error("Expected index to be created")
	}
}

func TestInitIndexWithDifferentExtensions(t *testing.T) {
	fixture := testutils.Setup(t)
	defer fixture.Cleanup()

	// Test with .sqlite extension
	sqlitePath := filepath.Join(fixture.TempDir, "test.sqlite")
	err := InitIndex(sqlitePath)
	if err != nil {
		t.Fatalf("InitIndex with .sqlite failed: %v", err)
	}
	CloseIndex()

	// Check index path
	expectedPath := filepath.Join(fixture.TempDir, "test.bleve")
	_, err = os.Stat(expectedPath)
	if err != nil {
		t.Errorf("Expected bleve index at %s", expectedPath)
	}
}

func TestCloseIndex(t *testing.T) {
	fixture := testutils.Setup(t)
	defer fixture.Cleanup()

	dbPath := fixture.DBPath

	err := InitIndex(dbPath)
	if err != nil {
		t.Fatalf("InitIndex failed: %v", err)
	}

	err = CloseIndex()
	if err != nil {
		t.Errorf("CloseIndex failed: %v", err)
	}

	// Verify index is nil after close
	index := GetIndex()
	if index != nil {
		t.Error("Expected index to be nil after CloseIndex")
	}

	// Test closing already closed index (should not error)
	err = CloseIndex()
	if err != nil {
		t.Errorf("CloseIndex on already closed index failed: %v", err)
	}
}

func TestIndexDocument(t *testing.T) {
	fixture := testutils.Setup(t)
	defer fixture.Cleanup()

	dbPath := fixture.DBPath

	err := InitIndex(dbPath)
	if err != nil {
		t.Fatalf("InitIndex failed: %v", err)
	}
	defer CloseIndex()

	doc := &MediaDocument{
		ID:            "test-id-1",
		Path:          filepath.FromSlash("/test/path/video.mp4"),
		PathTokenized: filepath.FromSlash("/test/path"),
		Title:         "Test Video",
		Description:   "A test video file",
		Type:          "video",
		Size:          1024,
		Duration:      3600,
	}

	err = IndexDocument(doc)
	if err != nil {
		t.Errorf("IndexDocument failed: %v", err)
	}

	// Verify document count
	count, err := Count()
	if err != nil {
		t.Errorf("Count failed: %v", err)
	}
	if count != 1 {
		t.Errorf("Expected count 1, got %d", count)
	}
}

func TestIndexDocumentWithoutInit(t *testing.T) {
	// Ensure index is closed
	CloseIndex()

	doc := &MediaDocument{
		ID:   "test-id",
		Path: filepath.FromSlash("/test/path.mp4"),
	}

	err := IndexDocument(doc)
	if err == nil {
		t.Error("Expected error when indexing without initialization")
	}
	if err.Error() != "bleve index not initialized" {
		t.Errorf("Expected 'bleve index not initialized' error, got %v", err)
	}
}

func TestDeleteDocument(t *testing.T) {
	fixture := testutils.Setup(t)
	defer fixture.Cleanup()

	dbPath := fixture.DBPath

	err := InitIndex(dbPath)
	if err != nil {
		t.Fatalf("InitIndex failed: %v", err)
	}
	defer CloseIndex()

	// Index a document
	doc := &MediaDocument{
		ID:            "test-id-1",
		Path:          filepath.FromSlash("/test/path/video.mp4"),
		PathTokenized: filepath.FromSlash("/test/path"),
		Title:         "Test Video",
	}

	err = IndexDocument(doc)
	if err != nil {
		t.Fatalf("IndexDocument failed: %v", err)
	}

	// Verify it exists
	count, err := Count()
	if err != nil {
		t.Errorf("Count failed: %v", err)
	}
	if count != 1 {
		t.Fatalf("Expected count 1, got %d", count)
	}

	// Delete the document
	err = DeleteDocument("test-id-1")
	if err != nil {
		t.Errorf("DeleteDocument failed: %v", err)
	}

	// Verify it's deleted
	count, err = Count()
	if err != nil {
		t.Errorf("Count after delete failed: %v", err)
	}
	if count != 0 {
		t.Errorf("Expected count 0 after delete, got %d", count)
	}
}

func TestDeleteDocumentWithoutInit(t *testing.T) {
	CloseIndex()

	err := DeleteDocument("test-id")
	if err == nil {
		t.Error("Expected error when deleting without initialization")
	}
}

func TestSearch(t *testing.T) {
	fixture := testutils.Setup(t)
	defer fixture.Cleanup()

	dbPath := fixture.DBPath

	err := InitIndex(dbPath)
	if err != nil {
		t.Fatalf("InitIndex failed: %v", err)
	}
	defer CloseIndex()

	// Index multiple documents
	docs := []*MediaDocument{
		{
			ID:            "doc-1",
			Path:          filepath.FromSlash("/media/videos/sample-video1.mp4"),
			PathTokenized: filepath.FromSlash("/media/videos/sample-video1"),
			Title:         "Sample Video One",
			Description:   "A sample video file",
			Type:          "video",
		},
		{
			ID:            "doc-2",
			Path:          filepath.FromSlash("/media/videos/sample-video2.mp4"),
			PathTokenized: filepath.FromSlash("/media/videos/sample-video2"),
			Title:         "Sample Video Two",
			Description:   "Another sample video clip",
			Type:          "video",
		},
		{
			ID:            "doc-3",
			Path:          filepath.FromSlash("/media/music/sample-audio.mp3"),
			PathTokenized: filepath.FromSlash("/media/music/sample-audio"),
			Title:         "Sample Audio",
			Description:   "A sample audio track",
			Type:          "audio",
		},
	}

	for _, doc := range docs {
		if err := IndexDocument(doc); err != nil {
			t.Fatalf("IndexDocument failed: %v", err)
		}
	}

	// Test search for "sample" in path_tokenized field
	ids, total, err := Search("sample", 10)
	if err != nil {
		t.Errorf("Search failed: %v", err)
	}
	if total < 1 {
		t.Errorf("Expected at least 1 result for 'sample', got %d", total)
	}
	if len(ids) < 1 {
		t.Errorf("Expected at least 1 result for 'sample', got %v", ids)
	}

	// Test search for "videos" in path_tokenized (should match multiple)
	ids, total, err = Search("videos", 10)
	if err != nil {
		t.Errorf("Search for 'videos' failed: %v", err)
	}
	if total < 1 {
		t.Errorf("Expected at least 1 result for 'videos', got %d", total)
	}
	if len(ids) < 1 {
		t.Errorf("Expected at least 1 result for 'videos', got %v", ids)
	}

	// Test search with limit
	ids, total, err = Search("media", 1)
	if err != nil {
		t.Errorf("Search with limit failed: %v", err)
	}
	if len(ids) > 1 {
		t.Errorf("Expected max 1 result with limit 1, got %d", len(ids))
	}
	if total < 1 {
		t.Errorf("Expected at least 1 total result for 'media', got %d", total)
	}
}

func TestSearchWithoutInit(t *testing.T) {
	CloseIndex()

	ids, total, err := Search("test", 10)
	if err == nil {
		t.Error("Expected error when searching without initialization")
	}
	if len(ids) != 0 {
		t.Errorf("Expected empty ids, got %v", ids)
	}
	if total != 0 {
		t.Errorf("Expected total 0, got %d", total)
	}
}

func TestSearchPath(t *testing.T) {
	fixture := testutils.Setup(t)
	defer fixture.Cleanup()

	dbPath := fixture.DBPath

	err := InitIndex(dbPath)
	if err != nil {
		t.Fatalf("InitIndex failed: %v", err)
	}
	defer CloseIndex()

	// Index documents with different paths
	docs := []*MediaDocument{
		{
			ID:            "doc-1",
			Path:          filepath.FromSlash("/home/user/videos/movie.mp4"),
			PathTokenized: filepath.FromSlash("/home/user/videos/movie"),
			Title:         "Movie",
		},
		{
			ID:            "doc-2",
			Path:          filepath.FromSlash("/home/user/videos/clip.mp4"),
			PathTokenized: filepath.FromSlash("/home/user/videos/clip"),
			Title:         "Clip",
		},
		{
			ID:            "doc-3",
			Path:          filepath.FromSlash("/home/user/music/song.mp3"),
			PathTokenized: filepath.FromSlash("/home/user/music/song"),
			Title:         "Song",
		},
	}

	for _, doc := range docs {
		if err := IndexDocument(doc); err != nil {
			t.Fatalf("IndexDocument failed: %v", err)
		}
	}

	// Test path search with wildcard
	ids, err := SearchPath(filepath.FromSlash("/home/user/videos/"), 10)
	if err != nil {
		t.Errorf("SearchPath failed: %v", err)
	}
	if len(ids) != 2 {
		t.Errorf("Expected 2 results for videos path, got %d: %v", len(ids), ids)
	}

	// Test path search with limit
	ids, err = SearchPath(filepath.FromSlash("/home/user/"), 1)
	if err != nil {
		t.Errorf("SearchPath with limit failed: %v", err)
	}
	if len(ids) > 1 {
		t.Errorf("Expected max 1 result with limit 1, got %d", len(ids))
	}
}

func TestSearchPathWithoutInit(t *testing.T) {
	CloseIndex()

	ids, err := SearchPath(filepath.FromSlash("/test"), 10)
	if err == nil {
		t.Error("Expected error when searching path without initialization")
	}
	if len(ids) != 0 {
		t.Errorf("Expected empty ids, got %v", ids)
	}
}

func TestCount(t *testing.T) {
	fixture := testutils.Setup(t)
	defer fixture.Cleanup()

	dbPath := fixture.DBPath

	err := InitIndex(dbPath)
	if err != nil {
		t.Fatalf("InitIndex failed: %v", err)
	}
	defer CloseIndex()

	// Count empty index
	count, err := Count()
	if err != nil {
		t.Errorf("Count failed: %v", err)
	}
	if count != 0 {
		t.Errorf("Expected count 0 for empty index, got %d", count)
	}

	// Add documents
	for i := range 5 {
		doc := &MediaDocument{
			ID:            fmt.Sprintf("doc-%d", i),
			Path:          filepath.FromSlash(fmt.Sprintf("/test/path%d.mp4", i)),
			PathTokenized: filepath.FromSlash("/test/path"),
			Title:         "Test",
		}
		if err := IndexDocument(doc); err != nil {
			t.Fatalf("IndexDocument failed: %v", err)
		}
	}

	count, err = Count()
	if err != nil {
		t.Errorf("Count after indexing failed: %v", err)
	}
	if count != 5 {
		t.Errorf("Expected count 5, got %d", count)
	}
}

func TestCountWithoutInit(t *testing.T) {
	CloseIndex()

	count, err := Count()
	if err == nil {
		t.Error("Expected error when counting without initialization")
	}
	if count != 0 {
		t.Errorf("Expected count 0, got %d", count)
	}
}

func TestReindexAll(t *testing.T) {
	fixture := testutils.Setup(t)
	defer fixture.Cleanup()

	dbPath := fixture.DBPath

	err := InitIndex(dbPath)
	if err != nil {
		t.Fatalf("InitIndex failed: %v", err)
	}
	defer CloseIndex()

	// Add some documents
	for i := range 3 {
		doc := &MediaDocument{
			ID:            fmt.Sprintf("doc-%d", i),
			Path:          filepath.FromSlash(fmt.Sprintf("/test/path%d.mp4", i)),
			PathTokenized: filepath.FromSlash("/test/path"),
			Title:         "Test",
		}
		if err := IndexDocument(doc); err != nil {
			t.Fatalf("IndexDocument failed: %v", err)
		}
	}

	// Verify count
	count, err := Count()
	if err != nil {
		t.Errorf("Count before reindex failed: %v", err)
	}
	if count != 3 {
		t.Errorf("Expected count 3 before reindex, got %d", count)
	}

	// Reindex
	err = ReindexAll()
	if err != nil {
		t.Errorf("ReindexAll failed: %v", err)
	}

	// Count should be 0 after reindex (documents not re-added)
	count, err = Count()
	if err != nil {
		t.Errorf("Count after reindex failed: %v", err)
	}
	if count != 0 {
		t.Errorf("Expected count 0 after reindex, got %d", count)
	}
}

func TestReindexAllWithoutInit(t *testing.T) {
	CloseIndex()

	err := ReindexAll()
	if err == nil {
		t.Error("Expected error when reindexing without initialization")
	}
}

func TestIndexUpdateDocument(t *testing.T) {
	fixture := testutils.Setup(t)
	defer fixture.Cleanup()

	dbPath := fixture.DBPath

	err := InitIndex(dbPath)
	if err != nil {
		t.Fatalf("InitIndex failed: %v", err)
	}
	defer CloseIndex()

	// Index a document
	doc1 := &MediaDocument{
		ID:            "doc-1",
		Path:          filepath.FromSlash("/test/old.mp4"),
		PathTokenized: filepath.FromSlash("/test/old"),
		Title:         "Old Title",
	}
	if err := IndexDocument(doc1); err != nil {
		t.Fatalf("IndexDocument failed: %v", err)
	}

	// Update the same document with new data
	doc2 := &MediaDocument{
		ID:            "doc-1",
		Path:          filepath.FromSlash("/test/new.mp4"),
		PathTokenized: filepath.FromSlash("/test/new"),
		Title:         "New Title",
	}
	if err := IndexDocument(doc2); err != nil {
		t.Fatalf("IndexDocument update failed: %v", err)
	}

	// Count should still be 1
	count, err := Count()
	if err != nil {
		t.Errorf("Count failed: %v", err)
	}
	if count != 1 {
		t.Errorf("Expected count 1 after update, got %d", count)
	}

	// Search for new title
	ids, total, err := Search("New", 10)
	if err != nil {
		t.Errorf("Search for new title failed: %v", err)
	}
	if total != 1 {
		t.Errorf("Expected total 1 for 'New', got %d", total)
	}
	if len(ids) != 1 || ids[0] != "doc-1" {
		t.Errorf("Expected doc-1, got %v", ids)
	}
}
