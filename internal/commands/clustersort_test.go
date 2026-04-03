package commands

import (
	"os"
	"testing"

	"github.com/chapmanjacobd/discoteca/internal/models"
)

func TestClusterSortCmd_Run(t *testing.T) {
	input := `path/to/video1.mp4
path/to/video2.mp4
some/other/file.mp3
path/to/video3.mp4
`

	t.Run("DefaultStdin", func(t *testing.T) {
		// Mocking stdin is hard, let's use a file instead for the basic test
		tmpFile, err := os.CreateTemp(t.TempDir(), "clustersort-input-*.txt")
		if err != nil {
			t.Fatal(err)
		}
		defer os.Remove(tmpFile.Name())
		tmpFile.WriteString(input)
		tmpFile.Close()

		cmd := &ClusterSortCmd{
			InputPath: tmpFile.Name(),
		}
		// Since ClusterSortCmd writes directly to os.Stdout in some cases,
		// and we can't easily override it without changing the struct to take a writer,
		// let's just make sure it runs without error for now.
		// Actually, it uses a writer if OutputPath is set.

		outputFile, err := os.CreateTemp(t.TempDir(), "clustersort-output-*.txt")
		if err != nil {
			t.Fatal(err)
		}
		defer os.Remove(outputFile.Name())
		outputFile.Close()

		cmd.OutputPath = outputFile.Name()

		if err := cmd.Run(nil); err != nil {
			t.Fatalf("ClusterSortCmd failed: %v", err)
		}

		content, _ := os.ReadFile(outputFile.Name())
		if len(content) == 0 {
			t.Errorf("Expected output, got nothing")
		}
	})

	t.Run("PrintGroups", func(t *testing.T) {
		tmpFile, _ := os.CreateTemp(t.TempDir(), "clustersort-input-*.txt")
		defer os.Remove(tmpFile.Name())
		tmpFile.WriteString(input)
		tmpFile.Close()

		cmd := &ClusterSortCmd{
			InputPath: tmpFile.Name(),
			SimilarityFlags: models.SimilarityFlags{
				PrintGroups: true,
			},
		}
		// This will write to os.Stdout
		if err := cmd.Run(nil); err != nil {
			t.Fatalf("ClusterSortCmd failed: %v", err)
		}
	})
}
