package query_test

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"strings"
	"testing"

	_ "github.com/mattn/go-sqlite3"

	"github.com/chapmanjacobd/discoteca/internal/models"
	"github.com/chapmanjacobd/discoteca/internal/query"
	"github.com/chapmanjacobd/discoteca/internal/testutils"
)

type percentileTestEnv struct {
	dbs    []string
	dbPath string
}

func TestResolvePercentileFlags(t *testing.T) {
	f, _ := os.CreateTemp(t.TempDir(), "percentile-test-*.db")
	dbPath := f.Name()
	f.Close()
	defer os.Remove(dbPath)

	dbConn, _ := sql.Open("sqlite3", dbPath)
	defer dbConn.Close()

	if err := testutils.InitTestDBNoFTS(dbConn); err != nil {
		t.Fatalf("Failed to init test DB: %v", err)
	}

	for i := 1; i <= 100; i++ {
		dbConn.Exec("INSERT INTO media (path, size, duration, categories) VALUES (?, ?, ?, ?)",
			fmt.Sprintf("/dir%d/file%d.mp4", (i-1)/10, i), i*1000, i*10, fmt.Sprintf(";dir%d;", (i-1)/10))
	}

	env := &percentileTestEnv{
		dbs:    []string{dbPath},
		dbPath: dbPath,
	}
	ctx := context.Background()

	t.Run("Size Percentile", func(t *testing.T) { env.testSizePercentile(ctx, t) })
	t.Run("Duration Percentile", func(t *testing.T) { env.testDurationPercentile(ctx, t) })
	t.Run("Episodes Percentile", func(t *testing.T) { env.testEpisodesPercentile(ctx, t) })
	t.Run("Specials Button (Absolute Count)", func(t *testing.T) { env.testSpecialsAbsoluteCount(ctx, t) })
	t.Run("Stability Test (Global vs Dynamic)", func(t *testing.T) { env.testStabilityGlobalVsDynamic(ctx, t) })
}

func (e *percentileTestEnv) testSizePercentile(ctx context.Context, t *testing.T) {
	flags := models.GlobalFlags{
		FilterFlags: models.FilterFlags{Size: []string{"p10-50"}},
	}
	resolved, err := query.ResolvePercentileFlags(ctx, e.dbs, flags)
	if err != nil {
		t.Fatalf("query.ResolvePercentileFlags failed: %v", err)
	}

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

	results, _ := query.MediaQuery(ctx, e.dbs, flags)
	if len(results) == 0 {
		t.Error("Expected results for percentile query")
	}
	for _, r := range results {
		if *r.Size < 10000 || *r.Size > 51000 {
			t.Errorf("Result size %d out of expected range", *r.Size)
		}
	}
}

func (e *percentileTestEnv) testDurationPercentile(ctx context.Context, t *testing.T) {
	flags := models.GlobalFlags{
		FilterFlags: models.FilterFlags{Duration: []string{"p20-30"}},
	}
	resolved, err := query.ResolvePercentileFlags(ctx, e.dbs, flags)
	if err != nil {
		t.Fatalf("query.ResolvePercentileFlags failed: %v", err)
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
}

func (e *percentileTestEnv) testEpisodesPercentile(ctx context.Context, t *testing.T) {
	flags := models.GlobalFlags{
		AggregateFlags: models.AggregateFlags{FileCounts: "p0-50"},
	}
	resolved, err := query.ResolvePercentileFlags(ctx, e.dbs, flags)
	if err != nil {
		t.Fatalf("query.ResolvePercentileFlags failed: %v", err)
	}

	if !strings.Contains(resolved.FileCounts, "10") {
		t.Errorf("Expected count 10 in resolved FileCounts, got %s", resolved.FileCounts)
	}

	results, _ := query.MediaQuery(ctx, e.dbs, flags)
	if len(results) != 100 {
		t.Errorf("Expected 100 results (all match count 10), got %d", len(results))
	}
}

func (e *percentileTestEnv) testSpecialsAbsoluteCount(ctx context.Context, t *testing.T) {
	flags := models.GlobalFlags{
		AggregateFlags: models.AggregateFlags{FileCounts: "1"},
	}
	resolved, err := query.ResolvePercentileFlags(ctx, e.dbs, flags)
	if err != nil {
		t.Fatalf("query.ResolvePercentileFlags failed: %v", err)
	}

	if resolved.FileCounts != "1" {
		t.Errorf("Expected Specials absolute count '1', got %s", resolved.FileCounts)
	}
}

func (e *percentileTestEnv) testStabilityGlobalVsDynamic(ctx context.Context, t *testing.T) {
	flags := models.GlobalFlags{
		MediaFilterFlags: models.MediaFilterFlags{Category: []string{"dir0"}},
		FilterFlags:      models.FilterFlags{Size: []string{"p0-100"}},
	}

	resolved, err := query.ResolvePercentileFlags(ctx, e.dbs, flags)
	if err != nil {
		t.Fatalf("query.ResolvePercentileFlags failed: %v", err)
	}

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

	flags.FilterFlags.Size = []string{"p10-50"}
	resolved, _ = query.ResolvePercentileFlags(ctx, e.dbs, flags)

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

	flags.MediaFilterFlags.Category = []string{"dir9"}
	resolved, _ = query.ResolvePercentileFlags(ctx, e.dbs, flags)

	foundP10Dir9 := false
	for _, s := range resolved.Size {
		if s == "+91000" {
			foundP10Dir9 = true
		}
	}
	if !foundP10Dir9 {
		t.Errorf("Expected p10-50 for dir9 to resolve to +91000, got %v", resolved.Size)
	}
}
