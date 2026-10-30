package db_test

import (
	"strings"
	"testing"

	"github.com/chapmanjacobd/discoteca/internal/db"
	"github.com/chapmanjacobd/discoteca/internal/media"
)

// TestSubtitlePathExclusionMatchesExtensions prevents the SQL predicate used
// by aggregate queries from drifting away from the subtitle list used by the
// media grid filter.
func TestSubtitlePathExclusionMatchesExtensions(t *testing.T) {
	for _, ext := range media.SubtitleExtensions {
		if !strings.Contains(db.SubtitlePathExclusion, "'%"+ext+"'") {
			t.Errorf("SubtitlePathExclusion is missing %q", ext)
		}
	}
}
