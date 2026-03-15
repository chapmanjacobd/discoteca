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
	Episodes   []FilterBin `json:"episodes"`
	Size       []FilterBin `json:"size"`
	Duration   []FilterBin `json:"duration"`
	Type       []FilterBin `json:"type"`
	Modified   []FilterBin `json:"modified"`
	Created    []FilterBin `json:"created"`
	Downloaded []FilterBin `json:"downloaded"`

	EpisodesMin   int64 `json:"episodes_min"`
	EpisodesMax   int64 `json:"episodes_max"`
	SizeMin       int64 `json:"size_min"`
	SizeMax       int64 `json:"size_max"`
	DurationMin   int64 `json:"duration_min"`
	DurationMax   int64 `json:"duration_max"`
	ModifiedMin   int64 `json:"modified_min"`
	ModifiedMax   int64 `json:"modified_max"`
	CreatedMin    int64 `json:"created_min"`
	CreatedMax    int64 `json:"created_max"`
	DownloadedMin int64 `json:"downloaded_min"`
	DownloadedMax int64 `json:"downloaded_max"`

	EpisodesPercentiles   []int64 `json:"episodes_percentiles"`
	SizePercentiles       []int64 `json:"size_percentiles"`
	DurationPercentiles   []int64 `json:"duration_percentiles"`
	ModifiedPercentiles   []int64 `json:"modified_percentiles"`
	CreatedPercentiles    []int64 `json:"created_percentiles"`
	DownloadedPercentiles []int64 `json:"downloaded_percentiles"`
}

type PlaylistResponse []string

type ErrorResponse struct {
	Error string `json:"error"`
}
