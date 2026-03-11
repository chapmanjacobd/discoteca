package query

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/chapmanjacobd/discoteca/internal/models"
	"github.com/chapmanjacobd/discoteca/internal/testutils"
	_ "github.com/mattn/go-sqlite3"
)

func TestResolvePercentileFlags(t *testing.T) {
	// Create a temporary database
	f, _ := os.CreateTemp("", "percentile-test-*.db")
	dbPath := f.Name()
	f.Close()
	defer os.Remove(dbPath)

	dbConn, _ := sql.Open("sqlite3", dbPath)
	defer dbConn.Close()

	if err := testutils.InitTestDBNoFTS(dbConn); err != nil {
		t.Fatalf("Failed to init test DB: %v", err)
	}

	// Insert 100 items with increasing size and duration
	for i := 1; i <= 100; i++ {
		dbConn.Exec("INSERT INTO media (path, size, duration, categories) VALUES (?, ?, ?, ?)",
			fmt.Sprintf("/dir%d/file%d.mp4", (i-1)/10, i), i*1000, i*10, fmt.Sprintf(";dir%d;", (i-1)/10))
	}

	ctx := context.Background()
	dbs := []string{dbPath}

	t.Run("Size Percentile", func(t *testing.T) {
		flags := models.GlobalFlags{
			FilterFlags: models.FilterFlags{Size: []string{"p10-50"}},
		}
		resolved, err := ResolvePercentileFlags(ctx, dbs, flags)
		if err != nil {
			t.Fatalf("ResolvePercentileFlags failed: %v", err)
		}

		// 10th percentile of 1000, 2000, ..., 100000 is approx 10000
		// 50th percentile is approx 50000
		// Our utils.Percentile uses index-based interpolation
		// index = 10/100 * 99 = 9.9 -> sorted[9]*0.1 + sorted[10]*0.9 = 10000*0.1 + 11000*0.9 = 10900
		// index = 50/100 * 99 = 49.5 -> sorted[49]*0.5 + sorted[50]*0.5 = 50000*0.5 + 51000*0.5 = 50500

		foundMin := false
		foundMax := false
		for _, s := range resolved.Size {
			if strings.HasPrefix(s, "+") {
				foundMin = true
			}
			if strings.HasPrefix(s, "-") {
				foundMax = true
			}
		}
		if !foundMin || !foundMax {
			t.Errorf("Expected min/max range in resolved flags, got %v", resolved.Size)
		}

		results, _ := MediaQuery(ctx, dbs, flags)
		if len(results) == 0 {
			t.Error("Expected results for percentile query")
		}
		for _, r := range results {
			if *r.Size < 10000 || *r.Size > 51000 {
				t.Errorf("Result size %d out of expected range", *r.Size)
			}
		}
	})

	t.Run("Duration Percentile", func(t *testing.T) {
		flags := models.GlobalFlags{
			FilterFlags: models.FilterFlags{Duration: []string{"p20-30"}},
		}
		resolved, err := ResolvePercentileFlags(ctx, dbs, flags)
		if err != nil {
			t.Fatalf("ResolvePercentileFlags failed: %v", err)
		}

		foundMin := false
		for _, d := range resolved.Duration {
			if strings.HasPrefix(d, "+") {
				foundMin = true
			}
		}
		if !foundMin {
			t.Errorf("Expected min range in resolved flags, got %v", resolved.Duration)
		}
	})

	t.Run("Episodes Percentile", func(t *testing.T) {
		// We have 10 directories with 10 files each
		flags := models.GlobalFlags{
			AggregateFlags: models.AggregateFlags{FileCounts: "p0-50"},
		}
		resolved, err := ResolvePercentileFlags(ctx, dbs, flags)
		if err != nil {
			t.Fatalf("ResolvePercentileFlags failed: %v", err)
		}

		// All directories have count 10, so 0-50th percentile is still 10
		if !strings.Contains(resolved.FileCounts, "10") {
			t.Errorf("Expected count 10 in resolved FileCounts, got %s", resolved.FileCounts)
		}

		results, _ := MediaQuery(ctx, dbs, flags)
		if len(results) != 100 {
			t.Errorf("Expected 100 results (all match count 10), got %d", len(results))
		}
	})

	t.Run("Specials Button (Absolute Count)", func(t *testing.T) {
		flags := models.GlobalFlags{
			AggregateFlags: models.AggregateFlags{FileCounts: "1"},
		}
		// ResolvePercentileFlags should not change absolute values
		resolved, err := ResolvePercentileFlags(ctx, dbs, flags)
		if err != nil {
			t.Fatalf("ResolvePercentileFlags failed: %v", err)
		}

		if resolved.FileCounts != "1" {
			t.Errorf("Expected Specials absolute count '1', got %s", resolved.FileCounts)
		}
	})
	t.Run("Stability Test (Global vs Dynamic)", func(t *testing.T) {
		// Category filter that would change the distribution
		// dir0 has files with sizes 1000, 2000, ..., 10000
		flags := models.GlobalFlags{
			MediaFilterFlags: models.MediaFilterFlags{Category: []string{"dir0"}},
			FilterFlags:      models.FilterFlags{Size: []string{"p0-100"}},
		}

		// Resolved Size (p0-100) should now be resolved to the filtered set (dir0)
		resolved, err := ResolvePercentileFlags(ctx, dbs, flags)
		if err != nil {
			t.Fatalf("ResolvePercentileFlags failed: %v", err)
		}

		// Verify it resolved to something (absolute values start with + or -)
		hasAbsolute := false
		for _, s := range resolved.Size {
			if strings.HasPrefix(s, "+") || strings.HasPrefix(s, "-") {
				hasAbsolute = true
				break
			}
		}
		if !hasAbsolute {
			t.Errorf("Expected p0-100 to be resolved to absolute values, got %v", resolved.Size)
		}

		// dir0 sizes: 1000, 2000, ..., 10000.
		// p10 of dir0 should be 1000.
		// p50 of dir0 should be 5000.
		flags.FilterFlags.Size = []string{"p10-50"}
		resolved, _ = ResolvePercentileFlags(ctx, dbs, flags)

		foundP10 := false
		foundP50 := false
		for _, s := range resolved.Size {
			if s == "+1000" {
				foundP10 = true
			}
			if s == "-5000" {
				foundP50 = true
			}
		}

		if !foundP10 || !foundP50 {
			t.Errorf("Expected p10-50 for dir0 to resolve to +1000 and -5000, got %v", resolved.Size)
		}

		// Now change category to dir9 (sizes 91000 to 100000)
		flags.MediaFilterFlags.Category = []string{"dir9"}
		resolved, _ = ResolvePercentileFlags(ctx, dbs, flags)

		foundP10Dir9 := false
		for _, s := range resolved.Size {
			if s == "+91000" {
				foundP10Dir9 = true
			}
		}
		if !foundP10Dir9 {
			t.Errorf("Expected p10-50 for dir9 to resolve to +91000, got %v", resolved.Size)
		}
	})
}
