package schema

// GetCaptionsTable returns the captions table SQL
func GetCaptionsTable() string {
	data, err := SchemaFS.ReadFile("captions.sql")
	if err != nil {
		panic("captions.sql not found: " + err.Error())
	}
	return string(data)
}

// GetCaptionsFTS returns the captions FTS SQL
func GetCaptionsFTS() string {
	data, err := SchemaFS.ReadFile("captions_fts.sql")
	if err != nil {
		panic("captions_fts.sql not found: " + err.Error())
	}
	return string(data)
}
