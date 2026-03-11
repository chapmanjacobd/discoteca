//go:build bleve

package bleve

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/blevesearch/bleve/v2"
	"github.com/blevesearch/bleve/v2/analysis/analyzer/keyword"
	"github.com/blevesearch/bleve/v2/analysis/analyzer/standard"
	"github.com/chapmanjacobd/discoteca/internal/models"
)

var (
	indexInstance bleve.Index
	indexMutex    sync.RWMutex
	indexPath     string
)

// MediaDocument represents a document to be indexed in Bleve
type MediaDocument struct {
	ID          string `json:"id"`
	Path        string `json:"path"`
	FtsPath     string `json:"fts_path"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Type        string `json:"type"`
	Size        int64  `json:"size"`
	Duration    int64  `json:"duration"`
}

// ToBleveDoc converts a Media model to a BleveDocument
func ToBleveDoc(m models.Media) *MediaDocument {
	doc := &MediaDocument{
		ID:   m.Path,
		Path: m.Path,
	}

	if m.FtsPath != nil {
		doc.FtsPath = *m.FtsPath
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
	pathField := bleve.NewTextFieldMapping()
	pathField.Analyzer = keyword.Name // For exact path matching

	ftsPathField := bleve.NewTextFieldMapping()
	ftsPathField.Analyzer = standard.Name // For full-text search

	titleField := bleve.NewTextFieldMapping()
	titleField.Analyzer = standard.Name
	titleField.IncludeInAll = true

	descriptionField := bleve.NewTextFieldMapping()
	descriptionField.Analyzer = standard.Name
	descriptionField.IncludeInAll = true

	typeField := bleve.NewTextFieldMapping()
	typeField.Analyzer = keyword.Name

	// Create document mapping
	docMapping := bleve.NewDocumentMapping()
	docMapping.AddFieldMappingsAt("path", pathField)
	docMapping.AddFieldMappingsAt("fts_path", ftsPathField)
	docMapping.AddFieldMappingsAt("title", titleField)
	docMapping.AddFieldMappingsAt("description", descriptionField)
	docMapping.AddFieldMappingsAt("type", typeField)

	// Create index mapping
	indexMapping := bleve.NewIndexMapping()
	indexMapping.DefaultMapping = docMapping
	indexMapping.DefaultAnalyzer = standard.Name

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
	bleveQuery.FieldVal = "fts_path"

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
