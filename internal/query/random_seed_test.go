package query_test

import (
	"fmt"
	"testing"

	"github.com/chapmanjacobd/discoteca/internal/models"
	"github.com/chapmanjacobd/discoteca/internal/query"
)

func TestSortMediaRandomSeedIsStable(t *testing.T) {
	makeMedia := func() []models.MediaWithDB {
		items := make([]models.MediaWithDB, 20)
		for i := range items {
			items[i] = models.MediaWithDB{
				Media: models.Media{Path: fmt.Sprintf("/media/%02d.mp3", i)},
			}
		}
		return items
	}
	paths := func(items []models.MediaWithDB) []string {
		out := make([]string, len(items))
		for i, item := range items {
			out[i] = item.Path
		}
		return out
	}

	flags := models.GlobalFlags{
		SortFlags: models.SortFlags{Random: true, RandomSeed: 42},
	}

	first := makeMedia()
	query.SortMedia(first, flags)
	second := makeMedia()
	query.SortMedia(second, flags)

	gotFirst := paths(first)
	gotSecond := paths(second)
	for i := range gotFirst {
		if gotFirst[i] != gotSecond[i] {
			t.Fatalf("random order changed between calls at index %d: %q != %q", i, gotFirst[i], gotSecond[i])
		}
	}
}
