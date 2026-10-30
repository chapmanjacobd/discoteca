package aggregate_test

import (
	"testing"

	"github.com/chapmanjacobd/discoteca/internal/aggregate"
	"github.com/chapmanjacobd/discoteca/internal/models"
)

func TestFilterNearDuplicates_UsesBasenames(t *testing.T) {
	groups := []models.FolderStats{{
		Path: "/library",
		Files: []models.MediaWithDB{
			{Media: models.Media{Path: "/one/movie.mp4"}},
			{Media: models.Media{Path: "/two/movie.mp4"}},
		},
		Count: 2,
	}}

	got := aggregate.FilterNearDuplicates(groups)
	if len(got) != 1 {
		t.Fatalf("expected matching basenames to form one group, got %d groups", len(got))
	}
	if len(got[0].Files) != 2 {
		t.Errorf("expected both files in the group, got %d", len(got[0].Files))
	}
}

func TestClusterFoldersByName_NameFullPath(t *testing.T) {
	folders := []models.FolderStats{
		{Path: "/alpha/collection", Count: 1},
		{Path: "/z/collection", Count: 1},
	}

	basenameMode := aggregate.ClusterFoldersByName(
		models.GlobalFlags{SimilarityFlags: models.SimilarityFlags{}},
		folders,
	)
	if len(basenameMode) != 1 {
		t.Errorf("basename mode expected 1 group, got %d", len(basenameMode))
	}

	fullPathMode := aggregate.ClusterFoldersByName(
		models.GlobalFlags{SimilarityFlags: models.SimilarityFlags{NameFullPath: true}},
		folders,
	)
	if len(fullPathMode) != 2 {
		t.Errorf("full-path mode expected 2 groups, got %d", len(fullPathMode))
	}
}
