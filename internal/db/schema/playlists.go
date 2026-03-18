package schema

// GetPlaylistsTables returns the playlists tables SQL
func GetPlaylistsTables() string {
	data, err := SchemaFS.ReadFile("playlists.sql")
	if err != nil {
		panic("playlists.sql not found: " + err.Error())
	}
	return string(data)
}
