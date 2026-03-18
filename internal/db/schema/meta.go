package schema

// GetMetaTables returns the metadata tables SQL
func GetMetaTables() string {
	data, err := SchemaFS.ReadFile("meta.sql")
	if err != nil {
		panic("meta.sql not found: " + err.Error())
	}
	return string(data)
}
