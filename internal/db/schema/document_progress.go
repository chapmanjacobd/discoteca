package schema

// GetDocumentProgressTable returns the document progress table SQL.
func GetDocumentProgressTable() string {
	data, err := SchemaFS.ReadFile("document_progress.sql")
	if err != nil {
		panic("document_progress.sql not found: " + err.Error())
	}
	return string(data)
}
