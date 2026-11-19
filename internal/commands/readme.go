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
	models.DatabaseFlags    `embed:""`
}

func (c *ReadmeCmd) Run(ctx *kong.Context) error {
	var sb strings.Builder

	writeIntro(&sb)
	writeScreenshots(&sb)
	writeRequirements(&sb)
	writeDependencies(&sb)
	writeBinaries(&sb)
	writeBuildFromSource(&sb)
	writeUsage(&sb)
	writeCommandOverview(&sb, ctx)
	writeCommandReference(&sb, ctx)

	fmt.Print(sb.String())
	return nil
}

func writeIntro(sb *strings.Builder) {
	sb.WriteString("# discoteca\n\n")
	sb.WriteString("Golang implementation of xklb/library\n\n")
	sb.WriteString("## Quick Start\n\n")
	sb.WriteString("```bash\n")
	sb.WriteString("go install -tags fts5 github.com/chapmanjacobd/discoteca/cmd/disco@latest\n")
	sb.WriteString("\n")
	sb.WriteString("disco database add library.db ./audio\n")
	sb.WriteString("disco database add library.db ./video --scan-subtitles\n")
	sb.WriteString("disco database import calibre library.db --calibre-web-dir ~/calibre-web\n")
	sb.WriteString("disco serve library.db\n")
	sb.WriteString("```\n\n")
	sb.WriteString("## Downloads\n\n")
	sb.WriteString(
		"Paste an http(s) URL into the search bar and click the download button. Downloads use `lb`, or `lb-wrapper` when `lb` is unavailable. Set `--download-command` to choose a command and `--download-dir` to change the output directory. Downloaded files are imported into the selected database automatically. Downloads are disabled with `--read-only`.\n\n",
	)
}

func writeScreenshots(sb *strings.Builder) {
	sb.WriteString("## Screenshots\n\n")
	screenshots := []struct{ title, path string }{
		{"Grid View", "docs/screenshots/search-results.png"},
		{"Group View", "docs/screenshots/group-view.png"},
		{"Details View", "docs/screenshots/home-details-view.png"},
		{"Disk Usage View", "docs/screenshots/disk-usage-view.png"},
		{"Video Player", "docs/screenshots/video-player.png"},
		{"Audio Player", "docs/screenshots/audio-player.png"},
		{"EPUB Viewer", "docs/screenshots/epub-viewer.png"},
		{"Settings Modal", "docs/screenshots/settings-modal.png"},
	}
	for _, s := range screenshots {
		fmt.Fprintf(sb, "### %s\n\n", s.title)
		fmt.Fprintf(sb, "![%s](%s)\n\n", s.title, s.path)
	}
}

func writeRequirements(sb *strings.Builder) {
	sb.WriteString("## Requirements\n\n")
	sb.WriteString("- Go 1.26 or newer to build from source\n")
	sb.WriteString("- SQLite **3.37.0 or newer** (the schema uses STRICT tables)\n")
	sb.WriteString("- A SQLite build with FTS5 enabled (`-tags fts5`)\n\n")
}

func writeDependencies(sb *strings.Builder) {
	sb.WriteString("## Optional dependencies\n\n")
	sb.WriteString("### Core Media Features\n\n")
	sb.WriteString(
		"- `ffmpeg` - Media transcoding, streaming, duration detection, subtitle extraction, image conversion\n",
	)
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
	sb.WriteString(
		"- `whisper` (openai-whisper) - High-accuracy speech-to-text (optional, requires Python, GPU recommended)\n\n",
	)
	sb.WriteString("### Archive & Legacy Formats\n\n")
	sb.WriteString("- `catdoc` - Old Microsoft Office formats (.doc, .xls, .ppt)\n")
	sb.WriteString("- `xls2csv` - Excel .xls spreadsheet extraction\n")
	sb.WriteString("- `unar` or`p7zip-full` - 7-Zip archive listing (`7z`)\n")
	sb.WriteString("- `unar` or`unrar` - RAR archive listing and CBR extraction\n")
	sb.WriteString("- `chmextractor` or `libmspack-tools` - CHM help file extraction\n\n")
	sb.WriteString("### Accessibility & TTS\n\n")
	sb.WriteString("- `espeak-ng` - Text-to-speech generation\n\n")
	sb.WriteString("### Web & Proxy\n\n")
	sb.WriteString("- `kiwix-serve` or `zimantic` - ZIM file serving and search (Wikipedia offline)\n\n")
	sb.WriteString("See [INSTALL.md](docs/INSTALL.md) for installation instructions on your platform.\n")
	sb.WriteString("See [DEPLOYMENT.md](docs/DEPLOYMENT.md) for systemd, nginx, and URL-prefix setup.\n\n")
}

func writeBinaries(sb *strings.Builder) {
	sb.WriteString("## Pre-built Binaries\n\n")
	sb.WriteString("Download from [GitHub Releases](https://github.com/chapmanjacobd/discoteca/releases) for:\n")
	sb.WriteString("- **Linux**: amd64, arm64\n")
	sb.WriteString("- **Windows**: amd64\n")
	sb.WriteString("- **macOS**: amd64, arm64 (Apple Silicon)\n\n")
}

func writeBuildFromSource(sb *strings.Builder) {
	sb.WriteString("## Build from Source\n\n")
	sb.WriteString("```bash\n")
	sb.WriteString("git clone https://github.com/chapmanjacobd/discoteca.git\n")
	sb.WriteString("cd discoteca\n")
	sb.WriteString("go build -tags \"fts5\" -o disco ./cmd/disco\n")
	sb.WriteString("```\n\n")
}

func writeUsage(sb *strings.Builder) {
	sb.WriteString("## Usage\n\n")
	sb.WriteString(
		"Commands are organized by workflow. Nested paths are canonical; the shorter flat command name remains available as an alias.\n",
	)
	sb.WriteString("For example, `disco database add` and `disco add` run the same command.\n\n")
}

var subcommandExamples = map[string][]string{
	"add": {
		"disco database add my_videos.db ~/Videos",
		"disco database add --video-only my_videos.db /mnt/media",
	},
	"calibre-import": {
		"disco database import calibre my_videos.db --library ~/Calibre",
		"disco database import calibre my_videos.db --calibre-web-dir ~/calibre-web --one-format",
	},
	"check": {
		"disco database check my_videos.db ~/Videos",
	},
	"print": {
		"disco print my_videos.db",
		"disco print my_videos.db -u size --reverse",
		"disco print my_videos.db --big-dirs -u count",
	},
	"search": {
		"disco search my_videos.db --search matrix",
		"disco search my_videos.db --search cyberpunk --video-only",
	},
	"watch": {
		"disco play video my_videos.db",
		"disco play video my_videos.db -r --limit 10",
		"disco play video my_videos.db --size '>1GB'",
	},
	"listen": {
		"disco play audio my_music.db",
		"disco play audio my_music.db --random",
	},
	"serve": {
		"disco serve my_videos.db my_music.db",
		"disco serve --read-only my_videos.db",
	},
	"dedupe": {
		"disco database dedupe my_videos.db",
	},
	"big-dirs": {
		"disco folder large my_videos.db",
	},
	"files-info": {
		"disco filesystem list my_videos.db",
	},
	"disk-usage": {
		"disco filesystem usage my_videos.db",
		"disco filesystem usage my_videos.db --depth 2",
	},
	"history": {
		"disco database history my_videos.db",
		"disco database history my_videos.db --in-progress",
	},
	"optimize": {
		"disco database optimize my_videos.db",
	},
}

// SubcommandExamples returns the documented examples keyed by command name.
// It exists so tests can verify that every published example parses.
func SubcommandExamples() map[string][]string {
	return subcommandExamples
}

func readmeNodes(ctx *kong.Context) map[string]*kong.Node {
	nodes := make(map[string]*kong.Node, len(ctx.Model.Children))
	for _, node := range ctx.Model.Children {
		if node.Hidden {
			continue
		}
		nodes[node.Name] = node
	}
	return nodes
}

func writeCommandOverview(sb *strings.Builder, ctx *kong.Context) {
	sb.WriteString("## Commands\n\n")
	nodes := readmeNodes(ctx)
	for _, category := range CommandCatalog() {
		fmt.Fprintf(sb, "### %s\n\n", category.Name)
		for _, spec := range category.Commands {
			node := nodes[spec.Name]
			if node == nil {
				continue
			}
			fmt.Fprintf(sb, "- `disco %s` - %s\n", CommandDisplayPath(spec), node.Help)
		}
		sb.WriteString("\n")
	}
}

func writeCommandReference(sb *strings.Builder, ctx *kong.Context) {
	sb.WriteString("## Command reference\n\n")
	nodes := readmeNodes(ctx)
	for _, category := range CommandCatalog() {
		for _, spec := range category.Commands {
			node := nodes[spec.Name]
			if node == nil {
				continue
			}
			writeSubcommand(sb, node, spec)
		}
	}
}

func writeSubcommand(sb *strings.Builder, node *kong.Node, spec CommandSpec) {
	path := CommandDisplayPath(spec)
	fmt.Fprintf(sb, "### %s\n\n", path)
	fmt.Fprintf(sb, "%s\n\n", node.Help)

	if spec.Path != "" && spec.Path != spec.Name {
		fmt.Fprintf(sb, "Flat alias: `disco %s`\n\n", spec.Name)
	}

	if ex, ok := SubcommandExamples()[node.Name]; ok {
		writeExamples(sb, ex)
	}

	writeSubcommandOptions(sb, node, path)
}

func writeExamples(sb *strings.Builder, examples []string) {
	sb.WriteString("Examples:\n\n```bash\n")
	for _, line := range examples {
		fmt.Fprintf(sb, "$ %s\n", line)
	}
	sb.WriteString("```\n\n")
}

func writeSubcommandOptions(sb *strings.Builder, node *kong.Node, path string) {
	sb.WriteString("<details><summary>All Options</summary>\n\n")
	sb.WriteString("```bash\n")
	fmt.Fprintf(sb, "$ disco %s --help\n", path)

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
			fmt.Fprintf(sb, "  %s--%s\n", short, flag.Name)
			fmt.Fprintf(sb, "        %s\n", flag.Help)
		}
	}

	sb.WriteString("```\n\n")
	sb.WriteString("</details>\n\n")
}
