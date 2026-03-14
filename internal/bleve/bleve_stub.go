//go:build !bleve

package bleve

import (
	"fmt"

	"github.com/blevesearch/bleve/v2"
	"github.com/chapmanjacobd/discoteca/internal/db"
	"github.com/chapmanjacobd/discoteca/internal/models"
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
	return nil
}

// ToBleveDocFromUpsert converts UpsertMediaParams to a BleveDocument
func ToBleveDocFromUpsert(p db.UpsertMediaParams) *MediaDocument {
	return nil
}

// InitIndex initializes or opens the Bleve index
func InitIndex(dbPath string) error {
	return fmt.Errorf("bleve support not enabled in this build")
}

// CloseIndex closes the Bleve index
func CloseIndex() error {
	return nil
}

// GetIndex returns the current Bleve index instance
func GetIndex() any {
	return nil
}

// IndexDocument adds or updates a document in the index
func IndexDocument(doc *MediaDocument) error {
	return fmt.Errorf("bleve support not enabled in this build")
}

// DeleteDocument removes a document from the index
func DeleteDocument(id string) error {
	return fmt.Errorf("bleve support not enabled in this build")
}

// Search performs a search query on the Bleve index
func Search(query string, limit int) ([]string, float64, error) {
	return nil, 0, fmt.Errorf("bleve support not enabled in this build")
}

// SearchPath performs a path-specific search
func SearchPath(pathPattern string, limit int) ([]string, error) {
	return nil, fmt.Errorf("bleve support not enabled in this build")
}

// Count returns the total number of documents in the index
func Count() (uint64, error) {
	return 0, fmt.Errorf("bleve support not enabled in this build")
}

// ReindexAll rebuilds the entire index
func ReindexAll() error {
	return fmt.Errorf("bleve support not enabled in this build")
}

// Batch indexes a batch of documents
func Batch(batch *bleve.Batch) error {
	return fmt.Errorf("bleve support not enabled in this build")
}

// NewBatch creates a new batch for bulk indexing
func NewBatch() *bleve.Batch {
	return nil
}

// FacetResult holds the results of a facet query
type FacetResult struct {
	Name   string           `json:"name"`
	Terms  map[string]int64 `json:"terms,omitempty"`
	Ranges map[string]int64 `json:"ranges,omitempty"`
	Total  int64            `json:"total"`
}

// SearchWithFacets performs a search with faceting support
func SearchWithFacets(query string, limit int, facetRequests map[string]*bleve.FacetRequest) ([]string, uint64, map[string]*FacetResult, error) {
	return nil, 0, nil, fmt.Errorf("bleve support not enabled in this build")
}

// NewTermFacetRequest creates a term facet request for categorical fields
func NewTermFacetRequest(field string, size int) *bleve.FacetRequest {
	return nil
}

// NewNumericRangeFacetRequest creates a numeric range facet request
func NewNumericRangeFacetRequest(field string, ranges []struct {
	Name string
	Min  *float64
	Max  *float64
},
) *bleve.FacetRequest {
	return nil
}

// NewDateRangeFacetRequest creates a date range facet request (timestamps in seconds)
func NewDateRangeFacetRequest(field string, ranges []struct {
	Name string
	Min  int64
	Max  int64
},
) *bleve.FacetRequest {
	return nil
}

// SortField represents a field to sort by
type SortField struct {
	Field      string
	Descending bool
	Missing    string
}

// SearchWithSort performs a search with custom sorting using docValues
func SearchWithSort(query string, limit, offset int, sortFields []SortField) ([]string, uint64, []string, error) {
	return nil, 0, nil, fmt.Errorf("bleve support not enabled in this build")
}

// SearchWithSortAndFacets performs a search with both sorting and faceting
func SearchWithSortAndFacets(query string, limit, offset int, sortFields []SortField, facetRequests map[string]*bleve.FacetRequest) (*SearchResult, error) {
	return nil, fmt.Errorf("bleve support not enabled in this build")
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
func BatchIndexDocuments(docs []*MediaDocument, batchSize int) error {
	return fmt.Errorf("bleve support not enabled in this build")
}

// BatchIndexDocumentsWithProgress indexes multiple documents with progress callback
func BatchIndexDocumentsWithProgress(docs []*MediaDocument, batchSize int, progressFn func(indexed, total int)) error {
	return fmt.Errorf("bleve support not enabled in this build")
}

// ForceMerge forces a merge of index segments to optimize search performance
func ForceMerge() error {
	return fmt.Errorf("bleve support not enabled in this build")
}

// IndexStats holds statistics about the Bleve index
type IndexStats struct {
	DocCount     uint64 `json:"doc_count"`
	IndexSize    int64  `json:"index_size_bytes"`
	Fields       int    `json:"fields"`
	HasDocValues bool   `json:"has_doc_values"`
}

// GetIndexStats returns statistics about the current index
func GetIndexStats() (*IndexStats, error) {
	return nil, fmt.Errorf("bleve support not enabled in this build")
}

// SearchWithExactMatch performs an exact match search using Bleve
func SearchWithExactMatch(query string, limit int, exact bool) ([]string, uint64, error) {
	return nil, 0, fmt.Errorf("bleve support not enabled in this build")
}

// SearchWithExactMatchAndPagination performs exact match search with pagination
func SearchWithExactMatchAndPagination(query string, limit, offset int, exact bool, searchAfter []string) ([]string, uint64, []string, error) {
	return nil, 0, nil, fmt.Errorf("bleve support not enabled in this build")
}

// MultiFieldSearch performs a search across multiple fields with customizable boost
func MultiFieldSearch(query string, limit int, fieldBoosts map[string]float64) ([]string, uint64, error) {
	return nil, 0, fmt.Errorf("bleve support not enabled in this build")
}

// PrefixSearch performs a prefix/autocomplete search using edge_ngram
func PrefixSearch(prefix string, limit int) ([]string, uint64, error) {
	return nil, 0, fmt.Errorf("bleve support not enabled in this build")
}

// DirectoryStats holds disk usage statistics for a directory
type DirectoryStats struct {
	Path          string `json:"path"`
	Count         int    `json:"count"`
	TotalSize     int64  `json:"total_size"`
	AvgSize       int64  `json:"avg_size"`
	TotalDuration int64  `json:"total_duration"`
}

// DiskUsageByDirectory aggregates disk usage by parent directory using Bleve facets
func DiskUsageByDirectory(prefix string, limit int) (map[string]*DirectoryStats, error) {
	return nil, fmt.Errorf("bleve support not enabled in this build")
}

// GetTermFacetCounts returns term facet counts for a categorical field
func GetTermFacetCounts(field string, limit int) (map[string]int64, error) {
	return nil, fmt.Errorf("bleve support not enabled in this build")
}
