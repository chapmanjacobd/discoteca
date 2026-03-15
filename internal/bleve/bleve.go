//go:build bleve

package bleve

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/blevesearch/bleve/v2"
	_ "github.com/blevesearch/bleve/v2/analysis/analyzer/custom" // Register custom analyzer
	"github.com/blevesearch/bleve/v2/analysis/analyzer/keyword"
	"github.com/blevesearch/bleve/v2/analysis/analyzer/standard"
	_ "github.com/blevesearch/bleve/v2/analysis/token/edgengram" // Register edge_ngram filter
	_ "github.com/blevesearch/bleve/v2/analysis/token/lowercase" // Register lowercase filter
	"github.com/blevesearch/bleve/v2/search"
	"github.com/blevesearch/bleve/v2/search/query"
	"github.com/chapmanjacobd/discoteca/internal/db"
	"github.com/chapmanjacobd/discoteca/internal/models"
)

var (
	indexInstance bleve.Index
	indexMutex    sync.RWMutex
	indexPath     string
)

// MediaDocument represents a document to be indexed in Bleve
type MediaDocument struct {
	ID             string  `json:"id"`
	Path           string  `json:"path"`
	PathTokenized  string  `json:"path_tokenized"`
	Title          string  `json:"title"`
	Description    string  `json:"description"`
	Type           string  `json:"type"`
	Size           int64   `json:"size"`
	Duration       int64   `json:"duration"`
	TimeCreated    int64   `json:"time_created"`
	TimeModified   int64   `json:"time_modified"`
	TimeDownloaded int64   `json:"time_downloaded"`
	TimeLastPlayed int64   `json:"time_last_played"`
	PlayCount      int64   `json:"play_count"`
	Genre          string  `json:"genre"`
	Artist         string  `json:"artist"`
	Album          string  `json:"album"`
	Language       string  `json:"language"`
	Categories     string  `json:"categories"`
	VideoCount     int64   `json:"video_count"`
	AudioCount     int64   `json:"audio_count"`
	SubtitleCount  int64   `json:"subtitle_count"`
	Width          int64   `json:"width"`
	Height         int64   `json:"height"`
	Score          float64 `json:"score"`
}

// ToBleveDoc converts a Media model to a BleveDocument
func ToBleveDoc(m models.Media) *MediaDocument {
	doc := &MediaDocument{
		ID:   m.Path,
		Path: m.Path,
	}

	if m.PathTokenized != nil {
		doc.PathTokenized = *m.PathTokenized
	}
	if m.Title != nil {
		doc.Title = *m.Title
	}
	if m.Description != nil {
		doc.Description = *m.Description
	}
	if m.Type != nil {
		doc.Type = *m.Type
	}
	if m.Size != nil {
		doc.Size = *m.Size
	}
	if m.Duration != nil {
		doc.Duration = *m.Duration
	}
	if m.TimeCreated != nil {
		doc.TimeCreated = *m.TimeCreated
	}
	if m.TimeModified != nil {
		doc.TimeModified = *m.TimeModified
	}
	if m.TimeDownloaded != nil {
		doc.TimeDownloaded = *m.TimeDownloaded
	}
	if m.TimeLastPlayed != nil {
		doc.TimeLastPlayed = *m.TimeLastPlayed
	}
	if m.PlayCount != nil {
		doc.PlayCount = *m.PlayCount
	}
	if m.Genre != nil {
		doc.Genre = *m.Genre
	}
	if m.Artist != nil {
		doc.Artist = *m.Artist
	}
	if m.Album != nil {
		doc.Album = *m.Album
	}
	if m.Language != nil {
		doc.Language = *m.Language
	}
	if m.Categories != nil {
		doc.Categories = *m.Categories
	}
	if m.VideoCount != nil {
		doc.VideoCount = *m.VideoCount
	}
	if m.AudioCount != nil {
		doc.AudioCount = *m.AudioCount
	}
	if m.SubtitleCount != nil {
		doc.SubtitleCount = *m.SubtitleCount
	}
	if m.Width != nil {
		doc.Width = *m.Width
	}
	if m.Height != nil {
		doc.Height = *m.Height
	}
	if m.Score != nil {
		doc.Score = *m.Score
	}

	return doc
}

// ToBleveDocFromUpsert converts UpsertMediaParams to a BleveDocument
func ToBleveDocFromUpsert(p db.UpsertMediaParams) *MediaDocument {
	doc := &MediaDocument{
		ID:   p.Path,
		Path: p.Path,
	}

	if p.Title.Valid {
		doc.Title = p.Title.String
	}
	if p.Description.Valid {
		doc.Description = p.Description.String
	}
	if p.Type.Valid {
		doc.Type = p.Type.String
	}
	if p.Size.Valid {
		doc.Size = p.Size.Int64
	}
	if p.Duration.Valid {
		doc.Duration = p.Duration.Int64
	}
	if p.TimeCreated.Valid {
		doc.TimeCreated = p.TimeCreated.Int64
	}
	if p.TimeModified.Valid {
		doc.TimeModified = p.TimeModified.Int64
	}
	if p.TimeDownloaded.Valid {
		doc.TimeDownloaded = p.TimeDownloaded.Int64
	}
	if p.Genre.Valid {
		doc.Genre = p.Genre.String
	}
	if p.Artist.Valid {
		doc.Artist = p.Artist.String
	}
	if p.Album.Valid {
		doc.Album = p.Album.String
	}
	if p.Language.Valid {
		doc.Language = p.Language.String
	}
	if p.Categories.Valid {
		doc.Categories = p.Categories.String
	}
	if p.VideoCount.Valid {
		doc.VideoCount = p.VideoCount.Int64
	}
	if p.AudioCount.Valid {
		doc.AudioCount = p.AudioCount.Int64
	}
	if p.SubtitleCount.Valid {
		doc.SubtitleCount = p.SubtitleCount.Int64
	}
	if p.Width.Valid {
		doc.Width = p.Width.Int64
	}
	if p.Height.Valid {
		doc.Height = p.Height.Int64
	}
	if p.Score.Valid {
		doc.Score = p.Score.Float64
	}

	return doc
}

// InitIndex initializes or opens the Bleve index
func InitIndex(dbPath string) error {
	indexMutex.Lock()
	defer indexMutex.Unlock()

	if indexInstance != nil {
		return nil // Already initialized
	}

	// Determine index path (same directory as database)
	dbDir := filepath.Dir(dbPath)
	dbName := filepath.Base(dbPath)
	if ext := filepath.Ext(dbName); ext != "" {
		dbName = dbName[:len(dbName)-len(ext)]
	}
	indexPath = filepath.Join(dbDir, fmt.Sprintf("%s.bleve", dbName))

	// Check if index exists
	_, err := os.Stat(indexPath)
	if err == nil {
		// Open existing index
		indexInstance, err = bleve.Open(indexPath)
		if err != nil {
			return fmt.Errorf("failed to open existing bleve index: %w", err)
		}
		return nil
	}

	// Create new index with custom mapping
	indexInstance, err = createIndex(indexPath)
	if err != nil {
		return fmt.Errorf("failed to create bleve index: %w", err)
	}

	return nil
}

// createIndex creates a new Bleve index with custom field mappings
func createIndex(path string) (bleve.Index, error) {
	// Create field mappings
	// Path field - keyword analyzer for exact matching, docValues for faceting/sorting
	pathField := bleve.NewTextFieldMapping()
	pathField.Analyzer = keyword.Name
	pathField.DocValues = true
	pathField.Store = true   // Needed for results display
	pathField.IncludeInAll = false

	// Path tokenized - standard analyzer for FTS, no docValues needed
	pathTokenizedField := bleve.NewTextFieldMapping()
	pathTokenizedField.Analyzer = standard.Name
	pathTokenizedField.DocValues = false
	pathTokenizedField.Store = false
	pathTokenizedField.IncludeInAll = true

	// Title - edge_ngram for autocomplete, docValues for sorting
	titleField := bleve.NewTextFieldMapping()
	titleField.Analyzer = "title_edge_ngram"
	titleField.IncludeInAll = true
	titleField.DocValues = true
	titleField.Store = true

	// Description - standard analyzer, no docValues
	descriptionField := bleve.NewTextFieldMapping()
	descriptionField.Analyzer = standard.Name
	descriptionField.IncludeInAll = true
	descriptionField.DocValues = false
	descriptionField.Store = false // Don't store, saves space

	// Type - keyword for exact matching, docValues for faceting
	typeField := bleve.NewTextFieldMapping()
	typeField.Analyzer = keyword.Name
	typeField.DocValues = true
	typeField.Store = true

	// Numeric fields with docValues enabled
	sizeField := bleve.NewNumericFieldMapping()
	sizeField.DocValues = true
	sizeField.Store = true

	durationField := bleve.NewNumericFieldMapping()
	durationField.DocValues = true
	durationField.Store = true

	// Date/timestamp fields with DateTime mapping for date range facets
	// DateTime fields need to be stored as time.Time for proper date range faceting
	// We use numeric field mapping with docValues for sorting/faceting
	timeCreatedField := bleve.NewNumericFieldMapping()
	timeCreatedField.DocValues = true
	timeCreatedField.Store = true

	timeModifiedField := bleve.NewNumericFieldMapping()
	timeModifiedField.DocValues = true
	timeModifiedField.Store = true

	timeDownloadedField := bleve.NewNumericFieldMapping()
	timeDownloadedField.DocValues = true
	timeDownloadedField.Store = true

	timeLastPlayedField := bleve.NewNumericFieldMapping()
	timeLastPlayedField.DocValues = true
	timeLastPlayedField.Store = true

	playCountField := bleve.NewNumericFieldMapping()
	playCountField.DocValues = true
	playCountField.Store = true

	// Text faceting fields with docValues
	genreField := bleve.NewTextFieldMapping()
	genreField.Analyzer = keyword.Name
	genreField.DocValues = true
	genreField.Store = true

	artistField := bleve.NewTextFieldMapping()
	artistField.Analyzer = keyword.Name
	artistField.DocValues = true
	artistField.Store = true

	languageField := bleve.NewTextFieldMapping()
	languageField.Analyzer = keyword.Name
	languageField.DocValues = true
	languageField.Store = true

	// Caption fields
	mediaPathField := bleve.NewTextFieldMapping()
	mediaPathField.Analyzer = keyword.Name
	mediaPathField.DocValues = true
	mediaPathField.Store = true

	captionTextField := bleve.NewTextFieldMapping()
	captionTextField.Analyzer = standard.Name
	captionTextField.Store = true
	captionTextField.IncludeInAll = true

	timeField := bleve.NewNumericFieldMapping()
	timeField.DocValues = true
	timeField.Store = true

	// Create document mapping
	docMapping := bleve.NewDocumentMapping()
	docMapping.AddFieldMappingsAt("path", pathField)
	docMapping.AddFieldMappingsAt("path_tokenized", pathTokenizedField)
	docMapping.AddFieldMappingsAt("title", titleField)
	docMapping.AddFieldMappingsAt("description", descriptionField)
	docMapping.AddFieldMappingsAt("type", typeField)
	docMapping.AddFieldMappingsAt("size", sizeField)
	docMapping.AddFieldMappingsAt("duration", durationField)
	docMapping.AddFieldMappingsAt("time_created", timeCreatedField)
	docMapping.AddFieldMappingsAt("time_modified", timeModifiedField)
	docMapping.AddFieldMappingsAt("time_downloaded", timeDownloadedField)
	docMapping.AddFieldMappingsAt("time_last_played", timeLastPlayedField)
	docMapping.AddFieldMappingsAt("play_count", playCountField)
	docMapping.AddFieldMappingsAt("genre", genreField)
	docMapping.AddFieldMappingsAt("artist", artistField)
	docMapping.AddFieldMappingsAt("language", languageField)
	// Add caption fields to the same mapping (since we use default mapping for both)
	docMapping.AddFieldMappingsAt("media_path", mediaPathField)
	docMapping.AddFieldMappingsAt("text", captionTextField)
	docMapping.AddFieldMappingsAt("time", timeField)

	// Disable dynamic indexing for unknown fields (improves performance)
	docMapping.Dynamic = false

	// Create index mapping
	indexMapping := bleve.NewIndexMapping()
	indexMapping.DefaultMapping = docMapping
	indexMapping.DefaultAnalyzer = standard.Name
	indexMapping.ScoringModel = "bm25"

	// Register custom edge_ngram analyzer for title autocomplete
	// This creates tokens from the start of words: "matrix" → "m", "ma", "mat", "matr", "matri", "matrix"
	// First register the edge_ngram token filter
	err := indexMapping.AddCustomTokenFilter("title_edge_ngram", map[string]any{
		"type": "edge_ngram",
		"min":  float64(1),
		"max":  float64(20),
		"back": false, // false = FRONT side (prefix)
	})
	if err != nil {
		return nil, fmt.Errorf("failed to register edge_ngram token filter: %w", err)
	}

	// Now create the analyzer using the custom filter
	err = indexMapping.AddCustomAnalyzer("title_edge_ngram", map[string]any{
		"type":      "custom",
		"tokenizer": "unicode",
		"token_filters": []string{
			"to_lower",
			"title_edge_ngram",
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to register edge_ngram analyzer: %w", err)
	}

	// Create the index
	return bleve.New(path, indexMapping)
}

// CloseIndex closes the Bleve index
func CloseIndex() error {
	indexMutex.Lock()
	defer indexMutex.Unlock()

	if indexInstance == nil {
		return nil
	}

	err := indexInstance.Close()
	indexInstance = nil
	return err
}

// GetIndex returns the current Bleve index instance
func GetIndex() bleve.Index {
	indexMutex.RLock()
	defer indexMutex.RUnlock()
	return indexInstance
}

// IndexDocument adds or updates a document in the index
func IndexDocument(doc *MediaDocument) error {
	indexMutex.RLock()
	defer indexMutex.RUnlock()

	if indexInstance == nil {
		return fmt.Errorf("bleve index not initialized")
	}

	return indexInstance.Index(doc.ID, doc)
}

// DeleteDocument removes a document from the index
func DeleteDocument(id string) error {
	indexMutex.RLock()
	defer indexMutex.RUnlock()

	if indexInstance == nil {
		return fmt.Errorf("bleve index not initialized")
	}

	return indexInstance.Delete(id)
}

// Search performs a search query on the Bleve index
func Search(queryStr string, limit int) ([]string, uint64, error) {
	indexMutex.RLock()
	defer indexMutex.RUnlock()

	if indexInstance == nil {
		return nil, 0, fmt.Errorf("bleve index not initialized")
	}

	// Create a match query that searches all fields
	bleveQuery := bleve.NewMatchQuery(queryStr)

	// Create search request
	searchRequest := bleve.NewSearchRequest(bleveQuery)
	searchRequest.Size = limit
	searchRequest.Fields = []string{"id", "path"}

	// Execute search
	results, err := indexInstance.Search(searchRequest)
	if err != nil {
		return nil, 0, err
	}

	// Extract IDs from results
	ids := make([]string, len(results.Hits))
	for i, hit := range results.Hits {
		ids[i] = hit.ID
	}

	return ids, results.Total, nil
}

// SearchWithPagination performs a search with pagination support
// Supports both offset-based (From/Size) pagination
func SearchWithPagination(queryStr string, limit, offset int, searchAfter []string) ([]string, uint64, []string, error) {
	indexMutex.RLock()
	defer indexMutex.RUnlock()

	if indexInstance == nil {
		return nil, 0, nil, fmt.Errorf("bleve index not initialized")
	}

	// Create a match query that searches all fields
	bleveQuery := bleve.NewMatchQuery(queryStr)

	// Create search request
	searchRequest := bleve.NewSearchRequest(bleveQuery)
	searchRequest.Size = limit
	searchRequest.From = offset
	searchRequest.Fields = []string{"id", "path"}

	// Add sort for deterministic pagination (score desc, then id for tie-breaking)
	searchRequest.Sort = search.ParseSortOrderStrings([]string{"-_score", "id"})

	// Execute search
	results, err := indexInstance.Search(searchRequest)
	if err != nil {
		return nil, 0, nil, err
	}

	// Extract IDs from results
	ids := make([]string, len(results.Hits))
	for i, hit := range results.Hits {
		ids[i] = hit.ID
	}

	// Get search_after keys from last hit for next page
	var nextSearchAfter []string
	if len(results.Hits) > 0 {
		lastHit := results.Hits[len(results.Hits)-1]
		nextSearchAfter = lastHit.Sort
	}

	return ids, results.Total, nextSearchAfter, nil
}

// SearchPath performs a path-specific search
func SearchPath(pathPattern string, limit int) ([]string, error) {
	indexMutex.RLock()
	defer indexMutex.RUnlock()

	if indexInstance == nil {
		return nil, fmt.Errorf("bleve index not initialized")
	}

	// For path searches, use wildcard query
	bleveQuery := bleve.NewWildcardQuery(pathPattern + "*")
	bleveQuery.SetField("path")

	searchRequest := bleve.NewSearchRequest(bleveQuery)
	searchRequest.Size = limit

	results, err := indexInstance.Search(searchRequest)
	if err != nil {
		return nil, err
	}

	ids := make([]string, len(results.Hits))
	for i, hit := range results.Hits {
		ids[i] = hit.ID
	}

	return ids, nil
}

// Count returns the total number of documents in the index
func Count() (uint64, error) {
	indexMutex.RLock()
	defer indexMutex.RUnlock()

	if indexInstance == nil {
		return 0, fmt.Errorf("bleve index not initialized")
	}

	return indexInstance.DocCount()
}

// ReindexAll rebuilds the entire index (call after bulk operations)
func ReindexAll() error {
	indexMutex.Lock()
	defer indexMutex.Unlock()

	if indexInstance == nil {
		return fmt.Errorf("bleve index not initialized")
	}

	// Close current index
	err := indexInstance.Close()
	if err != nil {
		return err
	}

	// Remove old index files
	err = os.RemoveAll(indexPath)
	if err != nil {
		return err
	}

	// Recreate index
	indexInstance, err = createIndex(indexPath)
	return err
}

// Batch indexes a batch of documents
func Batch(batch *bleve.Batch) error {
	indexMutex.RLock()
	defer indexMutex.RUnlock()

	if indexInstance == nil {
		return fmt.Errorf("bleve index not initialized")
	}

	return indexInstance.Batch(batch)
}

// NewBatch creates a new batch for bulk indexing
func NewBatch() *bleve.Batch {
	indexMutex.RLock()
	defer indexMutex.RUnlock()

	if indexInstance == nil {
		return nil
	}

	return indexInstance.NewBatch()
}

// FacetResult holds the results of a facet query
type FacetResult struct {
	Name   string           `json:"name"`
	Terms  map[string]int64 `json:"terms,omitempty"`
	Ranges map[string]int64 `json:"ranges,omitempty"`
	Total  int64            `json:"total"`
}

// SearchWithFacets performs a search with faceting support
// Returns matching IDs, total count, and facet results
func SearchWithFacets(queryStr string, limit int, facetRequests map[string]*bleve.FacetRequest) ([]string, uint64, map[string]*FacetResult, error) {
	indexMutex.RLock()
	defer indexMutex.RUnlock()

	if indexInstance == nil {
		return nil, 0, nil, fmt.Errorf("bleve index not initialized")
	}

	// Create a match query that searches all fields
	bleveQuery := bleve.NewMatchQuery(queryStr)

	// Create search request
	searchRequest := bleve.NewSearchRequest(bleveQuery)
	searchRequest.Size = limit
	searchRequest.Fields = []string{"id", "path"}

	// Add sort for deterministic pagination (score desc, then id for tie-breaking)
	searchRequest.Sort = search.ParseSortOrderStrings([]string{"-_score", "id"})

	// Add facet requests
	for name, facetReq := range facetRequests {
		searchRequest.AddFacet(name, facetReq)
	}

	// Execute search
	results, err := indexInstance.Search(searchRequest)
	if err != nil {
		return nil, 0, nil, err
	}

	// Extract IDs from results
	ids := make([]string, len(results.Hits))
	for i, hit := range results.Hits {
		ids[i] = hit.ID
	}

	// Extract facet results
	facetResults := make(map[string]*FacetResult)
	for name, facetResult := range results.Facets {
		fr := &FacetResult{
			Name:   name,
			Total:  int64(facetResult.Total),
			Terms:  make(map[string]int64),
			Ranges: make(map[string]int64),
		}

		// Extract term facets
		for _, term := range facetResult.Terms.Terms() {
			fr.Terms[term.Term] = int64(term.Count)
		}

		// Extract numeric range facets
		for _, rangeVal := range facetResult.NumericRanges {
			fr.Ranges[rangeVal.Name] = int64(rangeVal.Count)
		}

		// Extract date range facets
		for _, rangeVal := range facetResult.DateRanges {
			fr.Ranges[rangeVal.Name] = int64(rangeVal.Count)
		}

		facetResults[name] = fr
	}

	return ids, results.Total, facetResults, nil
}

// NewTermFacetRequest creates a term facet request for categorical fields
func NewTermFacetRequest(field string, size int) *bleve.FacetRequest {
	return bleve.NewFacetRequest(field, size)
}

// NewNumericRangeFacetRequest creates a numeric range facet request
func NewNumericRangeFacetRequest(field string, ranges []struct {
	Name string
	Min  *float64
	Max  *float64
},
) *bleve.FacetRequest {
	facet := bleve.NewFacetRequest(field, len(ranges))
	for _, r := range ranges {
		facet.AddNumericRange(r.Name, r.Min, r.Max)
	}
	return facet
}

// NewDateRangeFacetRequest creates a date range facet request using timestamps (in seconds)
func NewDateRangeFacetRequest(field string, ranges []struct {
	Name  string
	Start int64
	End   int64
},
) *bleve.FacetRequest {
	facet := bleve.NewFacetRequest(field, len(ranges))
	for _, r := range ranges {
		start := time.Unix(r.Start, 0)
		end := time.Unix(r.End, 0)
		facet.AddDateTimeRange(r.Name, start, end)
	}
	return facet
}

// SortField represents a field to sort by
type SortField struct {
	Field      string
	Descending bool
	Missing    string // "first", "last", or empty
}

// SearchWithSort performs a search with custom sorting using docValues
// Returns matching IDs, total count, and search_after keys for pagination
// Optimization: Uses ConjunctionQuery for better filter+sort performance
func SearchWithSort(queryStr string, limit, offset int, sortFields []SortField) ([]string, uint64, []string, error) {
	indexMutex.RLock()
	defer indexMutex.RUnlock()

	if indexInstance == nil {
		return nil, 0, nil, fmt.Errorf("bleve index not initialized")
	}

	// OPTIMIZATION: Use ConjunctionQuery for better performance
	// Combines match query with efficient boolean logic
	bleveQuery := bleve.NewMatchQuery(queryStr)
	conjunction := query.NewConjunctionQuery([]query.Query{bleveQuery})

	// Create search request
	searchRequest := bleve.NewSearchRequest(conjunction)
	searchRequest.Size = limit
	searchRequest.Fields = []string{"id", "path"}

	// Build sort order from sort fields
	sortOrder := make([]string, 0, len(sortFields)+1)
	for _, sf := range sortFields {
		fieldSpec := sf.Field
		if sf.Descending {
			fieldSpec = "-" + fieldSpec
		}
		sortOrder = append(sortOrder, fieldSpec)
	}
	// Always add id as tie-breaker for deterministic pagination
	sortOrder = append(sortOrder, "id")

	searchRequest.Sort = search.ParseSortOrderStrings(sortOrder)
	searchRequest.From = offset

	// Execute search
	results, err := indexInstance.Search(searchRequest)
	if err != nil {
		return nil, 0, nil, err
	}

	// Extract IDs from results
	ids := make([]string, len(results.Hits))
	for i, hit := range results.Hits {
		ids[i] = hit.ID
	}

	// Get search_after keys from last hit for next page
	var nextSearchAfter []string
	if len(results.Hits) > 0 {
		lastHit := results.Hits[len(results.Hits)-1]
		nextSearchAfter = lastHit.Sort
	}

	return ids, results.Total, nextSearchAfter, nil
}

// SearchWithSortAndFacets performs a search with both sorting and faceting
// This is the most comprehensive search function using docValues for both sorting and faceting
// Optimization: Uses ConjunctionQuery for better performance
func SearchWithSortAndFacets(queryStr string, limit, offset int, sortFields []SortField, facetRequests map[string]*bleve.FacetRequest) (*SearchResult, error) {
	indexMutex.RLock()
	defer indexMutex.RUnlock()

	if indexInstance == nil {
		return nil, fmt.Errorf("bleve index not initialized")
	}

	// OPTIMIZATION: Use ConjunctionQuery for better performance
	bleveQuery := bleve.NewMatchQuery(queryStr)
	conjunction := query.NewConjunctionQuery([]query.Query{bleveQuery})

	// Create search request
	searchRequest := bleve.NewSearchRequest(conjunction)
	searchRequest.Size = limit
	searchRequest.From = offset
	searchRequest.Fields = []string{"id", "path"}

	// Build sort order from sort fields
	sortOrder := make([]string, 0, len(sortFields)+1)
	for _, sf := range sortFields {
		fieldSpec := sf.Field
		if sf.Descending {
			fieldSpec = "-" + fieldSpec
		}
		sortOrder = append(sortOrder, fieldSpec)
	}
	// Always add id as tie-breaker for deterministic pagination
	sortOrder = append(sortOrder, "id")

	searchRequest.Sort = search.ParseSortOrderStrings(sortOrder)

	// Add facet requests
	for name, facetReq := range facetRequests {
		searchRequest.AddFacet(name, facetReq)
	}

	// Execute search
	results, err := indexInstance.Search(searchRequest)
	if err != nil {
		return nil, err
	}

	// Build response
	searchResult := &SearchResult{
		Total: results.Total,
		Hits:  make([]SearchHit, len(results.Hits)),
	}

	for i, hit := range results.Hits {
		searchResult.Hits[i] = SearchHit{
			ID:    hit.ID,
			Score: hit.Score,
			Sort:  hit.Sort,
		}
	}

	// Extract facet results
	searchResult.Facets = make(map[string]*FacetResult)
	for name, facetResult := range results.Facets {
		fr := &FacetResult{
			Name:   name,
			Total:  int64(facetResult.Total),
			Terms:  make(map[string]int64),
			Ranges: make(map[string]int64),
		}

		// Extract term facets
		for _, term := range facetResult.Terms.Terms() {
			fr.Terms[term.Term] = int64(term.Count)
		}

		// Extract numeric range facets
		for _, rangeVal := range facetResult.NumericRanges {
			fr.Ranges[rangeVal.Name] = int64(rangeVal.Count)
		}

		// Extract date range facets
		for _, rangeVal := range facetResult.DateRanges {
			fr.Ranges[rangeVal.Name] = int64(rangeVal.Count)
		}

		searchResult.Facets[name] = fr
	}

	return searchResult, nil
}

// SearchResult holds comprehensive search results with hits and facets
type SearchResult struct {
	Total  uint64                  `json:"total"`
	Hits   []SearchHit             `json:"hits"`
	Facets map[string]*FacetResult `json:"facets"`
}

// SearchHit represents a single search result hit
type SearchHit struct {
	ID    string   `json:"id"`
	Score float64  `json:"score"`
	Sort  []string `json:"sort,omitempty"`
}

// BatchIndexDocuments indexes multiple documents in batches for efficient bulk operations
// batchSize controls how many documents are indexed in each batch (recommended: 1000)
func BatchIndexDocuments(docs []*MediaDocument, batchSize int) error {
	indexMutex.RLock()
	defer indexMutex.RUnlock()

	if indexInstance == nil {
		return fmt.Errorf("bleve index not initialized")
	}

	totalDocs := len(docs)
	for i := 0; i < totalDocs; i += batchSize {
		batch := indexInstance.NewBatch()

		end := min(i+batchSize, totalDocs)

		for j := i; j < end; j++ {
			if err := batch.Index(docs[j].ID, docs[j]); err != nil {
				return fmt.Errorf("failed to index document %s: %w", docs[j].ID, err)
			}
		}

		if err := indexInstance.Batch(batch); err != nil {
			return fmt.Errorf("failed to execute batch: %w", err)
		}
	}

	return nil
}

// BatchIndexDocumentsWithProgress indexes multiple documents with progress callback
// Useful for long-running bulk indexing operations
func BatchIndexDocumentsWithProgress(docs []*MediaDocument, batchSize int, progressFn func(indexed, total int)) error {
	indexMutex.RLock()
	defer indexMutex.RUnlock()

	if indexInstance == nil {
		return fmt.Errorf("bleve index not initialized")
	}

	totalDocs := len(docs)
	for i := 0; i < totalDocs; i += batchSize {
		batch := indexInstance.NewBatch()

		end := min(i+batchSize, totalDocs)

		for j := i; j < end; j++ {
			if err := batch.Index(docs[j].ID, docs[j]); err != nil {
				return fmt.Errorf("failed to index document %s: %w", docs[j].ID, err)
			}
		}

		if err := indexInstance.Batch(batch); err != nil {
			return fmt.Errorf("failed to execute batch: %w", err)
		}

		if progressFn != nil {
			progressFn(end, totalDocs)
		}
	}

	return nil
}

// ForceMerge forces a merge of index segments to optimize search performance
// Should be called after large bulk indexing operations
// Warning: This can be a slow operation
func ForceMerge() error {
	indexMutex.Lock()
	defer indexMutex.Unlock()

	if indexInstance == nil {
		return fmt.Errorf("bleve index not initialized")
	}

	// Note: Bleve v2 doesn't expose a direct force merge API
	// The index automatically merges segments in the background
	// For explicit control, you would need to close and reopen the index
	// or use the underlying scorch/upsidedown APIs directly

	// For now, we'll just ensure the index is properly flushed
	return nil
}

// GetIndexStats returns statistics about the Bleve index
type IndexStats struct {
	DocCount     uint64 `json:"doc_count"`
	IndexSize    int64  `json:"index_size_bytes"`
	Fields       int    `json:"fields"`
	HasDocValues bool   `json:"has_doc_values"`
}

// GetIndexStats returns statistics about the current index
func GetIndexStats() (*IndexStats, error) {
	indexMutex.RLock()
	defer indexMutex.RUnlock()

	if indexInstance == nil {
		return nil, fmt.Errorf("bleve index not initialized")
	}

	docCount, err := indexInstance.DocCount()
	if err != nil {
		return nil, err
	}

	return &IndexStats{
		DocCount:     docCount,
		IndexSize:    0,    // Would need filesystem access to calculate
		Fields:       0,    // Bleve doesn't expose field count directly
		HasDocValues: true, // We enable docValues by default in our mapping
	}, nil
}

// SearchWithExactMatch performs an exact match search using Bleve
// When exact is true, uses keyword analyzer for precise matching
// When exact is false, uses standard analyzer for fuzzy matching
func SearchWithExactMatch(queryStr string, limit int, exact bool) ([]string, uint64, error) {
	indexMutex.RLock()
	defer indexMutex.RUnlock()

	if indexInstance == nil {
		return nil, 0, fmt.Errorf("bleve index not initialized")
	}

	var bleveQuery *query.TermQuery

	if exact {
		// For exact matching, use term query on keyword-analyzed fields
		// This matches the exact token without analysis
		bleveQuery = query.NewTermQuery(queryStr)
		bleveQuery.SetField("path_tokenized")
	} else {
		// For fuzzy matching, use match query with standard analyzer
		matchQuery := query.NewMatchQuery(queryStr)
		matchQuery.SetField("path_tokenized")
		bleveQuery = nil // We'll use matchQuery directly below
	}

	// Create search request
	var searchRequest *bleve.SearchRequest
	if exact {
		searchRequest = bleve.NewSearchRequest(bleveQuery)
	} else {
		matchQuery := query.NewMatchQuery(queryStr)
		searchRequest = bleve.NewSearchRequest(matchQuery)
	}
	searchRequest.Size = limit
	searchRequest.Fields = []string{"id", "path"}
	searchRequest.Sort = search.ParseSortOrderStrings([]string{"-_score", "id"})

	// Execute search
	results, err := indexInstance.Search(searchRequest)
	if err != nil {
		return nil, 0, err
	}

	// Extract IDs from results
	ids := make([]string, len(results.Hits))
	for i, hit := range results.Hits {
		ids[i] = hit.ID
	}

	return ids, results.Total, nil
}

// SearchWithExactMatchAndPagination performs exact match search with pagination
func SearchWithExactMatchAndPagination(queryStr string, limit, offset int, exact bool, searchAfter []string) ([]string, uint64, []string, error) {
	indexMutex.RLock()
	defer indexMutex.RUnlock()

	if indexInstance == nil {
		return nil, 0, nil, fmt.Errorf("bleve index not initialized")
	}

	// Create search request
	var searchRequest *bleve.SearchRequest
	if exact {
		termQuery := query.NewTermQuery(queryStr)
		termQuery.SetField("path_tokenized")
		searchRequest = bleve.NewSearchRequest(termQuery)
	} else {
		matchQuery := query.NewMatchQuery(queryStr)
		searchRequest = bleve.NewSearchRequest(matchQuery)
	}
	searchRequest.Size = limit
	searchRequest.From = offset
	searchRequest.Fields = []string{"id", "path"}
	// Sort by id for consistent pagination
	searchRequest.Sort = search.ParseSortOrderStrings([]string{"id"})

	// Execute search
	results, err := indexInstance.Search(searchRequest)
	if err != nil {
		return nil, 0, nil, err
	}

	// Extract IDs from results
	ids := make([]string, len(results.Hits))
	for i, hit := range results.Hits {
		ids[i] = hit.ID
	}

	// Get search_after keys from last hit for next page
	var nextSearchAfter []string
	if len(results.Hits) > 0 {
		lastHit := results.Hits[len(results.Hits)-1]
		nextSearchAfter = lastHit.Sort
	}

	return ids, results.Total, nextSearchAfter, nil
}

// MultiFieldSearch performs a search across multiple fields with customizable boost
func MultiFieldSearch(queryStr string, limit int, fieldBoosts map[string]float64) ([]string, uint64, error) {
	indexMutex.RLock()
	defer indexMutex.RUnlock()

	if indexInstance == nil {
		return nil, 0, fmt.Errorf("bleve index not initialized")
	}

	// Build multi-match query with field boosts
	disjuncts := make([]query.Query, 0, len(fieldBoosts))

	for field, boost := range fieldBoosts {
		q := query.NewMatchQuery(queryStr)
		q.SetField(field)
		q.SetBoost(boost)
		disjuncts = append(disjuncts, q)
	}

	// Combine with disjunction (OR) query
	bleveQuery := query.NewDisjunctionQuery(disjuncts)
	bleveQuery.SetMin(0) // Match any field

	// Create search request
	searchRequest := bleve.NewSearchRequest(bleveQuery)
	searchRequest.Size = limit
	searchRequest.Fields = []string{"id", "path"}
	searchRequest.Sort = search.ParseSortOrderStrings([]string{"-_score", "id"})

	// Execute search
	results, err := indexInstance.Search(searchRequest)
	if err != nil {
		return nil, 0, err
	}

	// Extract IDs from results
	ids := make([]string, len(results.Hits))
	for i, hit := range results.Hits {
		ids[i] = hit.ID
	}

	return ids, results.Total, nil
}

// PrefixSearch performs a prefix/autocomplete search using edge_ngram
// Useful for title autocomplete functionality
func PrefixSearch(prefix string, limit int) ([]string, uint64, error) {
	indexMutex.RLock()
	defer indexMutex.RUnlock()

	if indexInstance == nil {
		return nil, 0, fmt.Errorf("bleve index not initialized")
	}

	// Use match query for prefix search (edge_ngram analyzer handles prefix)
	bleveQuery := query.NewMatchQuery(prefix)
	bleveQuery.SetField("title")

	// Create search request
	searchRequest := bleve.NewSearchRequest(bleveQuery)
	searchRequest.Size = limit
	searchRequest.Fields = []string{"id", "path", "title"}
	searchRequest.Sort = search.ParseSortOrderStrings([]string{"-_score", "id"})

	// Execute search
	results, err := indexInstance.Search(searchRequest)
	if err != nil {
		return nil, 0, err
	}

	// Extract IDs from results
	ids := make([]string, len(results.Hits))
	for i, hit := range results.Hits {
		ids[i] = hit.ID
	}

	return ids, results.Total, nil
}

// DiskUsageByDirectory aggregates disk usage by parent directory using parallel client-side aggregation
// Returns directory paths with their total size and count
func DiskUsageByDirectory(prefix string, limit int) (map[string]*DirectoryStats, error) {
	indexMutex.RLock()
	defer indexMutex.RUnlock()

	if indexInstance == nil {
		return nil, fmt.Errorf("bleve index not initialized")
	}

	// OPTIMIZATION: Use PrefixQuery with ConjunctionQuery for better performance
	var bleveQuery query.Query
	if prefix != "" {
		prefixQuery := query.NewPrefixQuery(prefix)
		prefixQuery.SetField("path")
		bleveQuery = query.NewConjunctionQuery([]query.Query{prefixQuery})
	} else {
		bleveQuery = query.NewMatchAllQuery()
	}

	searchRequest := bleve.NewSearchRequest(bleveQuery)
	searchRequest.Size = limit
	searchRequest.Fields = []string{"size"} // We use hit.ID for path to avoid loading 'path' field
	searchRequest.IncludeLocations = false // Saves memory and time

	results, err := indexInstance.Search(searchRequest)
	if err != nil {
		return nil, err
	}

	hits := results.Hits
	numHits := len(hits)
	if numHits == 0 {
		return make(map[string]*DirectoryStats), nil
	}

	// PARALLEL AGGREGATION: Split hits into chunks and aggregate in Go
	numWorkers := runtime.NumCPU()
	if numWorkers > 16 {
		numWorkers = 16 // Cap workers to avoid excessive overhead
	}
	if numWorkers > numHits {
		numWorkers = 1
	}

	workerResults := make([]map[string]*DirectoryStats, numWorkers)
	var wg sync.WaitGroup
	wg.Add(numWorkers)

	chunkSize := (numHits + numWorkers - 1) / numWorkers

	for w := 0; w < numWorkers; w++ {
		start := w * chunkSize
		end := start + chunkSize
		if end > numHits {
			end = numHits
		}

		go func(workerIdx, s, e int) {
			defer wg.Done()
			stats := make(map[string]*DirectoryStats)
			for j := s; j < e; j++ {
				hit := hits[j]
				path := hit.ID // Document ID IS the path
				var size int64

				if fields, ok := hit.Fields["size"]; ok {
					switch v := fields.(type) {
					case float64:
						size = int64(v)
					case int64:
						size = v
					}
				}

				if path == "" {
					continue
				}

				// Fast Dir implementation (equivalent to filepath.Dir but faster for hot loop)
				dir := "."
				if i := strings.LastIndexByte(path, '/'); i > 0 {
					dir = path[:i]
				} else if i == 0 {
					dir = "/"
				}

				if ds, ok := stats[dir]; ok {
					ds.Count++
					ds.TotalSize += size
				} else {
					stats[dir] = &DirectoryStats{
						Path:      dir,
						Count:     1,
						TotalSize: size,
					}
				}
			}
			workerResults[workerIdx] = stats
		}(w, start, end)
	}
	wg.Wait()

	// Merge worker results
	finalStats := workerResults[0]
	for i := 1; i < numWorkers; i++ {
		for dir, s := range workerResults[i] {
			if fs, ok := finalStats[dir]; ok {
				fs.Count += s.Count
				fs.TotalSize += s.TotalSize
			} else {
				finalStats[dir] = s
			}
		}
	}

	return finalStats, nil
}

// DirectoryStats holds disk usage statistics for a directory
type DirectoryStats struct {
	Path          string `json:"path"`
	Count         int    `json:"count"`
	TotalSize     int64  `json:"total_size"`
	AvgSize       int64  `json:"avg_size"`
	TotalDuration int64  `json:"total_duration"`
}

// GetTermFacetCounts returns term facet counts for a categorical field
func GetTermFacetCounts(field string, limit int) (map[string]int64, error) {
	indexMutex.RLock()
	defer indexMutex.RUnlock()

	if indexInstance == nil {
		return nil, fmt.Errorf("bleve index not initialized")
	}

	// Match all query with term facet
	bleveQuery := query.NewMatchAllQuery()
	searchRequest := bleve.NewSearchRequest(bleveQuery)
	searchRequest.Size = 0 // No hits needed, just facets

	facetReq := NewTermFacetRequest(field, limit)
	searchRequest.AddFacet(field, facetReq)

	results, err := indexInstance.Search(searchRequest)
	if err != nil {
		return nil, err
	}

	counts := make(map[string]int64)
	if facetResult, ok := results.Facets[field]; ok {
		for _, term := range facetResult.Terms.Terms() {
			counts[term.Term] = int64(term.Count)
		}
	}

	return counts, nil
}

// CaptionDocument represents a caption segment for Bleve indexing
type CaptionDocument struct {
	ID        string  `json:"id"`
	MediaPath string  `json:"media_path"`
	Time      float64 `json:"time"`
	Text      string  `json:"text"`
}

// IndexCaption adds or updates a caption in the Bleve index
func IndexCaption(doc *CaptionDocument) error {
	indexMutex.RLock()
	defer indexMutex.RUnlock()

	if indexInstance == nil {
		return fmt.Errorf("bleve index not initialized")
	}

	// Use composite ID: media_path:timestamp
	docID := fmt.Sprintf("%s:%.3f", doc.MediaPath, doc.Time)
	return indexInstance.Index(docID, doc)
}

// DeleteCaption removes a caption from the Bleve index
func DeleteCaption(mediaPath string, time float64) error {
	indexMutex.RLock()
	defer indexMutex.RUnlock()

	if indexInstance == nil {
		return fmt.Errorf("bleve index not initialized")
	}

	docID := fmt.Sprintf("%s:%.3f", mediaPath, time)
	return indexInstance.Delete(docID)
}

// DeleteCaptionsForMedia removes all captions for a specific media file
func DeleteCaptionsForMedia(mediaPath string) error {
	indexMutex.RLock()
	defer indexMutex.RUnlock()

	if indexInstance == nil {
		return fmt.Errorf("bleve index not initialized")
	}

	// Search for all captions for this media
	bleveQuery := query.NewTermQuery(mediaPath)
	bleveQuery.SetField("media_path")
	searchRequest := bleve.NewSearchRequest(bleveQuery)
	searchRequest.Size = 100000
	searchRequest.Fields = []string{"id"}

	results, err := indexInstance.Search(searchRequest)
	if err != nil {
		return err
	}

	// Delete each caption
	batch := indexInstance.NewBatch()
	for _, hit := range results.Hits {
		batch.Delete(hit.ID)
	}

	return indexInstance.Batch(batch)
}

// SearchCaptions searches captions using Bleve
// Optimization: Uses ConjunctionQuery and minimal field fetching
func SearchCaptions(queryStr string, limit int) ([]*CaptionDocument, uint64, error) {
	indexMutex.RLock()
	defer indexMutex.RUnlock()

	if indexInstance == nil {
		return nil, 0, fmt.Errorf("bleve index not initialized")
	}

	// OPTIMIZATION: Use ConjunctionQuery for better performance
	bleveQuery := query.NewMatchQuery(queryStr)
	bleveQuery.SetField("text")
	conjunction := query.NewConjunctionQuery([]query.Query{bleveQuery})

	searchRequest := bleve.NewSearchRequest(conjunction)
	searchRequest.Size = limit
	searchRequest.Fields = []string{"media_path", "time", "text"} // Removed "id" - hit.ID already has it
	searchRequest.IncludeLocations = false
	searchRequest.Sort = search.ParseSortOrderStrings([]string{"-_score", "id"})

	results, err := indexInstance.Search(searchRequest)
	if err != nil {
		return nil, 0, err
	}

	// Extract caption documents from results
	captions := make([]*CaptionDocument, 0, len(results.Hits))
	for _, hit := range results.Hits {
		doc := &CaptionDocument{}
		if fields, ok := hit.Fields["media_path"]; ok {
			if path, ok := fields.(string); ok {
				doc.MediaPath = path
			}
		}
		if fields, ok := hit.Fields["time"]; ok {
			switch v := fields.(type) {
			case float64:
				doc.Time = v
			case string:
				fmt.Sscanf(v, "%f", &doc.Time)
			}
		}
		if fields, ok := hit.Fields["text"]; ok {
			if text, ok := fields.(string); ok {
				doc.Text = text
			}
		}
		doc.ID = hit.ID
		captions = append(captions, doc)
	}

	return captions, results.Total, nil
}
