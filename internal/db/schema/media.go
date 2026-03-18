package schema

// GetMediaTable returns the media table SQL
func GetMediaTable() string {
	data, err := SchemaFS.ReadFile("media.sql")
	if err != nil {
		panic("media.sql not found: " + err.Error())
	}
	return string(data)
}

// GetMediaIndexes returns the media indexes SQL
func GetMediaIndexes() string {
	data, err := SchemaFS.ReadFile("media_indexes.sql")
	if err != nil {
		panic("media_indexes.sql not found: " + err.Error())
	}
	return string(data)
}

// GetMediaFTS returns the media FTS SQL
func GetMediaFTS() string {
	data, err := SchemaFS.ReadFile("media_fts.sql")
	if err != nil {
		panic("media_fts.sql not found: " + err.Error())
	}
	return string(data)
}
