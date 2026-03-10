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
	Trashcan  bool     `json:"trashcan"`
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
	Path  string `json:"path"`
	Name  string `json:"name"`
	IsDir bool   `json:"is_dir"`
	Type  string `json:"type"`
}

type FilterBin struct {
	Label string `json:"label"`
	Min   int64  `json:"min,omitempty"`
	Max   int64  `json:"max,omitempty"`
	Value int64  `json:"value,omitempty"`
}

type FilterBinsResponse struct {
	Episodes []FilterBin `json:"episodes"`
	Size     []FilterBin `json:"size"`
	Duration []FilterBin `json:"duration"`
	Type     []FilterBin `json:"type"`

	EpisodesMin int64 `json:"episodes_min"`
	EpisodesMax int64 `json:"episodes_max"`
	SizeMin     int64 `json:"size_min"`
	SizeMax     int64 `json:"size_max"`
	DurationMin int64 `json:"duration_min"`
	DurationMax int64 `json:"duration_max"`

	EpisodesPercentiles []int64 `json:"episodes_percentiles"`
	SizePercentiles     []int64 `json:"size_percentiles"`
	DurationPercentiles []int64 `json:"duration_percentiles"`
}

type PlaylistResponse []string

type ErrorResponse struct {
	Error string `json:"error"`
}
