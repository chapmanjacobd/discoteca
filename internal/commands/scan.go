package commands

import (
	"context"
	"fmt"
	"io/fs"
	"path/filepath"

	"github.com/chapmanjacobd/discoteca/internal/formatutil"
	"github.com/chapmanjacobd/discoteca/internal/logging"
	"github.com/chapmanjacobd/discoteca/internal/metadata"
	"github.com/chapmanjacobd/discoteca/internal/models"
	"github.com/chapmanjacobd/discoteca/internal/query"
)

// ScanMediaPaths walks every root and extracts metadata for each file. Roots may
// be files or directories. Duplicate paths (for example a file passed both
// directly and through its parent directory) are only included once.
func ScanMediaPaths(
	ctx context.Context,
	roots []string,
	opts metadata.ExtractOptions,
) ([]models.MediaWithDB, error) {
	if len(roots) == 0 {
		return nil, nil
	}

	var result []models.MediaWithDB
	seen := make(map[string]bool)

	for _, root := range roots {
		root = filepath.Clean(root)
		if err := filepath.WalkDir(root, func(path string, d fs.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			if d.IsDir() {
				return nil
			}
			if seen[path] {
				return nil
			}

			meta, err := metadata.Extract(ctx, path, opts)
			if err != nil {
				logging.Log.Warn("Failed to extract metadata; skipping file", "path", path, "error", err)
				return nil
			}
			seen[path] = true
			result = append(result, mediaFromMetadata(meta))
			return nil
		}); err != nil {
			return nil, fmt.Errorf("failed to scan %q: %w", root, err)
		}
	}

	return result, nil
}

func mediaFromMetadata(meta *metadata.MediaMetadata) models.MediaWithDB {
	m := meta.Media
	return models.MediaWithDB{
		Media: models.Media{
			Path:            m.Path,
			PathTokenized:   models.NullStringPtr(m.PathTokenized),
			Title:           models.NullStringPtr(m.Title),
			Duration:        models.NullInt64Ptr(m.Duration),
			Size:            models.NullInt64Ptr(m.Size),
			TimeCreated:     models.NullInt64Ptr(m.TimeCreated),
			TimeModified:    models.NullInt64Ptr(m.TimeModified),
			MediaType:       models.NullStringPtr(m.MediaType),
			Width:           models.NullInt64Ptr(m.Width),
			Height:          models.NullInt64Ptr(m.Height),
			Fps:             models.NullFloat64Ptr(m.Fps),
			VideoCodecs:     models.NullStringPtr(m.VideoCodecs),
			AudioCodecs:     models.NullStringPtr(m.AudioCodecs),
			SubtitleCodecs:  models.NullStringPtr(m.SubtitleCodecs),
			VideoCount:      models.NullInt64Ptr(m.VideoCount),
			AudioCount:      models.NullInt64Ptr(m.AudioCount),
			SubtitleCount:   models.NullInt64Ptr(m.SubtitleCount),
			Album:           models.NullStringPtr(m.Album),
			Artist:          models.NullStringPtr(m.Artist),
			Genre:           models.NullStringPtr(m.Genre),
			Categories:      models.NullStringPtr(m.Categories),
			Description:     models.NullStringPtr(m.Description),
			Language:        models.NullStringPtr(m.Language),
			TimeDownloaded:  models.NullInt64Ptr(m.TimeDownloaded),
			Score:           models.NullFloat64Ptr(m.Score),
			Fasthash:        models.NullStringPtr(m.Fasthash),
			Sha256:          models.NullStringPtr(m.Sha256),
			IsDeduped:       models.NullInt64Ptr(m.IsDeduped),
			ContainerFormat: meta.ContainerFormat,
		},
	}
}

// filterScannedMedia applies the filters that can be evaluated from filesystem
// metadata. Filters that require database columns are handled by the caller.
func filterScannedMedia(media []models.MediaWithDB, flags models.GlobalFlags) []models.MediaWithDB {
	media = query.FilterMedia(media, flags)

	filtered := media[:0]
	for _, m := range media {
		if m.MediaType != nil {
			mediaType := *m.MediaType
			if flags.VideoOnly && mediaType != "video" {
				continue
			}
			if flags.AudioOnly && mediaType != "audio" && mediaType != "audiobook" {
				continue
			}
			if flags.ImageOnly && mediaType != "image" {
				continue
			}
			if flags.TextOnly && mediaType != "text" {
				continue
			}
		} else if flags.VideoOnly || flags.AudioOnly || flags.ImageOnly || flags.TextOnly {
			continue
		}

		if flags.Portrait && (m.Width == nil || m.Height == nil || *m.Width >= *m.Height) {
			continue
		}
		if !matchesScannedTimeFilters(m, flags) {
			continue
		}

		filtered = append(filtered, m)
	}
	return filtered
}

func matchesScannedTimeFilters(m models.MediaWithDB, flags models.GlobalFlags) bool {
	after := func(raw string, value *int64) bool {
		if raw == "" {
			return true
		}
		ts := formatutil.ParseDateOrRelative(raw)
		return ts == 0 || (value != nil && *value >= ts)
	}
	before := func(raw string, value *int64) bool {
		if raw == "" {
			return true
		}
		ts := formatutil.ParseDateOrRelative(raw)
		return ts == 0 || (value != nil && *value <= ts)
	}

	return after(flags.CreatedAfter, m.TimeCreated) &&
		before(flags.CreatedBefore, m.TimeCreated) &&
		after(flags.ModifiedAfter, m.TimeModified) &&
		before(flags.ModifiedBefore, m.TimeModified) &&
		after(flags.DownloadedAfter, m.TimeDownloaded) &&
		before(flags.DownloadedBefore, m.TimeDownloaded)
}

// scanOptionsForFlags derives metadata extraction options from the shared flags.
func scanOptionsForFlags(flags models.GlobalFlags) metadata.ExtractOptions {
	return metadata.ExtractOptions{
		ScanSubtitles: flags.ScanSubtitles,
		ProbeImages:   flags.ProbeImages,
	}
}

// loadMedia gathers rows from databases and metadata from scan paths, then
// applies in-memory filters to both sources.
func loadMedia(
	ctx context.Context,
	databases, scanPaths []string,
	flags models.GlobalFlags,
) ([]models.MediaWithDB, error) {
	var media []models.MediaWithDB

	if len(databases) > 0 {
		dbMedia, err := query.MediaQuery(ctx, databases, flags)
		if err != nil {
			return nil, err
		}
		media = append(media, dbMedia...)
	}

	scanned, err := ScanMediaPaths(ctx, scanPaths, scanOptionsForFlags(flags))
	if err != nil {
		return nil, err
	}
	media = append(media, filterScannedMedia(scanned, flags)...)

	return media, nil
}
