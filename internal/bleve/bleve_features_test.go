//go:build bleve

package bleve

import (
	"fmt"
	"math"
	"path/filepath"
	"testing"
	"time"

	"github.com/blevesearch/bleve/v2"
	"github.com/chapmanjacobd/discoteca/internal/testutils"
)

// TestMediaDocumentWithAllFields tests that MediaDocument can hold all fields
func TestMediaDocumentWithAllFields(t *testing.T) {
	var size int64 = 1024 * 1024 * 100 // 100MB
	var duration int64 = 3600          // 1 hour
	var timeCreated int64 = time.Now().Unix()
	var timeModified int64 = time.Now().Unix()
	var timeDownloaded int64 = time.Now().Unix() - 86400 // 1 day ago
	var timeLastPlayed int64 = time.Now().Unix() - 3600  // 1 hour ago
	var playCount int64 = 5
	var videoCount int64 = 1
	var audioCount int64 = 2
	var subtitleCount int64 = 3
	var width int64 = 1920
	var height int64 = 1080
	var score float64 = 0.95

	title := "Test Video Title"
	description := "Test video description"
	mediaType := "video"
	genre := "Action"
	artist := "Test Artist"
	album := "Test Album"
	language := "en"
	categories := "action,movie"

	media := MediaDocument{
		ID:             filepath.FromSlash("/test/path/video.mp4"),
		Path:           filepath.FromSlash("/test/path/video.mp4"),
		PathTokenized:  filepath.FromSlash("/test/path"),
		Title:          title,
		Description:    description,
		Type:           mediaType,
		Size:           size,
		Duration:       duration,
		TimeCreated:    timeCreated,
		TimeModified:   timeModified,
		TimeDownloaded: timeDownloaded,
		TimeLastPlayed: timeLastPlayed,
		PlayCount:      playCount,
		Genre:          genre,
		Artist:         artist,
		Album:          album,
		Language:       language,
		Categories:     categories,
		VideoCount:     videoCount,
		AudioCount:     audioCount,
		SubtitleCount:  subtitleCount,
		Width:          width,
		Height:         height,
		Score:          score,
	}

	// Verify all fields are set correctly
	if doc := media; doc.ID != filepath.FromSlash("/test/path/video.mp4") {
		t.Errorf("ID mismatch: got %s", doc.ID)
	}
	if media.Title != title {
		t.Errorf("Title mismatch: got %s", media.Title)
	}
	if media.Size != size {
		t.Errorf("Size mismatch: got %d", media.Size)
	}
	if media.Genre != genre {
		t.Errorf("Genre mismatch: got %s", media.Genre)
	}
	if media.PlayCount != playCount {
		t.Errorf("PlayCount mismatch: got %d", media.PlayCount)
	}
}

// TestSearchWithExactMatch tests exact vs fuzzy matching
func TestSearchWithExactMatch(t *testing.T) {
	fixture := testutils.Setup(t)
	defer fixture.Cleanup()

	dbPath := fixture.DBPath

	err := InitIndex(dbPath)
	if err != nil {
		t.Fatalf("InitIndex failed: %v", err)
	}
	defer CloseIndex()

	// Index documents with similar names
	docs := []*MediaDocument{
		{ID: "exact", Path: filepath.FromSlash("/exact.mp4"), PathTokenized: "exact", Title: "Exact Match"},
		{ID: "exact_match", Path: filepath.FromSlash("/exact_match.mp4"), PathTokenized: "exact_match", Title: "Exact Match Video"},
		{ID: "exactly", Path: filepath.FromSlash("/exactly.mp4"), PathTokenized: "exactly", Title: "Exactly What You Need"},
		{ID: "other", Path: filepath.FromSlash("/other.mp4"), PathTokenized: "other", Title: "Other Video"},
	}

	for _, doc := range docs {
		if err := IndexDocument(doc); err != nil {
			t.Fatalf("IndexDocument failed: %v", err)
		}
	}

	// Test exact match - should only return "exact"
	ids, total, err := SearchWithExactMatch("exact", 10, true)
	if err != nil {
		t.Fatalf("SearchWithExactMatch failed: %v", err)
	}
	if total != 1 {
		t.Errorf("Expected 1 exact match, got %d", total)
	}
	if len(ids) != 1 || ids[0] != "exact" {
		t.Errorf("Expected only 'exact', got %v", ids)
	}

	// Test fuzzy match - should return multiple results
	ids, total, err = SearchWithExactMatch("exact", 10, false)
	if err != nil {
		t.Fatalf("SearchWithExactMatch fuzzy failed: %v", err)
	}
	if total < 1 {
		t.Errorf("Expected at least 1 fuzzy match, got %d", total)
	}
	if len(ids) < 1 {
		t.Errorf("Expected at least 1 fuzzy result, got %d", len(ids))
	}
}

// TestSearchWithSort tests sorting with docValues
func TestSearchWithSort(t *testing.T) {
	fixture := testutils.Setup(t)
	defer fixture.Cleanup()

	dbPath := fixture.DBPath

	err := InitIndex(dbPath)
	if err != nil {
		t.Fatalf("InitIndex failed: %v", err)
	}
	defer CloseIndex()

	// Index documents with different sizes and timestamps
	now := time.Now().Unix()
	docs := []*MediaDocument{
		{ID: "doc1", Path: filepath.FromSlash("/doc1.mp4"), PathTokenized: "test", Size: 100, TimeCreated: now - 300, PlayCount: 5},
		{ID: "doc2", Path: filepath.FromSlash("/doc2.mp4"), PathTokenized: "test", Size: 300, TimeCreated: now - 100, PlayCount: 1},
		{ID: "doc3", Path: filepath.FromSlash("/doc3.mp4"), PathTokenized: "test", Size: 200, TimeCreated: now - 200, PlayCount: 10},
	}

	for _, doc := range docs {
		if err := IndexDocument(doc); err != nil {
			t.Fatalf("IndexDocument failed: %v", err)
		}
	}

	// Test sorting by size descending
	sortFields := []SortField{
		{Field: "size", Descending: true},
	}
	ids, total, _, err := SearchWithSort("test", 10, 0, sortFields)
	if err != nil {
		t.Fatalf("SearchWithSort failed: %v", err)
	}
	if total != 3 {
		t.Errorf("Expected 3 results, got %d", total)
	}
	// Should be sorted: doc2 (300), doc3 (200), doc1 (100)
	if len(ids) >= 1 && ids[0] != "doc2" {
		t.Errorf("Expected first result to be doc2 (largest), got %s", ids[0])
	}

	// Test sorting by play count ascending
	sortFields = []SortField{
		{Field: "play_count", Descending: false},
	}
	ids, _, _, err = SearchWithSort("test", 10, 0, sortFields)
	if err != nil {
		t.Fatalf("SearchWithSort ascending failed: %v", err)
	}
	// Should be sorted: doc2 (1), doc1 (5), doc3 (10)
	if len(ids) >= 1 && ids[0] != "doc2" {
		t.Errorf("Expected first result to be doc2 (lowest play count), got %s", ids[0])
	}
}

// TestSearchWithFacets tests faceted search
func TestSearchWithFacets(t *testing.T) {
	fixture := testutils.Setup(t)
	defer fixture.Cleanup()

	dbPath := fixture.DBPath

	err := InitIndex(dbPath)
	if err != nil {
		t.Fatalf("InitIndex failed: %v", err)
	}
	defer CloseIndex()

	// Index documents with different types and genres
	docs := []*MediaDocument{
		{ID: "doc1", Path: filepath.FromSlash("/doc1.mp4"), PathTokenized: "media", Type: "video", Genre: "Action", Size: 100},
		{ID: "doc2", Path: filepath.FromSlash("/doc2.mp4"), PathTokenized: "media", Type: "video", Genre: "Comedy", Size: 200},
		{ID: "doc3", Path: filepath.FromSlash("/doc3.mp4"), PathTokenized: "media", Type: "audio", Genre: "Music", Size: 50},
		{ID: "doc4", Path: filepath.FromSlash("/doc4.mp4"), PathTokenized: "media", Type: "video", Genre: "Action", Size: 150},
	}

	for _, doc := range docs {
		if err := IndexDocument(doc); err != nil {
			t.Fatalf("IndexDocument failed: %v", err)
		}
	}

	// Create facet requests
	facetRequests := make(map[string]*bleve.FacetRequest)
	facetRequests["type_facet"] = NewTermFacetRequest("type", 10)
	facetRequests["genre_facet"] = NewTermFacetRequest("genre", 10)

	// Search with facets
	ids, total, facets, err := SearchWithFacets("media", 10, facetRequests)
	if err != nil {
		t.Fatalf("SearchWithFacets failed: %v", err)
	}
	if total != 4 {
		t.Errorf("Expected 4 results, got %d", total)
	}
	if len(ids) != 4 {
		t.Errorf("Expected 4 IDs, got %d", len(ids))
	}

	// Check type facet results
	if typeFacet, ok := facets["type_facet"]; ok {
		if videoCount, ok := typeFacet.Terms["video"]; !ok || videoCount != 3 {
			t.Errorf("Expected 3 videos, got %d", videoCount)
		}
		if audioCount, ok := typeFacet.Terms["audio"]; !ok || audioCount != 1 {
			t.Errorf("Expected 1 audio, got %d", audioCount)
		}
	} else {
		t.Error("Expected type_facet in results")
	}

	// Check genre facet results
	if genreFacet, ok := facets["genre_facet"]; ok {
		if actionCount, ok := genreFacet.Terms["Action"]; !ok || actionCount != 2 {
			t.Errorf("Expected 2 Action, got %d", actionCount)
		}
	} else {
		t.Error("Expected genre_facet in results")
	}
}

// TestNumericRangeFacet tests numeric range faceting
func TestNumericRangeFacet(t *testing.T) {
	fixture := testutils.Setup(t)
	defer fixture.Cleanup()

	dbPath := fixture.DBPath

	err := InitIndex(dbPath)
	if err != nil {
		t.Fatalf("InitIndex failed: %v", err)
	}
	defer CloseIndex()

	// Index documents with different sizes
	docs := []*MediaDocument{
		{ID: "doc1", PathTokenized: "test", Size: 50},  // Small: 0-100
		{ID: "doc2", PathTokenized: "test", Size: 150}, // Medium: 100-200
		{ID: "doc3", PathTokenized: "test", Size: 250}, // Large: 200+
		{ID: "doc4", PathTokenized: "test", Size: 75},  // Small: 0-100
	}

	for _, doc := range docs {
		if err := IndexDocument(doc); err != nil {
			t.Fatalf("IndexDocument failed: %v", err)
		}
	}

	// Create numeric range facet
	small := 0.0
	medium := 100.0
	large := 200.0
	huge := math.MaxFloat64

	facetRequests := make(map[string]*bleve.FacetRequest)
	facetRequests["size_ranges"] = NewNumericRangeFacetRequest("size", []struct {
		Name string
		Min  *float64
		Max  *float64
	}{
		{"Small", &small, &medium},
		{"Medium", &medium, &large},
		{"Large", &large, &huge},
	})

	// Search with facets
	_, _, facets, err := SearchWithFacets("test", 10, facetRequests)
	if err != nil {
		t.Fatalf("SearchWithFacets failed: %v", err)
	}

	// Check size range results
	if sizeFacet, ok := facets["size_ranges"]; ok {
		if smallCount, ok := sizeFacet.Ranges["Small"]; !ok || smallCount != 2 {
			t.Errorf("Expected 2 Small, got %d", smallCount)
		}
		if mediumCount, ok := sizeFacet.Ranges["Medium"]; !ok || mediumCount != 1 {
			t.Errorf("Expected 1 Medium, got %d", mediumCount)
		}
		if largeCount, ok := sizeFacet.Ranges["Large"]; !ok || largeCount != 1 {
			t.Errorf("Expected 1 Large, got %d", largeCount)
		}
	} else {
		t.Error("Expected size_ranges in results")
	}
}

// TestDateRangeFacet tests date range faceting using numeric ranges on timestamps
func TestDateRangeFacet(t *testing.T) {
	fixture := testutils.Setup(t)
	defer fixture.Cleanup()

	dbPath := fixture.DBPath

	err := InitIndex(dbPath)
	if err != nil {
		t.Fatalf("InitIndex failed: %v", err)
	}
	defer CloseIndex()

	now := time.Now().Unix()
	day := int64(86400)

	// Index documents with different timestamps
	docs := []*MediaDocument{
		{ID: "doc1", PathTokenized: "test", TimeCreated: now - (3 * day)},  // 3 days ago
		{ID: "doc2", PathTokenized: "test", TimeCreated: now - (1 * day)},  // 1 day ago
		{ID: "doc3", PathTokenized: "test", TimeCreated: now - (10 * day)}, // 10 days ago
	}

	for _, doc := range docs {
		if err := IndexDocument(doc); err != nil {
			t.Fatalf("IndexDocument failed: %v", err)
		}
	}

	// Create numeric range facet for timestamps (using float64 for range boundaries)
	daySeconds := float64(86400)
	nowFloat := float64(now)

	facetRequests := make(map[string]*bleve.FacetRequest)
	facetRequests["date_ranges"] = NewNumericRangeFacetRequest("time_created", []struct {
		Name string
		Min  *float64
		Max  *float64
	}{
		{"Last 2 days", ptrFloat64(nowFloat - (2 * daySeconds)), ptrFloat64(nowFloat)},
		{"2-7 days ago", ptrFloat64(nowFloat - (7 * daySeconds)), ptrFloat64(nowFloat - (2 * daySeconds))},
		{"Older than 7 days", ptrFloat64(0), ptrFloat64(nowFloat - (7 * daySeconds))},
	})

	// Search with facets
	_, _, facets, err := SearchWithFacets("test", 10, facetRequests)
	if err != nil {
		t.Fatalf("SearchWithFacets failed: %v", err)
	}

	// Check date range results
	if dateFacet, ok := facets["date_ranges"]; ok {
		if last2Days, ok := dateFacet.Ranges["Last 2 days"]; !ok || last2Days != 1 {
			t.Errorf("Expected 1 in Last 2 days, got %d", last2Days)
		}
		if days2to7, ok := dateFacet.Ranges["2-7 days ago"]; !ok || days2to7 != 1 {
			t.Errorf("Expected 1 in 2-7 days ago, got %d", days2to7)
		}
		if older, ok := dateFacet.Ranges["Older than 7 days"]; !ok || older != 1 {
			t.Errorf("Expected 1 in Older than 7 days, got %d", older)
		}
	} else {
		t.Error("Expected date_ranges in results")
	}
}

// ptrFloat64 returns a pointer to a float64 value
func ptrFloat64(f float64) *float64 {
	return &f
}

// TestBatchIndexDocuments tests batch indexing
func TestBatchIndexDocuments(t *testing.T) {
	fixture := testutils.Setup(t)
	defer fixture.Cleanup()

	dbPath := fixture.DBPath

	err := InitIndex(dbPath)
	if err != nil {
		t.Fatalf("InitIndex failed: %v", err)
	}
	defer CloseIndex()

	// Create 100 documents for batch indexing
	docs := make([]*MediaDocument, 100)
	for i := range 100 {
		docs[i] = &MediaDocument{
			ID:   fmt.Sprintf("doc%d", i),
			Path: filepath.FromSlash(fmt.Sprintf("/path/doc%d.mp4", i)),
		}
	}

	// Batch index with batch size of 25
	err = BatchIndexDocuments(docs, 25)
	if err != nil {
		t.Fatalf("BatchIndexDocuments failed: %v", err)
	}

	// Verify count
	count, err := Count()
	if err != nil {
		t.Fatalf("Count failed: %v", err)
	}
	if count != 100 {
		t.Errorf("Expected 100 documents, got %d", count)
	}
}

// TestBatchIndexDocumentsWithProgress tests batch indexing with progress callback
func TestBatchIndexDocumentsWithProgress(t *testing.T) {
	fixture := testutils.Setup(t)
	defer fixture.Cleanup()

	dbPath := fixture.DBPath

	err := InitIndex(dbPath)
	if err != nil {
		t.Fatalf("InitIndex failed: %v", err)
	}
	defer CloseIndex()

	// Create 50 documents
	docs := make([]*MediaDocument, 50)
	for i := range 50 {
		docs[i] = &MediaDocument{
			ID: fmt.Sprintf("doc%d", i),
		}
	}

	// Track progress
	progressCalls := 0
	lastIndexed := 0

	err = BatchIndexDocumentsWithProgress(docs, 10, func(indexed, total int) {
		progressCalls++
		lastIndexed = indexed
	})
	if err != nil {
		t.Fatalf("BatchIndexDocumentsWithProgress failed: %v", err)
	}

	// Should have 5 progress calls (50 docs / 10 batch size)
	if progressCalls != 5 {
		t.Errorf("Expected 5 progress calls, got %d", progressCalls)
	}
	if lastIndexed != 50 {
		t.Errorf("Expected last indexed to be 50, got %d", lastIndexed)
	}
}

// TestGetIndexStats tests index statistics
func TestGetIndexStats(t *testing.T) {
	fixture := testutils.Setup(t)
	defer fixture.Cleanup()

	dbPath := fixture.DBPath

	err := InitIndex(dbPath)
	if err != nil {
		t.Fatalf("InitIndex failed: %v", err)
	}
	defer CloseIndex()

	// Stats on empty index
	stats, err := GetIndexStats()
	if err != nil {
		t.Fatalf("GetIndexStats failed: %v", err)
	}
	if stats.DocCount != 0 {
		t.Errorf("Expected 0 docs in empty index, got %d", stats.DocCount)
	}
	if !stats.HasDocValues {
		t.Error("Expected HasDocValues to be true")
	}

	// Add some documents
	docs := []*MediaDocument{
		{ID: "doc1"},
		{ID: "doc2"},
		{ID: "doc3"},
	}
	for _, doc := range docs {
		if err := IndexDocument(doc); err != nil {
			t.Fatalf("IndexDocument failed: %v", err)
		}
	}

	// Stats after adding documents
	stats, err = GetIndexStats()
	if err != nil {
		t.Fatalf("GetIndexStats failed: %v", err)
	}
	if stats.DocCount != 3 {
		t.Errorf("Expected 3 docs, got %d", stats.DocCount)
	}
}

// TestSearchWithSortAndFacets tests combined sorting and faceting
func TestSearchWithSortAndFacets(t *testing.T) {
	fixture := testutils.Setup(t)
	defer fixture.Cleanup()

	dbPath := fixture.DBPath

	err := InitIndex(dbPath)
	if err != nil {
		t.Fatalf("InitIndex failed: %v", err)
	}
	defer CloseIndex()

	// Index documents with various attributes
	docs := []*MediaDocument{
		{ID: "doc1", PathTokenized: "search", Type: "video", Size: 100, Title: "Action Video"},
		{ID: "doc2", PathTokenized: "search", Type: "video", Size: 200, Title: "Comedy Video"},
		{ID: "doc3", PathTokenized: "search", Type: "audio", Size: 50, Title: "Music Track"},
	}

	for _, doc := range docs {
		if err := IndexDocument(doc); err != nil {
			t.Fatalf("IndexDocument failed: %v", err)
		}
	}

	// Create sort and facet configuration
	sortFields := []SortField{
		{Field: "size", Descending: true},
	}

	facetRequests := make(map[string]*bleve.FacetRequest)
	facetRequests["type_facet"] = NewTermFacetRequest("type", 10)

	// Search with both sorting and faceting
	result, err := SearchWithSortAndFacets("search", 10, 0, sortFields, facetRequests)
	if err != nil {
		t.Fatalf("SearchWithSortAndFacets failed: %v", err)
	}

	if result.Total != 3 {
		t.Errorf("Expected 3 results, got %d", result.Total)
	}

	// Verify sorting (largest first)
	if len(result.Hits) > 0 && result.Hits[0].ID != "doc2" {
		t.Errorf("Expected first hit to be doc2 (largest), got %s", result.Hits[0].ID)
	}

	// Verify facets
	if typeFacet, ok := result.Facets["type_facet"]; ok {
		if videoCount, ok := typeFacet.Terms["video"]; !ok || videoCount != 2 {
			t.Errorf("Expected 2 videos, got %d", videoCount)
		}
	} else {
		t.Error("Expected type_facet in results")
	}
}

// TestMultiFieldSearch tests multi-field search with boosting
func TestMultiFieldSearch(t *testing.T) {
	fixture := testutils.Setup(t)
	defer fixture.Cleanup()

	dbPath := fixture.DBPath

	err := InitIndex(dbPath)
	if err != nil {
		t.Fatalf("InitIndex failed: %v", err)
	}
	defer CloseIndex()

	// Index documents where title and description have different relevance
	docs := []*MediaDocument{
		{ID: "doc1", Title: "Matrix Movie", Description: "A sci-fi film"},
		{ID: "doc2", Title: "Other Film", Description: "About matrix operations"},
	}

	for _, doc := range docs {
		if err := IndexDocument(doc); err != nil {
			t.Fatalf("IndexDocument failed: %v", err)
		}
	}

	// Search with title boost
	fieldBoosts := map[string]float64{
		"title":       2.0,
		"description": 1.0,
	}

	ids, total, err := MultiFieldSearch("matrix", 10, fieldBoosts)
	if err != nil {
		t.Fatalf("MultiFieldSearch failed: %v", err)
	}
	if total < 1 {
		t.Errorf("Expected at least 1 result, got %d", total)
	}
	if len(ids) < 1 {
		t.Errorf("Expected at least 1 ID, got %d", len(ids))
	}
}

// TestPrefixSearch tests autocomplete/prefix search
func TestPrefixSearch(t *testing.T) {
	fixture := testutils.Setup(t)
	defer fixture.Cleanup()

	dbPath := fixture.DBPath

	err := InitIndex(dbPath)
	if err != nil {
		t.Fatalf("InitIndex failed: %v", err)
	}
	defer CloseIndex()

	// Index documents with different title prefixes
	docs := []*MediaDocument{
		{ID: "doc1", Title: "Matrix"},
		{ID: "doc2", Title: "Matador"},
		{ID: "doc3", Title: "Matching"},
		{ID: "doc4", Title: "Other"},
	}

	for _, doc := range docs {
		if err := IndexDocument(doc); err != nil {
			t.Fatalf("IndexDocument failed: %v", err)
		}
	}

	// Test prefix search for "mat"
	ids, total, err := PrefixSearch("mat", 10)
	if err != nil {
		t.Fatalf("PrefixSearch failed: %v", err)
	}
	if total < 1 {
		t.Errorf("Expected at least 1 result for prefix 'mat', got %d", total)
	}
	// Should match Matrix, Matador, Matching (but not Other)
	if len(ids) < 1 {
		t.Errorf("Expected at least 1 ID, got %d", len(ids))
	}
}

// TestSearchWithExactMatchPagination tests exact match with pagination
func TestSearchWithExactMatchPagination(t *testing.T) {
	fixture := testutils.Setup(t)
	defer fixture.Cleanup()

	dbPath := fixture.DBPath

	err := InitIndex(dbPath)
	if err != nil {
		t.Fatalf("InitIndex failed: %v", err)
	}
	defer CloseIndex()

	// Index multiple documents
	for i := range 20 {
		doc := &MediaDocument{
			ID:            fmt.Sprintf("doc%d", i),
			PathTokenized: "test",
		}
		if err := IndexDocument(doc); err != nil {
			t.Fatalf("IndexDocument failed: %v", err)
		}
	}

	// Test pagination with exact match
	ids1, total1, searchAfter1, err := SearchWithExactMatchAndPagination("test", 5, 0, false, nil)
	if err != nil {
		t.Fatalf("SearchWithExactMatchAndPagination page 1 failed: %v", err)
	}
	if total1 < 5 {
		t.Logf("Warning: Expected at least 5 results, got %d", total1)
	}
	if len(ids1) == 0 {
		t.Fatalf("Expected at least 1 ID on page 1, got 0")
	}

	// Get second page using searchAfter (only if we have results)
	var ids2 []string
	if len(searchAfter1) > 0 && len(ids1) == 5 {
		ids2, _, _, err = SearchWithExactMatchAndPagination("test", 5, 0, false, searchAfter1)
		if err != nil {
			t.Fatalf("SearchWithExactMatchAndPagination page 2 failed: %v", err)
		}
		if len(ids2) == 0 {
			t.Errorf("Expected results on page 2, got none")
		}

		// Verify pages are different
		for i, id := range ids1 {
			if i < len(ids2) && id == ids2[i] {
				t.Errorf("Page 1 and page 2 should have different results at position %d", i)
			}
		}
	}
}
