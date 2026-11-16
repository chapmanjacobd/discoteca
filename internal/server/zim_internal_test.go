package server

import "testing"

func TestValidateKiwixConfig(t *testing.T) {
	tests := []struct {
		name    string
		port    int
		wantErr bool
	}{
		{"disabled", 0, false},
		{"default port", 8181, false},
		{"maximum port", 65535, false},
		{"negative port", -1, true},
		{"above maximum", 65536, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := &Server{KiwixPort: tt.port}
			err := cmd.validateKiwixConfig()
			if (err != nil) != tt.wantErr {
				t.Fatalf("validateKiwixConfig() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateZimanticConfig(t *testing.T) {
	tests := []struct {
		name    string
		port    int
		wantErr bool
	}{
		{"disabled", 0, false},
		{"standard port", 8090, false},
		{"maximum port", 65535, false},
		{"negative port", -1, true},
		{"above maximum", 65536, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := &Server{ZimanticPort: tt.port}
			err := cmd.validateZimanticConfig()
			if (err != nil) != tt.wantErr {
				t.Fatalf("validateZimanticConfig() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestZimPermaRef(t *testing.T) {
	tests := map[string]string{
		"wikipedia_en_all_maxi_2024-01":   "wikipedia_en_all_maxi",
		"wikimed_en_all_2024-01-15":       "wikimed_en_all",
		"wiktionary_en_all_nopic_2024-01": "wiktionary_en_all_nopic",
		"my-archive":                      "my-archive",
		"archive_2024-01-extra":           "archive_2024-01-extra",
		"archive-2024-01":                 "archive-2024-01",
	}
	for input, want := range tests {
		if got := zimPermaRef(input); got != want {
			t.Errorf("zimPermaRef(%q) = %q, want %q", input, got, want)
		}
	}
}

func TestZimanticSlug(t *testing.T) {
	tests := map[string]string{
		"Wikipedia_en_all_maxi": "wikipedia_en_all_maxi",
		"testy_2024-01":         "testy_2024-01",
		"!!!":                   "source",
		"a b  c":                "a-b-c",
		"archive.tar.zim":       "archive.tar.zim",
	}
	for input, want := range tests {
		if got := zimanticSlug(input); got != want {
			t.Errorf("zimanticSlug(%q) = %q, want %q", input, got, want)
		}
	}
}
