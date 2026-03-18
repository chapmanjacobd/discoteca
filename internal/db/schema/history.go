package schema

// GetHistoryTable returns the history table and indexes SQL
func GetHistoryTable() string {
	data, err := SchemaFS.ReadFile("history.sql")
	if err != nil {
		panic("history.sql not found: " + err.Error())
	}
	return string(data)
}
