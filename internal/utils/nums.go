package utils

import (
	"fmt"
	"math"
	"math/rand"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

type Number interface {
	~int | ~int8 | ~int16 | ~int32 | ~int64 | ~uint | ~uint8 | ~uint16 | ~uint32 | ~uint64 | ~uintptr | ~float32 | ~float64
}

func RandomFloat() float64 {
	return rand.Float64()
}

func RandomInt(min, max int) int {
	if min >= max {
		return min
	}
	return rand.Intn(max-min) + min
}

func LinearInterpolation(x float64, dataPoints [][2]float64) float64 {
	if len(dataPoints) == 0 {
		return 0
	}
	// Assume dataPoints are sorted by x
	n := len(dataPoints)
	if x < dataPoints[0][0] {
		return dataPoints[0][1]
	}
	if x > dataPoints[n-1][0] {
		return dataPoints[n-1][1]
	}

	for i := range n - 1 {
		if x >= dataPoints[i][0] && x <= dataPoints[i+1][0] {
			x1, y1 := dataPoints[i][0], dataPoints[i][1]
			x2, y2 := dataPoints[i+1][0], dataPoints[i+1][1]
			return y1 + ((x-x1)/(x2-x1))*(y2-y1)
		}
	}
	return dataPoints[n-1][1]
}

func SafeMean[T Number](slice []T) float64 {
	if len(slice) == 0 {
		return 0
	}
	var sum float64
	for _, v := range slice {
		sum += float64(v)
	}
	return sum / float64(len(slice))
}

func SafeMedian[T Number](slice []T) float64 {
	if len(slice) == 0 {
		return 0
	}
	sorted := make([]float64, len(slice))
	for i, v := range slice {
		sorted[i] = float64(v)
	}
	sort.Float64s(sorted)
	n := len(sorted)
	if n%2 == 1 {
		return sorted[n/2]
	}
	return (sorted[n/2-1] + sorted[n/2]) / 2
}

func Percentile[T Number](slice []T, p float64) float64 {
	if len(slice) == 0 {
		return 0
	}
	sorted := make([]float64, len(slice))
	for i, v := range slice {
		sorted[i] = float64(v)
	}
	sort.Float64s(sorted)

	if p <= 0 {
		return sorted[0]
	}
	if p >= 100 {
		return sorted[len(sorted)-1]
	}

	index := p / 100.0 * float64(len(sorted)-1)
	i := int(index)
	fraction := index - float64(i)

	if i >= len(sorted)-1 {
		return sorted[len(sorted)-1]
	}

	return sorted[i]*(1-fraction) + sorted[i+1]*fraction
}

func HumanToBytes(s string) (int64, error) {
	s = strings.ToUpper(strings.TrimSpace(s))

	suffixes := []struct {
		suffix string
		mult   int64
	}{
		{"KB", 1000},
		{"MB", 1000 * 1000},
		{"GB", 1000 * 1000 * 1000},
		{"TB", 1000 * 1000 * 1000 * 1000},
		{"K", 1000},
		{"M", 1000 * 1000},
		{"G", 1000 * 1000 * 1000},
		{"T", 1000 * 1000 * 1000 * 1000},
		{"B", 1},
	}

	for _, entry := range suffixes {
		if before, ok := strings.CutSuffix(s, entry.suffix); ok {
			numStr := strings.TrimSpace(before)
			num, err := strconv.ParseFloat(numStr, 64)
			if err != nil {
				return 0, err
			}
			return int64(num * float64(entry.mult)), nil
		}
	}

	num, err := strconv.ParseInt(s, 10, 64)
	return num, err
}

func HumanToBits(s string) (int64, error) {
	s = strings.ToUpper(strings.TrimSpace(s))

	suffixes := []struct {
		suffix string
		mult   int64
	}{
		{"KBIT", 1000},
		{"MBIT", 1000 * 1000},
		{"GBIT", 1000 * 1000 * 1000},
		{"TBIT", 1000 * 1000 * 1000 * 1000},
		{"K", 1000},
		{"M", 1000 * 1000},
		{"G", 1000 * 1000 * 1000},
		{"T", 1000 * 1000 * 1000 * 1000},
		{"BIT", 1},
	}

	for _, entry := range suffixes {
		if before, ok := strings.CutSuffix(s, entry.suffix); ok {
			numStr := strings.TrimSpace(before)
			num, err := strconv.ParseFloat(numStr, 64)
			if err != nil {
				return 0, err
			}
			return int64(num * float64(entry.mult)), nil
		}
	}

	num, err := strconv.ParseInt(s, 10, 64)
	return num, err
}

func HumanToSeconds(s string) (int64, error) {
	s = strings.ToLower(strings.TrimSpace(s))
	if s == "" {
		return 0, nil
	}

	multipliers := []struct {
		suffix string
		mult   int64
	}{
		{"minutes", 60},
		{"seconds", 1},
		{"months", 2592000},
		{"weeks", 604800},
		{"hours", 3600},
		{"years", 31536000},
		{"minute", 60},
		{"second", 1},
		{"month", 2592000},
		{"week", 604800},
		{"hour", 3600},
		{"year", 31536000},
		{"mins", 60},
		{"secs", 1},
		{"min", 60},
		{"sec", 1},
		{"days", 86400},
		{"day", 86400},
		{"mon", 2592000},
		{"mo", 2592000},
		{"yr", 31536000},
		{"hr", 3600},
		{"w", 604800},
		{"d", 86400},
		{"h", 3600},
		{"m", 60},
		{"s", 1},
		{"y", 31536000},
	}

	for _, entry := range multipliers {
		if before, ok := strings.CutSuffix(s, entry.suffix); ok {
			numStr := strings.TrimSpace(before)
			num, err := strconv.ParseFloat(numStr, 64)
			if err != nil {
				return 0, err
			}
			return int64(num * float64(entry.mult)), nil
		}
	}

	// Default to seconds
	return strconv.ParseInt(s, 10, 64)
}

func ParseRange(s string, humanToX func(string) (int64, error)) (Range, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return Range{}, nil
	}

	if strings.Contains(s, ",") {
		parts := strings.Split(s, ",")
		var merged Range
		for _, p := range parts {
			r, err := ParseRange(p, humanToX)
			if err != nil {
				return Range{}, err
			}
			if r.Min != nil {
				merged.Min = r.Min
			}
			if r.Max != nil {
				merged.Max = r.Max
			}
			if r.Value != nil {
				merged.Value = r.Value
			}
		}
		return merged, nil
	}

	if strings.Contains(s, "-") && !strings.HasPrefix(s, "-") {
		parts := strings.Split(s, "-")
		if len(parts) == 2 {
			min, err := humanToX(parts[0])
			if err != nil {
				return Range{}, err
			}
			max, err := humanToX(parts[1])
			if err != nil {
				return Range{}, err
			}
			return Range{Min: &min, Max: &max}, nil
		}
	}

	if strings.Contains(s, "%") {
		parts := strings.Split(s, "%")
		base, err := humanToX(parts[0])
		if err != nil {
			return Range{}, err
		}
		percent, err := strconv.ParseFloat(parts[1], 64)
		if err != nil {
			return Range{}, err
		}
		tolerance := int64(float64(base) * (percent / 100.0))
		min := base - tolerance
		max := base + tolerance
		return Range{Min: &min, Max: &max}, nil
	}

	if strings.HasPrefix(s, ">") {
		min, err := humanToX(s[1:])
		if err != nil {
			return Range{}, err
		}
		min++ // strictly greater
		return Range{Min: &min}, nil
	}
	if strings.HasPrefix(s, "<") {
		max, err := humanToX(s[1:])
		if err != nil {
			return Range{}, err
		}
		max-- // strictly less
		return Range{Max: &max}, nil
	}
	if strings.HasPrefix(s, "+") {
		min, err := humanToX(s[1:])
		if err != nil {
			return Range{}, err
		}
		return Range{Min: &min}, nil
	}
	if strings.HasPrefix(s, "-") {
		max, err := humanToX(s[1:])
		if err != nil {
			return Range{}, err
		}
		return Range{Max: &max}, nil
	}

	val, err := humanToX(s)
	if err != nil {
		return Range{}, err
	}
	return Range{Value: &val}, nil
}

func CalculatePercentiles(values []int64) []int64 {
	if len(values) == 0 {
		return make([]int64, 101)
	}

	// 1. Collect frequencies
	counts := make(map[int64]int)
	for _, v := range values {
		counts[v]++
	}

	type valFreq struct {
		val  int64
		freq float64
	}
	unique := make([]valFreq, 0, len(counts))
	for v, c := range counts {
		unique = append(unique, valFreq{val: v, freq: float64(c) / float64(len(values))})
	}
	sort.Slice(unique, func(i, j int) bool {
		return unique[i].val < unique[j].val
	})

	nUnique := float64(len(unique))
	uFreq := 1.0 / nUnique

	// 2. Calculate lambda to satisfy the 5% cap (0.05)
	// We want lambda*freq + (1-lambda)*uFreq <= 0.05
	lambda := 1.0
	for _, f := range unique {
		if f.freq > 0.05 {
			if f.freq > uFreq {
				l := (0.05 - uFreq) / (f.freq - uFreq)
				if l < lambda {
					lambda = l
				}
			}
		}
	}
	if lambda < 0 {
		lambda = 0
	}

	// 3. Apply lambda and calculate cumulative distribution
	for i := range unique {
		unique[i].freq = lambda*unique[i].freq + (1.0-lambda)*uFreq
	}

	res := make([]int64, 101)
	cum := 0.0
	uIdx := 0
	for i := range 101 {
		target := float64(i) / 100.0
		for uIdx < len(unique)-1 && cum+unique[uIdx].freq < target {
			cum += unique[uIdx].freq
			uIdx++
		}
		res[i] = unique[uIdx].val
	}

	return res
}

func Percent(value, total float64) float64 {
	if total == 0 {
		return 0
	}
	return (value / total) * 100
}

func FloatFromPercent(s string) (float64, error) {
	if before, ok := strings.CutSuffix(s, "%"); ok {
		v, err := strconv.ParseFloat(before, 64)
		if err != nil {
			return 0, err
		}
		return v / 100, nil
	}
	return strconv.ParseFloat(s, 64)
}

func PercentageDifference(v1, v2 float64) float64 {
	if v1+v2 == 0 {
		return 100.0
	}
	return math.Abs((v1-v2)/((v1+v2)/2)) * 100
}

func ParsePercentileRange(s string) (min, max float64, ok bool) {
	if !strings.HasPrefix(s, "p") {
		return 0, 0, false
	}
	s = s[1:]
	parts := strings.Split(s, "-")
	if len(parts) != 2 {
		return 0, 0, false
	}
	var err error
	min, err = strconv.ParseFloat(parts[0], 64)
	if err != nil {
		return 0, 0, false
	}
	max, err = strconv.ParseFloat(parts[1], 64)
	if err != nil {
		return 0, 0, false
	}
	return min, max, true
}

func CalculateSegments(total, chunk, gap float64) []float64 {
	if total <= 0 || chunk <= 0 {
		return nil
	}
	if total <= chunk*3 {
		return []float64{0}
	}

	var segments []float64
	start := 0.0
	endSegmentStart := total - chunk

	g := gap
	if g < 1 {
		g = math.Ceil(total * gap)
	}

	for start+chunk < endSegmentStart {
		segments = append(segments, start)
		start += chunk + g
	}

	return append(segments, endSegmentStart)
}

func CalculateSegmentsInt(total, chunk int64, gap float64) []int64 {
	if total <= 0 || chunk <= 0 {
		return nil
	}
	if total <= chunk*3 {
		return []int64{0}
	}

	var segments []int64
	start := int64(0)
	endSegmentStart := total - chunk

	var g int64
	if gap < 1 {
		g = int64(math.Ceil(float64(total) * gap))
	} else {
		g = int64(gap)
	}

	for start+chunk < endSegmentStart {
		segments = append(segments, start)
		start += chunk + g
	}

	return append(segments, endSegmentStart)
}

func SafeInt(s string) *int {
	if s == "" {
		return nil
	}
	f, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return nil
	}
	i := int(f)
	return &i
}

func SafeFloat(s string) *float64 {
	if s == "" {
		return nil
	}
	f, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return nil
	}
	return &f
}

func SQLHumanTime(s string) string {
	if _, err := strconv.Atoi(s); err == nil {
		return s + " minutes"
	}

	unitMapping := map[string]string{
		"min":  "minutes",
		"mins": "minutes",
		"s":    "seconds",
		"sec":  "seconds",
		"secs": "seconds",
	}

	re := regexp.MustCompile(`(\d+\.?\d*)([a-zA-Z]+)`)
	match := re.FindStringSubmatch(s)
	if match != nil {
		value := match[1]
		unit := strings.ToLower(strings.TrimSpace(match[2]))
		if mapped, ok := unitMapping[unit]; ok {
			unit = mapped
		}
		return fmt.Sprintf("%s %s", value, unit)
	}
	return s
}

func Max[T Number](a, b T) T {
	if a > b {
		return a
	}
	return b
}

func Min[T Number](a, b T) T {
	if a < b {
		return a
	}
	return b
}

func BoolToInt64(b bool) int64 {
	if b {
		return 1
	}
	return 0
}

// ParsePercentOrBytes parses a string like "5%" or "100MB" into a float64 ratio or bytes
func ParsePercentOrBytes(s string) float64 {
	s = strings.TrimSpace(s)
	if before, ok := strings.CutSuffix(s, "%"); ok {
		v, err := strconv.ParseFloat(before, 64)
		if err != nil {
			return 0.05 // default 5%
		}
		return v / 100.0
	}
	// Parse as bytes
	bytes, err := HumanToBytes(s)
	if err != nil {
		return 0.05
	}
	return float64(bytes)
}

// ParseSize parses a human-readable size string to int64 bytes
func ParseSize(s string) int64 {
	bytes, err := HumanToBytes(s)
	if err != nil {
		return 30 * 1024 // default 30KiB
	}
	return bytes
}

// ParseBitrate parses a human-readable bitrate string to int64 bits per second
func ParseBitrate(s string) int64 {
	bits, err := HumanToBits(s)
	if err != nil {
		return 128 * 1000 // default 128kbps
	}
	return bits
}
