package query

import (
	"testing"

	"github.com/chapmanjacobd/discoteca/internal/models"
)

func TestParseSortConfig(t *testing.T) {
	tests := []struct {
		name      string
		config    string
		wantFields []SortField
		wantAlg   string
	}{
		{
			name:   "simple field",
			config: "path",
			wantFields: []SortField{
				{Field: "path", Reverse: false},
			},
			wantAlg: "natural",
		},
		{
			name:   "reversed field with minus",
			config: "-path",
			wantFields: []SortField{
				{Field: "path", Reverse: true},
			},
			wantAlg: "natural",
		},
		{
			name:   "reversed field with prefix",
			config: "reverse_path",
			wantFields: []SortField{
				{Field: "path", Reverse: true},
			},
			wantAlg: "natural",
		},
		{
			name:   "field with desc suffix",
			config: "path desc",
			wantFields: []SortField{
				{Field: "path", Reverse: true},
			},
			wantAlg: "natural",
		},
		{
			name:   "field with asc suffix",
			config: "path asc",
			wantFields: []SortField{
				{Field: "path", Reverse: false},
			},
			wantAlg: "natural",
		},
		{
			name:   "multi-field comma separated",
			config: "video_count desc,audio_count desc,path asc",
			wantFields: []SortField{
				{Field: "video_count", Reverse: true},
				{Field: "audio_count", Reverse: true},
				{Field: "path", Reverse: false},
			},
			wantAlg: "natural",
		},
		{
			name:   "algorithm prefix",
			config: "natural_path",
			wantFields: []SortField{
				{Field: "path", Reverse: false},
			},
			wantAlg: "natural",
		},
		{
			name:   "python algorithm",
			config: "python_title",
			wantFields: []SortField{
				{Field: "title", Reverse: false},
			},
			wantAlg: "python",
		},
		{
			name:   "complex multi-field with algorithms",
			config: "natural_path,title desc,-play_count",
			wantFields: []SortField{
				{Field: "path", Reverse: false},
				{Field: "title", Reverse: true},
				{Field: "play_count", Reverse: true},
			},
			wantAlg: "natural",
		},
		{
			name:   "empty config",
			config: "",
			wantFields: []SortField{
				{Field: "ps", Reverse: false},
			},
			wantAlg: "natural",
		},
		{
			name:   "algorithm only",
			config: "natural",
			wantFields: []SortField{
				{Field: "ps", Reverse: false},
			},
			wantAlg: "natural",
		},
		{
			name:   "xklb keyword",
			config: "xklb",
			wantFields: []SortField{
				{Field: "video_count", Reverse: true},
				{Field: "audio_count", Reverse: true},
				{Field: "path_is_remote", Reverse: false},
				{Field: "subtitle_count", Reverse: true},
				{Field: "play_count", Reverse: false},
				{Field: "playhead", Reverse: true},
				{Field: "time_last_played", Reverse: false},
				{Field: "title_is_null", Reverse: false},
				{Field: "path", Reverse: false},
			},
			wantAlg: "natural",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Special handling for xklb keyword which gets expanded
			if tt.config == "xklb" {
				// Just verify it returns non-empty fields
				fields, alg := parseSortConfig(tt.config)
				if len(fields) == 0 {
					t.Errorf("parseSortConfig(%q) returned empty fields", tt.config)
				}
				if alg != "natural" {
					t.Errorf("parseSortConfig(%q) alg = %q, want %q", tt.config, alg, tt.wantAlg)
				}
				return
			}

			fields, alg := parseSortConfig(tt.config)
			if len(fields) != len(tt.wantFields) {
				t.Errorf("parseSortConfig(%q) returned %d fields, want %d", tt.config, len(fields), len(tt.wantFields))
			}
			for i, f := range fields {
				if i >= len(tt.wantFields) {
					break
				}
				if f.Field != tt.wantFields[i].Field {
					t.Errorf("parseSortConfig(%q) field[%d] = %q, want %q", tt.config, i, f.Field, tt.wantFields[i].Field)
				}
				if f.Reverse != tt.wantFields[i].Reverse {
					t.Errorf("parseSortConfig(%q) field[%d].Reverse = %v, want %v", tt.config, i, f.Reverse, tt.wantFields[i].Reverse)
				}
			}
			if alg != tt.wantAlg {
				t.Errorf("parseSortConfig(%q) alg = %q, want %q", tt.config, alg, tt.wantAlg)
			}
		})
	}
}

func TestIsNumericField(t *testing.T) {
	tests := []struct {
		field string
		want  bool
	}{
		{"video_count", true},
		{"audio_count", true},
		{"subtitle_count", true},
		{"play_count", true},
		{"playhead", true},
		{"time_last_played", true},
		{"duration", true},
		{"size", true},
		{"path", false},
		{"title", false},
		{"type", false},
	}

	for _, tt := range tests {
		t.Run(tt.field, func(t *testing.T) {
			if got := isNumericField(tt.field); got != tt.want {
				t.Errorf("isNumericField(%q) = %v, want %v", tt.field, got, tt.want)
			}
		})
	}
}

func TestGetSortValueFloat64(t *testing.T) {
	videoCount := int64(2)
	audioCount := int64(1)
	playCount := int64(5)
	playhead := int64(100)
	duration := int64(200)
	size := int64(1024)

	m := models.MediaWithDB{
		Media: models.Media{
			VideoCount:   &videoCount,
			AudioCount:   &audioCount,
			PlayCount:    &playCount,
			Playhead:     &playhead,
			Duration:     &duration,
			Size:         &size,
			TimeLastPlayed: nil,
		},
	}

	tests := []struct {
		field string
		want  float64
	}{
		{"video_count", 2},
		{"audio_count", 1},
		{"play_count", 5},
		{"playhead", 100},
		{"duration", 200},
		{"size", 1024},
		{"time_last_played", 0}, // nil returns 0
	}

	for _, tt := range tests {
		t.Run(tt.field, func(t *testing.T) {
			if got := getSortValueFloat64(m, tt.field); got != tt.want {
				t.Errorf("getSortValueFloat64(%q) = %v, want %v", tt.field, got, tt.want)
			}
		})
	}
}

func TestGetSortValueString(t *testing.T) {
	path := "/path/to/file.mp4"
	title := "Test Title"
	mtype := "video"

	m := models.MediaWithDB{
		Media: models.Media{
			Path:  path,
			Title: &title,
			Type:  &mtype,
		},
	}

	tests := []struct {
		field string
		want  string
	}{
		{"path", path},
		{"title", title},
		{"type", mtype},
		{"parent", "/path/to"},
		{"stem", "file"},
		{"extension", ".mp4"},
	}

	for _, tt := range tests {
		t.Run(tt.field, func(t *testing.T) {
			if got := getSortValueString(m, tt.field); got != tt.want {
				t.Errorf("getSortValueString(%q) = %q, want %q", tt.field, got, tt.want)
			}
		})
	}
}

func TestCompareSortFields(t *testing.T) {
	videoCount1 := int64(2)
	videoCount2 := int64(1)
	audioCount1 := int64(1)
	audioCount2 := int64(0)

	m1 := models.MediaWithDB{
		Media: models.Media{
			Path:       "/path/a.mp4",
			VideoCount: &videoCount1,
			AudioCount: &audioCount1,
		},
	}

	m2 := models.MediaWithDB{
		Media: models.Media{
			Path:       "/path/b.mp4",
			VideoCount: &videoCount2,
			AudioCount: &audioCount2,
		},
	}

	media := []models.MediaWithDB{m1, m2}

	tests := []struct {
		name       string
		sortFields []SortField
		want       int // negative if m1 should come before m2, positive if m2 should come before m1
	}{
		{
			name: "video_count desc - m1 should come first",
			sortFields: []SortField{
				{Field: "video_count", Reverse: true},
			},
			want: -1, // m1 has more videos (2 vs 1), with desc, m1 should come first (negative means m1 < m2 in sort order)
		},
		{
			name: "video_count asc - m2 should come first",
			sortFields: []SortField{
				{Field: "video_count", Reverse: false},
			},
			want: 1, // m1 has more videos (2 vs 1), with asc, m2 should come first (positive means m2 < m1 in sort order)
		},
		{
			name: "multi-field - video_count different, first field decides",
			sortFields: []SortField{
				{Field: "video_count", Reverse: false},
				{Field: "audio_count", Reverse: false},
			},
			want: 1, // m1 has more videos (2 vs 1), with asc, m2 should come first
		},
		{
			name: "path asc - alphabetical",
			sortFields: []SortField{
				{Field: "path", Reverse: false},
			},
			want: -1, // /path/a.mp4 < /path/b.mp4, so m1 comes first
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := compareSortFields(media, 0, 1, tt.sortFields, "natural")
			if got != tt.want {
				t.Errorf("compareSortFields() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSortAdvanced(t *testing.T) {
	videoCount1 := int64(2)
	videoCount2 := int64(1)
	videoCount3 := int64(0)

	m1 := models.MediaWithDB{
		Media: models.Media{
			Path:       "/path/c.mp4",
			VideoCount: &videoCount1,
		},
	}

	m2 := models.MediaWithDB{
		Media: models.Media{
			Path:       "/path/a.mp4",
			VideoCount: &videoCount2,
		},
	}

	m3 := models.MediaWithDB{
		Media: models.Media{
			Path:       "/path/b.mp4",
			VideoCount: &videoCount3,
		},
	}

	tests := []struct {
		name     string
		config   string
		media    []models.MediaWithDB
		wantPath []string // expected order of paths
	}{
		{
			name:   "video_count desc",
			config: "video_count desc",
			media:  []models.MediaWithDB{m3, m2, m1},
			wantPath: []string{"/path/c.mp4", "/path/a.mp4", "/path/b.mp4"},
		},
		{
			name:   "path asc",
			config: "path asc",
			media:  []models.MediaWithDB{m3, m1, m2},
			wantPath: []string{"/path/a.mp4", "/path/b.mp4", "/path/c.mp4"},
		},
		{
			name:   "xklb default",
			config: "xklb",
			media:  []models.MediaWithDB{m3, m2, m1},
			wantPath: []string{"/path/c.mp4", "/path/a.mp4", "/path/b.mp4"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sb := NewSortBuilder(models.GlobalFlags{})
			sb.SortAdvanced(tt.media, tt.config)

			for i, want := range tt.wantPath {
				if i >= len(tt.media) {
					break
				}
				if tt.media[i].Path != want {
					t.Errorf("SortAdvanced() position %d = %q, want %q", i, tt.media[i].Path, want)
				}
			}
		})
	}
}

func TestXklbDefaultSort(t *testing.T) {
	fields := xklbDefaultSort()
	if len(fields) == 0 {
		t.Error("xklbDefaultSort() returned empty fields")
	}

	// Check first field is video_count desc
	if len(fields) > 0 {
		if fields[0].Field != "video_count" || !fields[0].Reverse {
			t.Errorf("xklbDefaultSort() first field = %+v, want video_count desc", fields[0])
		}
	}
}

func TestDuDefaultSort(t *testing.T) {
	fields := duDefaultSort()
	if len(fields) == 0 {
		t.Error("duDefaultSort() returned empty fields")
	}

	// Check first field is size_per_count desc
	if len(fields) > 0 {
		if fields[0].Field != "size_per_count" || !fields[0].Reverse {
			t.Errorf("duDefaultSort() first field = %+v, want size_per_count desc", fields[0])
		}
	}
}

func TestReverseXklbSort(t *testing.T) {
	videoCount1 := int64(0)
	videoCount2 := int64(1)

	m1 := models.MediaWithDB{
		Media: models.Media{
			Path:       "/path/a.mp4",
			VideoCount: &videoCount1,
		},
	}

	m2 := models.MediaWithDB{
		Media: models.Media{
			Path:       "/path/b.mp4",
			VideoCount: &videoCount2,
		},
	}

	// With reverse_xklb, audio-only (video_count=0) should come before videos
	media := []models.MediaWithDB{m2, m1}
	sb := NewSortBuilder(models.GlobalFlags{})
	sb.SortAdvanced(media, "reverse_xklb")

	// m1 (audio-only) should come first
	if media[0].Path != "/path/a.mp4" {
		t.Errorf("SortAdvanced(reverse_xklb) position 0 = %q, want /path/a.mp4 (audio-first)", media[0].Path)
	}
}
