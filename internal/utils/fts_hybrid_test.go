package utils

import (
	"testing"
)

func TestParseHybridSearchQuery(t *testing.T) {
	tests := []struct {
		name         string
		query        string
		wantFTSTerms []string
		wantPhrases  []string
	}{
		{
			name:         "simple term search",
			query:        "video tutorial",
			wantFTSTerms: []string{"video", "tutorial"},
			wantPhrases:  []string{},
		},
		{
			name:         "phrase search",
			query:        `"video tutorial"`,
			wantFTSTerms: []string{},
			wantPhrases:  []string{"video tutorial"},
		},
		{
			name:         "mixed search",
			query:        `python "video tutorial" beginner`,
			wantFTSTerms: []string{"python", "beginner"},
			wantPhrases:  []string{"video tutorial"},
		},
		{
			name:         "multiple phrases",
			query:        `"machine learning" "deep learning"`,
			wantFTSTerms: []string{},
			wantPhrases:  []string{"machine learning", "deep learning"},
		},
		{
			name:         "short phrase ignored",
			query:        `"ab" video`,
			wantFTSTerms: []string{"video"},
			wantPhrases:  []string{}, // "ab" is < 3 chars
		},
		{
			name:         "single quotes",
			query:        `'video tutorial' python`,
			wantFTSTerms: []string{"python"},
			wantPhrases:  []string{"video tutorial"},
		},
		{
			name:         "boolean operators",
			query:        "video OR tutorial NOT music",
			wantFTSTerms: []string{"video", "OR", "tutorial", "NOT", "music"},
			wantPhrases:  []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ParseHybridSearchQuery(tt.query)

			// Check FTS terms
			if len(got.FTSTerms) != len(tt.wantFTSTerms) {
				t.Errorf("FTSTerms = %v, want %v", got.FTSTerms, tt.wantFTSTerms)
				return
			}
			for i, term := range got.FTSTerms {
				if term != tt.wantFTSTerms[i] {
					t.Errorf("FTSTerms[%d] = %q, want %q", i, term, tt.wantFTSTerms[i])
				}
			}

			// Check phrases
			if len(got.Phrases) != len(tt.wantPhrases) {
				t.Errorf("Phrases = %v, want %v", got.Phrases, tt.wantPhrases)
				return
			}
			for i, phrase := range got.Phrases {
				if phrase != tt.wantPhrases[i] {
					t.Errorf("Phrases[%d] = %q, want %q", i, phrase, tt.wantPhrases[i])
				}
			}
		})
	}
}

func TestHybridSearchQuery_BuildFTSQuery(t *testing.T) {
	tests := []struct {
		name      string
		terms     []string
		joinOp    string
		wantQuery string
	}{
		{
			name:      "simple OR join",
			terms:     []string{"video", "tutorial"},
			joinOp:    " OR ",
			wantQuery: "vid OR tut",
		},
		{
			name:      "with boolean operators",
			terms:     []string{"video", "OR", "tutorial"},
			joinOp:    "OR",
			wantQuery: "vid OR tut",
		},
		{
			name:      "empty terms",
			terms:     []string{},
			joinOp:    "OR",
			wantQuery: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := &HybridSearchQuery{FTSTerms: tt.terms}
			got := h.BuildFTSQuery(tt.joinOp)
			if got != tt.wantQuery {
				t.Errorf("BuildFTSQuery() = %q, want %q", got, tt.wantQuery)
			}
		})
	}
}
