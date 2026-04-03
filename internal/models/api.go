package models

type CatStat struct {
	Category string `json:"category"`
	Count    int64  `json:"count"`
}

type RatStat struct {
	Rating int64 `json:"rating"`
	Count  int64 `json:"count"`
}

type GenreStat struct {
	Genre string `json:"genre"`
	Count int64  `json:"count"`
}

type DatabaseInfo struct {
	Databases []string `json:"databases"`
	ReadOnly  bool     `json:"read_only"`
	Dev       bool     `json:"dev"`
}

type PlayResponse struct {
	Path string `json:"path"`
}

type DeleteRequest struct {
	Path    string `json:"path"`
	Restore bool   `json:"restore"`
}

type ProgressRequest struct {
	Path      string `json:"path"`
	Playhead  int64  `json:"playhead"`
	Duration  int64  `json:"duration"`
	Completed bool   `json:"completed"`
}

type LsEntry struct {
	Path      string `json:"path"`
	Name      string `json:"name"`
	IsDir     bool   `json:"is_dir"`
	MediaType string `json:"media_type"`
}

type FilterBin struct {
	Label string `json:"label"`
	Min   int64  `json:"min,omitempty"`
	Max   int64  `json:"max,omitempty"`
	Value int64  `json:"value,omitempty"`
}

type FilterBinsResponse struct {
	// Actual min/max values from the full dataset (stable across filtering)
	EpisodesMinVal   int64 `json:"episodes_min_val"`
	EpisodesMaxVal   int64 `json:"episodes_max_val"`
	SizeMinVal       int64 `json:"size_min_val"`
	SizeMaxVal       int64 `json:"size_max_val"`
	DurationMinVal   int64 `json:"duration_min_val"`
	DurationMaxVal   int64 `json:"duration_max_val"`
	ModifiedMinVal   int64 `json:"modified_min_val"`
	ModifiedMaxVal   int64 `json:"modified_max_val"`
	CreatedMinVal    int64 `json:"created_min_val"`
	CreatedMaxVal    int64 `json:"created_max_val"`
	DownloadedMinVal int64 `json:"downloaded_min_val"`
	DownloadedMaxVal int64 `json:"downloaded_max_val"`

	// Percentiles for slider calculations (0-100)
	EpisodesPercentiles   []int64 `json:"episodes_percentiles"`
	SizePercentiles       []int64 `json:"size_percentiles"`
	DurationPercentiles   []int64 `json:"duration_percentiles"`
	ModifiedPercentiles   []int64 `json:"modified_percentiles"`
	CreatedPercentiles    []int64 `json:"created_percentiles"`
	DownloadedPercentiles []int64 `json:"downloaded_percentiles"`

	// Media type counts (special case - not a percentile distribution)
	MediaType []FilterBin `json:"media_type"`
}

type PlaylistResponse []string

type ErrorResponse struct {
	Error string `json:"error"`
}
