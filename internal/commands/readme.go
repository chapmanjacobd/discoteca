package commands

import (
	"fmt"
	"strings"

	"github.com/alecthomas/kong"
	"github.com/chapmanjacobd/discoteca/internal/models"
)

type ReadmeCmd struct {
	models.CoreFlags        `embed:""`
	models.QueryFlags       `embed:""`
	models.PathFilterFlags  `embed:""`
	models.FilterFlags      `embed:""`
	models.MediaFilterFlags `embed:""`
	models.TimeFilterFlags  `embed:""`
	models.DeletedFlags     `embed:""`
	models.SortFlags        `embed:""`
	models.DisplayFlags     `embed:""`
	models.AggregateFlags   `embed:""`
	models.TextFlags        `embed:""`
	models.SimilarityFlags  `embed:""`
	models.DedupeFlags      `embed:""`
	models.FTSFlags         `embed:""`
	models.PlaybackFlags    `embed:""`
	models.MpvActionFlags   `embed:""`
	models.PostActionFlags  `embed:""`
	models.HashingFlags     `embed:""`
	models.MergeFlags       `embed:""`
	models.DatabaseFlags    `embed:""`
}

func (c *ReadmeCmd) Run(ctx *kong.Context) error {
	var sb strings.Builder

	sb.WriteString("# discoteca\n\n")
	sb.WriteString("Golang implementation of xklb/library\n\n")
	sb.WriteString("## Quick Install\n\n")
	sb.WriteString("```bash\n")
	sb.WriteString("go install -tags fts5 github.com/chapmanjacobd/discoteca/cmd/disco@latest\n")
	sb.WriteString("```\n\n")

	sb.WriteString("## Screenshots\n\n")
	sb.WriteString("### Grid View\n\n")
	sb.WriteString("![Search Results](docs/screenshots/search-results.png)\n\n")
	sb.WriteString("### Details View\n\n")
	sb.WriteString("![Details View](docs/screenshots/home-details-view.png)\n\n")
	sb.WriteString("### Disk Usage View\n\n")
	sb.WriteString("![Disk Usage View](docs/screenshots/disk-usage-view.png)\n\n")
	sb.WriteString("### Video Player\n\n")
	sb.WriteString("![Video Player](docs/screenshots/video-player.png)\n\n")
	sb.WriteString("### Audio Player\n\n")
	sb.WriteString("![Audio Player](docs/screenshots/audio-player.png)\n\n")
	sb.WriteString("### EPUB Viewer\n\n")
	sb.WriteString("![EPUB Viewer](docs/screenshots/epub-viewer.png)\n\n")
	sb.WriteString("### Settings Modal\n\n")
	sb.WriteString("![Settings Modal](docs/screenshots/settings-modal.png)\n\n")
	sb.WriteString("### Group View\n\n")
	sb.WriteString("![Group View](docs/screenshots/group-view.png)\n\n")

	sb.WriteString("## Optional dependencies\n\n")
	sb.WriteString("### Core Media Features\n\n")
	sb.WriteString("- `ffmpeg` - Media transcoding, streaming, duration detection, subtitle extraction, image conversion\n")
	sb.WriteString("- `mpv` - Playback control, keyboard shortcuts, playlist management\n\n")
	sb.WriteString("### Document & Ebook Support\n\n")
	sb.WriteString("- `calibre` - Ebook conversion (mobi, azw, fb2, djvu, cbz, cbr, old Office formats)\n")
	sb.WriteString("- `poppler-utils` - PDF text extraction (`pdftotext`) and thumbnails (`pdftoppm`)\n")
	sb.WriteString("- `unrtf` - RTF document text extraction\n")
	sb.WriteString("- `ghostscript` - PostScript text extraction (`ps2ascii`, `pstotext`)\n\n")
	sb.WriteString("### OCR & Image Processing\n\n")
	sb.WriteString("- `tesseract` - OCR text extraction from images (ingest scanning)\n")
	sb.WriteString("- `paddleocr` - Advanced OCR with better accuracy on complex layouts (optional, requires Python)\n")
	sb.WriteString("- `imagemagick` - Image format conversion and manipulation\n\n")
	sb.WriteString("### Speech Recognition\n\n")
	sb.WriteString("- `vosk` + `python3` - Speech-to-text extraction from audio/video (offline, lightweight)\n")
	sb.WriteString("- `whisper` (openai-whisper) - High-accuracy speech-to-text (optional, requires Python, GPU recommended)\n\n")
	sb.WriteString("### Archive & Legacy Formats\n\n")
	sb.WriteString("- `catdoc` - Old Microsoft Office formats (.doc, .xls, .ppt)\n")
	sb.WriteString("- `xls2csv` - Excel .xls spreadsheet extraction\n")
	sb.WriteString("- `unar` or`p7zip-full` - 7-Zip archive listing (`7z`)\n")
	sb.WriteString("- `unar` or`unrar` - RAR archive listing and CBR extraction\n")
	sb.WriteString("- `chmextractor` or `libmspack-tools` - CHM help file extraction\n\n")
	sb.WriteString("### Accessibility & TTS\n\n")
	sb.WriteString("- `espeak-ng` - Text-to-speech generation\n\n")
	sb.WriteString("### Web & Proxy\n\n")
	sb.WriteString("- `kiwix-serve` - ZIM file serving (Wikipedia offline)\n\n")
	sb.WriteString("See [INSTALL.md](docs/INSTALL.md) for installation instructions on your platform.\n\n")

	sb.WriteString("## Pre-built Binaries\n\n")
	sb.WriteString("Download from [GitHub Releases](https://github.com/chapmanjacobd/discoteca/releases) for:\n")
	sb.WriteString("- **Linux**: amd64, arm64\n")
	sb.WriteString("- **Windows**: amd64\n")
	sb.WriteString("- **macOS**: amd64, arm64 (Apple Silicon)\n\n")
	sb.WriteString("## Build from Source\n\n")
	sb.WriteString("```bash\n")
	sb.WriteString("git clone https://github.com/chapmanjacobd/discoteca.git\n")
	sb.WriteString("cd discoteca\n")
	sb.WriteString("go build -tags \"fts5\" -o disco ./cmd/disco\n")
	sb.WriteString("```\n\n")

	sb.WriteString("## Usage\n\n")

	examples := map[string][]string{
		"add": {
			"disco add my_videos.db ~/Videos",
			"disco add --video-only my_videos.db /mnt/media",
		},
		"print": {
			"disco print my_videos.db",
			"disco print my_videos.db -u size --reverse",
			"disco print my_videos.db --big-dirs -u count",
		},
		"search": {
			"disco search my_videos.db 'matrix'",
			"disco search my_videos.db 'cyberpunk' --video-only",
		},
		"watch": {
			"disco watch my_videos.db",
			"disco watch my_videos.db -r --limit 10",
			"disco watch my_videos.db --size '>1GB'",
		},
		"listen": {
			"disco listen my_music.db",
			"disco listen my_music.db --random",
		},
		"serve": {
			"disco serve my_videos.db my_music.db",
			"disco serve --readonly my_videos.db",
		},
		"disk-usage": {
			"disco du my_videos.db",
			"disco du my_videos.db --depth 2",
		},
		"history": {
			"disco history my_videos.db",
			"disco history my_videos.db --inprogress",
		},
		"optimize": {
			"disco optimize my_videos.db",
		},
	}

	// Iterate through subcommands
	for _, node := range ctx.Model.Children {
		if node.Hidden {
			continue
		}
		sb.WriteString(fmt.Sprintf("### %s\n\n", node.Name))
		sb.WriteString(fmt.Sprintf("%s\n\n", node.Help))

		if ex, ok := examples[node.Name]; ok {
			sb.WriteString("Examples:\n\n```bash\n")
			for _, line := range ex {
				sb.WriteString(fmt.Sprintf("$ %s\n", line))
			}
			sb.WriteString("```\n\n")
		}

		sb.WriteString("<details><summary>All Options</summary>\n\n")
		sb.WriteString("```bash\n")
		sb.WriteString(fmt.Sprintf("$ disco %s --help\n", node.Name))

		if len(node.Flags) > 0 {
			sb.WriteString("\nFlags:\n")
			for _, flag := range node.Flags {
				if flag.Hidden {
					continue
				}
				short := ""
				if flag.Short != 0 {
					short = fmt.Sprintf("-%c, ", flag.Short)
				}
				sb.WriteString(fmt.Sprintf("  %s--%s\n", short, flag.Name))
				sb.WriteString(fmt.Sprintf("        %s\n", flag.Help))
			}
		}

		sb.WriteString("```\n\n")
		sb.WriteString("</details>\n\n")
	}

	fmt.Print(sb.String())
	return nil
}
