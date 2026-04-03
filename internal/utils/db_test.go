package utils

import (
	"strings"
	"testing"
)

func TestMostSimilarSchema(t *testing.T) {
	existingTables := map[string]map[string]string{
		"table1": {"id": "INT", "name": "VARCHAR", "age": "INT"},
		"table2": {"id": "INT", "email": "VARCHAR", "phone": "VARCHAR"},
	}

	tests := []struct {
		name     string
		keys     []string
		expected string
	}{
		{"Exact Match", []string{"id", "name", "age"}, "table1"},
		{"Partial Match", []string{"id", "name", "address", "age"}, "table1"},
		{"No Match", []string{"salary", "position", "department"}, ""},
		{"Empty Input", []string{}, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := MostSimilarSchema(tt.keys, existingTables)
			if got != tt.expected {
				t.Errorf("MostSimilarSchema() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestBlockDictsLikeSQL(t *testing.T) {
	media := []map[string]any{
		{"title": "Movie 1", "genre": "Action"},
		{"title": "Movie 2", "genre": "Comedy"},
		{"title": "Movie 3", "genre": "Drama"},
		{"title": "Movie 4", "genre": "Thriller"},
	}
	blocklist := []map[string]any{{"genre": "Comedy"}, {"genre": "Thriller"}}

	got := BlockDictsLikeSQL(media, blocklist)
	if len(got) != 2 {
		t.Errorf("BlockDictsLikeSQL() len = %d, want 2", len(got))
	}
}

func TestAllowDictsLikeSQL(t *testing.T) {
	media := []map[string]any{
		{"title": "Movie 1", "genre": "Action"},
		{"title": "Movie 2", "genre": "Comedy"},
		{"title": "Movie 3", "genre": "Drama"},
		{"title": "Movie 4", "genre": "Thriller"},
	}
	allowlist := []map[string]any{{"genre": "Comedy"}, {"genre": "Thriller"}}

	got := AllowDictsLikeSQL(media, allowlist)
	if len(got) != 2 {
		t.Errorf("AllowDictsLikeSQL() len = %d, want 2", len(got))
	}
}

func TestConstructSearchBindings(t *testing.T) {
	t.Run("Includes", func(t *testing.T) {
		sql, bindings := ConstructSearchBindings([]string{"test"}, []string{}, []string{"col1", "col2"}, false)
		if !strings.Contains(sql, "col1 LIKE :S_include0 OR col2 LIKE :S_include0") {
			t.Errorf("ConstructSearchBindings() sql = %q", sql)
		}
		if bindings["S_include0"] != "%test%" {
			t.Errorf("ConstructSearchBindings() bindings = %v", bindings)
		}
	})

	t.Run("Exact Match", func(t *testing.T) {
		sql, bindings := ConstructSearchBindings([]string{"test"}, []string{}, []string{"col1"}, true)
		if !strings.Contains(sql, "col1 LIKE :S_include0") {
			t.Errorf("ConstructSearchBindings() sql = %q", sql)
		}
		if bindings["S_include0"] != "test" {
			t.Errorf("ConstructSearchBindings() bindings = %v", bindings)
		}
	})

	t.Run("Excludes", func(t *testing.T) {
		sql, bindings := ConstructSearchBindings([]string{}, []string{"test"}, []string{"col1", "col2"}, false)
		if !strings.Contains(
			sql,
			"AND ((COALESCE(col1,'') NOT LIKE :S_exclude0 AND COALESCE(col2,'') NOT LIKE :S_exclude0))",
		) {
			t.Errorf("ConstructSearchBindings() sql = %q", sql)
		}
		if bindings["S_exclude0"] != "%test%" {
			t.Errorf("ConstructSearchBindings() bindings = %v", bindings)
		}
	})
}
