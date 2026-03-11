//go:build !bleve

package query

// BleveSearch executes a Bleve search and returns matching paths
func BleveSearch(searchTerms []string, limit int) ([]string, error) {
	return nil, nil
}

// HasBleveIndex checks if a Bleve index is available
func HasBleveIndex() bool {
	return false
}
