//go:build !bleve

package bleve

import (
	"fmt"

	"github.com/chapmanjacobd/discoteca/internal/models"
)

// MediaDocument represents a document to be indexed in Bleve
type MediaDocument struct {
	ID            string `json:"id"`
	Path          string `json:"path"`
	PathTokenized string `json:"path_tokenized"`
	Title         string `json:"title"`
	Description   string `json:"description"`
	Type          string `json:"type"`
	Size          int64  `json:"size"`
	Duration      int64  `json:"duration"`
}

// ToBleveDoc converts a Media model to a BleveDocument
func ToBleveDoc(m models.Media) *MediaDocument {
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
