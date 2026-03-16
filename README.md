# discoteca

Golang implementation of xklb/library

## Quick Install

```bash
go install -tags fts5 github.com/chapmanjacobd/discoteca/cmd/disco@latest
```

## Optional dependencies

### Core Media Features

- `ffmpeg` - Media transcoding, streaming, duration detection, subtitle extraction, image conversion
- `mpv` - Playback control, keyboard shortcuts, playlist management

### Document & Ebook Support

- `calibre` - Ebook conversion (mobi, azw, fb2, djvu, cbz, cbr, old Office formats)
- `poppler-utils` - PDF text extraction (`pdftotext`) and thumbnails (`pdftoppm`)
- `unrtf` - RTF document text extraction
- `ghostscript` - PostScript text extraction (`ps2ascii`, `pstotext`)

### OCR & Image Processing

- `tesseract` - OCR text extraction from images (ingest scanning)
- `paddleocr` - Advanced OCR with better accuracy on complex layouts (optional, requires Python)
- `imagemagick` - Image format conversion and manipulation

### Speech Recognition

- `vosk` + `python3` - Speech-to-text extraction from audio/video (offline, lightweight)
- `whisper` (openai-whisper) - High-accuracy speech-to-text (optional, requires Python, GPU recommended)

### Archive & Legacy Formats

- `catdoc` - Old Microsoft Office formats (.doc, .xls, .ppt)
- `xls2csv` - Excel .xls spreadsheet extraction
- `unar` or`p7zip-full` - 7-Zip archive listing (`7z`)
- `unar` or`unrar` - RAR archive listing and CBR extraction
- `chmextractor` or `libmspack-tools` - CHM help file extraction

### Accessibility & TTS

- `espeak-ng` - Text-to-speech generation

### Web & Proxy

- `kiwix-serve` - ZIM file serving (Wikipedia offline)

See [INSTALL.md](docs/INSTALL.md) for installation instructions on your platform.

## Screenshots

### Grid View

![Search Results](docs/screenshots/search-results.png)

### Details View

![Details View](docs/screenshots/home-details-view.png)

### Video Player

![Video Player](docs/screenshots/video-player.png)

### EPUB Viewer

![EPUB Viewer](docs/screenshots/epub-viewer.png)

### Settings Modal

![Settings Modal](docs/screenshots/settings-modal.png)

### Group View

![Group View](docs/screenshots/group-view.png)

## Pre-built Binaries

Download from [GitHub Releases](https://github.com/chapmanjacobd/discoteca/releases) for:
- **Linux**: amd64, arm64
- **Windows**: amd64
- **macOS**: amd64, arm64 (Apple Silicon)

## Build from Source

```bash
git clone https://github.com/chapmanjacobd/discoteca.git
cd discoteca
go build -tags "fts5" -o disco ./cmd/disco
```

## Usage

### add

Add media to database

Examples:

```bash
$ disco add my_videos.db ~/Videos
$ disco add --video-only my_videos.db /mnt/media
```

<details><summary>All Options</summary>

```bash
$ disco add --help

Flags:
  -v, --verbose
        Enable verbose logging
  --simulate
        Dry run; don't actually do anything
  -y, --no-confirm
        Don't ask for confirmation
  -T, --timeout
        Quit after N minutes/seconds
  -s, --include
        Include paths matching pattern
  -E, --exclude
        Exclude paths matching pattern
  --regex
        Filter paths by regex pattern
  --path-contains
        Path must contain all these strings
  --paths
        Exact paths to include
  --search
        Search terms (space-separated for AND, | for OR)
  -S, --size
        Size range (e.g., >100MB, 1GB%10)
  -d, --duration
        Duration range (e.g., >1hour, 30min%10)
  --modified
        Filter by modification time
  --created
        Filter by creation time
  --downloaded
        Filter by download time
  --duration-from-size
        Constrain media to duration of videos which match any size constraints
  --watched
        Filter by watched status (true/false)
  --unfinished
        Has playhead but not finished
  -P, --partial
        Filter by partial playback status
  --play-count-min
        Minimum play count
  --play-count-max
        Maximum play count
  --completed
        Show only completed items
  --in-progress
        Show only items in progress
  --with-captions
        Show only items with captions
  --flexible-search
        Flexible search (fuzzy)
  --exact
        Exact match for search
  -w, --where
        SQL where clause(s)
  --exists
        Filter out non-existent files
  -o, --fetch-siblings
        Fetch siblings of matched files (each, all, if-audiobook)
  --fetch-siblings-max
        Maximum number of siblings to fetch
  --category
        Filter by category
  --genre
        Filter by genre
  --language
        Filter by language
  -e, --ext
        Filter by extensions (e.g., .mp4,.mkv)
  --video-only
        Only video files
  --audio-only
        Only audio files
  --image-only
        Only image files
  --text-only
        Only text/ebook files
  --portrait
        Only portrait orientation files
  --scan-subtitles
        Scan for external subtitles during import
  --online-media-only
        Exclude local media
  --local-media-only
        Exclude online media
  --mime-type
        Filter by mimetype substring (e.g., video, mp4)
  --no-mime-type
        Exclude by mimetype substring
  --no-default-categories
        Disable default categories
  -p, --parallel
        Number of parallel extractors (default: CPU count * 4)
  --extract-text
        Extract full text from documents (PDF, EPUB, TXT, MD) for caption search
  --ocr
        Extract text from images using OCR (tesseract) for caption search
  --ocr-engine
        OCR engine to use
  --speech-recognition
        Extract speech-to-text from audio/video files for caption search
  --speech-recognition-engine
        Speech recognition engine to use
```

</details>

### check

Check for missing files and mark as deleted

<details><summary>All Options</summary>

```bash
$ disco check --help

Flags:
  -v, --verbose
        Enable verbose logging
  --simulate
        Dry run; don't actually do anything
  -y, --no-confirm
        Don't ask for confirmation
  -T, --timeout
        Quit after N minutes/seconds
  -s, --include
        Include paths matching pattern
  -E, --exclude
        Exclude paths matching pattern
  --regex
        Filter paths by regex pattern
  --path-contains
        Path must contain all these strings
  --paths
        Exact paths to include
  --category
        Filter by category
  --genre
        Filter by genre
  --language
        Filter by language
  -e, --ext
        Filter by extensions (e.g., .mp4,.mkv)
  --video-only
        Only video files
  --audio-only
        Only audio files
  --image-only
        Only image files
  --text-only
        Only text/ebook files
  --portrait
        Only portrait orientation files
  --scan-subtitles
        Scan for external subtitles during import
  --online-media-only
        Exclude local media
  --local-media-only
        Exclude online media
  --mime-type
        Filter by mimetype substring (e.g., video, mp4)
  --no-mime-type
        Exclude by mimetype substring
  --no-default-categories
        Disable default categories
  --dry-run
        Don't actually mark files as deleted
```

</details>

### print

Print media information

Examples:

```bash
$ disco print my_videos.db
$ disco print my_videos.db -u size --reverse
$ disco print my_videos.db --big-dirs -u count
```

<details><summary>All Options</summary>

```bash
$ disco print --help

Flags:
  -v, --verbose
        Enable verbose logging
  --simulate
        Dry run; don't actually do anything
  -y, --no-confirm
        Don't ask for confirmation
  -T, --timeout
        Quit after N minutes/seconds
  -q, --query
        Raw SQL query (overrides all query building)
  -L, --limit
        Limit results per database
  -a, --all
        Return all results (no limit)
  --offset
        Skip N results
  -s, --include
        Include paths matching pattern
  -E, --exclude
        Exclude paths matching pattern
  --regex
        Filter paths by regex pattern
  --path-contains
        Path must contain all these strings
  --paths
        Exact paths to include
  --search
        Search terms (space-separated for AND, | for OR)
  -S, --size
        Size range (e.g., >100MB, 1GB%10)
  -d, --duration
        Duration range (e.g., >1hour, 30min%10)
  --modified
        Filter by modification time
  --created
        Filter by creation time
  --downloaded
        Filter by download time
  --duration-from-size
        Constrain media to duration of videos which match any size constraints
  --watched
        Filter by watched status (true/false)
  --unfinished
        Has playhead but not finished
  -P, --partial
        Filter by partial playback status
  --play-count-min
        Minimum play count
  --play-count-max
        Maximum play count
  --completed
        Show only completed items
  --in-progress
        Show only items in progress
  --with-captions
        Show only items with captions
  --flexible-search
        Flexible search (fuzzy)
  --exact
        Exact match for search
  -w, --where
        SQL where clause(s)
  --exists
        Filter out non-existent files
  -o, --fetch-siblings
        Fetch siblings of matched files (each, all, if-audiobook)
  --fetch-siblings-max
        Maximum number of siblings to fetch
  --category
        Filter by category
  --genre
        Filter by genre
  --language
        Filter by language
  -e, --ext
        Filter by extensions (e.g., .mp4,.mkv)
  --video-only
        Only video files
  --audio-only
        Only audio files
  --image-only
        Only image files
  --text-only
        Only text/ebook files
  --portrait
        Only portrait orientation files
  --scan-subtitles
        Scan for external subtitles during import
  --online-media-only
        Exclude local media
  --local-media-only
        Exclude online media
  --mime-type
        Filter by mimetype substring (e.g., video, mp4)
  --no-mime-type
        Exclude by mimetype substring
  --no-default-categories
        Disable default categories
  --created-after
        Created after date (YYYY-MM-DD)
  --created-before
        Created before date (YYYY-MM-DD)
  --modified-after
        Modified after date (YYYY-MM-DD)
  --modified-before
        Modified before date (YYYY-MM-DD)
  --downloaded-after
        Downloaded after date (YYYY-MM-DD)
  --downloaded-before
        Downloaded before date (YYYY-MM-DD)
  --deleted-after
        Deleted after date (YYYY-MM-DD)
  --deleted-before
        Deleted before date (YYYY-MM-DD)
  --played-after
        Last played after date (YYYY-MM-DD)
  --played-before
        Last played before date (YYYY-MM-DD)
  --hide-deleted
        Exclude deleted files from results
  --only-deleted
        Include only deleted files in results
  -u, --sort-by
        Sort by field
  -V, --reverse
        Reverse sort order
  -n, --nat-sort
        Use natural sorting
  -r, --random
        Random order
  -k, --re-rank
        Add key/value pairs re-rank sorting by multiple attributes (COLUMN=WEIGHT)
  -c, --columns
        Columns to display
  -j, --json
        Output results as JSON
  --summarize
        Print aggregate statistics
  -f, --frequency
        Group statistics by time frequency (daily, weekly, monthly, yearly)
  --tui
        Interactive TUI mode
  -B, --big-dirs
        Aggregate by parent directory
  --file-counts
        Filter by number of files in directory (e.g., >5, 10%1)
  --group-by-extensions
        Group by file extensions
  --group-by-mime-types
        Group by mimetypes
  --group-by-size
        Group by size buckets
  --group-by-parent
        Group media by parent directory with counts and totals
  -D, --depth
        Aggregate at specific directory depth
  --min-depth
        Minimum depth for aggregation
  --max-depth
        Maximum depth for aggregation
  --parents
        Include parent directories in aggregation
  --folders-only
        Only show folders
  --files-only
        Only show files
  --folder-sizes
        Filter folders by total size
  --folder-counts
        Filter folders by number of subfolders
  --regex-sort
        Sort by splitting lines and sorting words
  --regexs
        Regex patterns for line splitting
  --word-sorts
        Word sorting strategies
  --line-sorts
        Line sorting strategies
  --compat
        Use natsort compat mode
  --preprocess
        Remove junk common to filenames and URLs
  --stop-words
        List of words to ignore
  --duplicates
        Filter for duplicate words (true/false)
  --unique-only
        Filter for unique words (true/false)
  --fts
        Use FTS5 full-text search
  --fts-table
        FTS table name
  --no-fts
        Disable full-text search, use substring search only
  -R, --related
        Find media related to the first result
```

</details>

### search

Search media using FTS

Examples:

```bash
$ disco search my_videos.db 'matrix'
$ disco search my_videos.db 'cyberpunk' --video-only
```

<details><summary>All Options</summary>

```bash
$ disco search --help

Flags:
  -v, --verbose
        Enable verbose logging
  --simulate
        Dry run; don't actually do anything
  -y, --no-confirm
        Don't ask for confirmation
  -T, --timeout
        Quit after N minutes/seconds
  -q, --query
        Raw SQL query (overrides all query building)
  -L, --limit
        Limit results per database
  -a, --all
        Return all results (no limit)
  --offset
        Skip N results
  -s, --include
        Include paths matching pattern
  -E, --exclude
        Exclude paths matching pattern
  --regex
        Filter paths by regex pattern
  --path-contains
        Path must contain all these strings
  --paths
        Exact paths to include
  --search
        Search terms (space-separated for AND, | for OR)
  -S, --size
        Size range (e.g., >100MB, 1GB%10)
  -d, --duration
        Duration range (e.g., >1hour, 30min%10)
  --modified
        Filter by modification time
  --created
        Filter by creation time
  --downloaded
        Filter by download time
  --duration-from-size
        Constrain media to duration of videos which match any size constraints
  --watched
        Filter by watched status (true/false)
  --unfinished
        Has playhead but not finished
  -P, --partial
        Filter by partial playback status
  --play-count-min
        Minimum play count
  --play-count-max
        Maximum play count
  --completed
        Show only completed items
  --in-progress
        Show only items in progress
  --with-captions
        Show only items with captions
  --flexible-search
        Flexible search (fuzzy)
  --exact
        Exact match for search
  -w, --where
        SQL where clause(s)
  --exists
        Filter out non-existent files
  -o, --fetch-siblings
        Fetch siblings of matched files (each, all, if-audiobook)
  --fetch-siblings-max
        Maximum number of siblings to fetch
  --category
        Filter by category
  --genre
        Filter by genre
  --language
        Filter by language
  -e, --ext
        Filter by extensions (e.g., .mp4,.mkv)
  --video-only
        Only video files
  --audio-only
        Only audio files
  --image-only
        Only image files
  --text-only
        Only text/ebook files
  --portrait
        Only portrait orientation files
  --scan-subtitles
        Scan for external subtitles during import
  --online-media-only
        Exclude local media
  --local-media-only
        Exclude online media
  --mime-type
        Filter by mimetype substring (e.g., video, mp4)
  --no-mime-type
        Exclude by mimetype substring
  --no-default-categories
        Disable default categories
  --created-after
        Created after date (YYYY-MM-DD)
  --created-before
        Created before date (YYYY-MM-DD)
  --modified-after
        Modified after date (YYYY-MM-DD)
  --modified-before
        Modified before date (YYYY-MM-DD)
  --downloaded-after
        Downloaded after date (YYYY-MM-DD)
  --downloaded-before
        Downloaded before date (YYYY-MM-DD)
  --deleted-after
        Deleted after date (YYYY-MM-DD)
  --deleted-before
        Deleted before date (YYYY-MM-DD)
  --played-after
        Last played after date (YYYY-MM-DD)
  --played-before
        Last played before date (YYYY-MM-DD)
  --hide-deleted
        Exclude deleted files from results
  --only-deleted
        Include only deleted files in results
  -u, --sort-by
        Sort by field
  -V, --reverse
        Reverse sort order
  -n, --nat-sort
        Use natural sorting
  -r, --random
        Random order
  -k, --re-rank
        Add key/value pairs re-rank sorting by multiple attributes (COLUMN=WEIGHT)
  -c, --columns
        Columns to display
  -j, --json
        Output results as JSON
  --summarize
        Print aggregate statistics
  -f, --frequency
        Group statistics by time frequency (daily, weekly, monthly, yearly)
  --tui
        Interactive TUI mode
  --fts
        Use FTS5 full-text search
  --fts-table
        FTS table name
  --no-fts
        Disable full-text search, use substring search only
  -R, --related
        Find media related to the first result
```

</details>

### search-captions

Search captions using FTS

<details><summary>All Options</summary>

```bash
$ disco search-captions --help

Flags:
  -v, --verbose
        Enable verbose logging
  --simulate
        Dry run; don't actually do anything
  -y, --no-confirm
        Don't ask for confirmation
  -T, --timeout
        Quit after N minutes/seconds
  -q, --query
        Raw SQL query (overrides all query building)
  -L, --limit
        Limit results per database
  -a, --all
        Return all results (no limit)
  --offset
        Skip N results
  --fts
        Use FTS5 full-text search
  --fts-table
        FTS table name
  --no-fts
        Disable full-text search, use substring search only
  -R, --related
        Find media related to the first result
  --category
        Filter by category
  --genre
        Filter by genre
  --language
        Filter by language
  -e, --ext
        Filter by extensions (e.g., .mp4,.mkv)
  --video-only
        Only video files
  --audio-only
        Only audio files
  --image-only
        Only image files
  --text-only
        Only text/ebook files
  --portrait
        Only portrait orientation files
  --scan-subtitles
        Scan for external subtitles during import
  --online-media-only
        Exclude local media
  --local-media-only
        Exclude online media
  --mime-type
        Filter by mimetype substring (e.g., video, mp4)
  --no-mime-type
        Exclude by mimetype substring
  --no-default-categories
        Disable default categories
  -O, --play-in-order
        Play media in order
  --no-play-in-order
        Don't play media in order
  --loop
        Loop playback
  -M, --mute
        Start playback muted
  --override-player
        Override default player (e.g. --player 'vlc')
  --start
        Start playback at specific time/percentage
  --end
        Stop playback at specific time/percentage
  --volume
        Set initial volume (0-100)
  --fullscreen
        Start in fullscreen
  --no-subtitles
        Disable subtitles
  --subtitle-mix
        Probability to play no-subtitle content
  -4, --interdimensional-cable
        Duration to play (in seconds) while changing the channel
  --speed
        Playback speed
  --save-playhead
        Save playback position on quit
  --mpv-socket
        Mpv socket path
  --watch-later-dir
        Mpv watch_later directory
  --player-args-sub
        Player arguments for videos with subtitles
  --player-args-no-sub
        Player arguments for videos without subtitles
  --cast
        Cast to chromecast groups
  --cast-device
        Chromecast device name
  --cast-with-local
        Play music locally at the same time as chromecast
  --open
        Open results in media player
  --overlap
        Overlap in seconds for merging captions
```

</details>

### playlists

List scan roots (playlists)

<details><summary>All Options</summary>

```bash
$ disco playlists --help

Flags:
  -v, --verbose
        Enable verbose logging
  --simulate
        Dry run; don't actually do anything
  -y, --no-confirm
        Don't ask for confirmation
  -T, --timeout
        Quit after N minutes/seconds
  -c, --columns
        Columns to display
  -j, --json
        Output results as JSON
  --summarize
        Print aggregate statistics
  -f, --frequency
        Group statistics by time frequency (daily, weekly, monthly, yearly)
  --tui
        Interactive TUI mode
```

</details>

### search-db

Search arbitrary database table

<details><summary>All Options</summary>

```bash
$ disco search-db --help

Flags:
  -v, --verbose
        Enable verbose logging
  --simulate
        Dry run; don't actually do anything
  -y, --no-confirm
        Don't ask for confirmation
  -T, --timeout
        Quit after N minutes/seconds
  -q, --query
        Raw SQL query (overrides all query building)
  -L, --limit
        Limit results per database
  -a, --all
        Return all results (no limit)
  --offset
        Skip N results
  -s, --include
        Include paths matching pattern
  -E, --exclude
        Exclude paths matching pattern
  --regex
        Filter paths by regex pattern
  --path-contains
        Path must contain all these strings
  --paths
        Exact paths to include
  --search
        Search terms (space-separated for AND, | for OR)
  -S, --size
        Size range (e.g., >100MB, 1GB%10)
  -d, --duration
        Duration range (e.g., >1hour, 30min%10)
  --modified
        Filter by modification time
  --created
        Filter by creation time
  --downloaded
        Filter by download time
  --duration-from-size
        Constrain media to duration of videos which match any size constraints
  --watched
        Filter by watched status (true/false)
  --unfinished
        Has playhead but not finished
  -P, --partial
        Filter by partial playback status
  --play-count-min
        Minimum play count
  --play-count-max
        Maximum play count
  --completed
        Show only completed items
  --in-progress
        Show only items in progress
  --with-captions
        Show only items with captions
  --flexible-search
        Flexible search (fuzzy)
  --exact
        Exact match for search
  -w, --where
        SQL where clause(s)
  --exists
        Filter out non-existent files
  -o, --fetch-siblings
        Fetch siblings of matched files (each, all, if-audiobook)
  --fetch-siblings-max
        Maximum number of siblings to fetch
  --category
        Filter by category
  --genre
        Filter by genre
  --language
        Filter by language
  -e, --ext
        Filter by extensions (e.g., .mp4,.mkv)
  --video-only
        Only video files
  --audio-only
        Only audio files
  --image-only
        Only image files
  --text-only
        Only text/ebook files
  --portrait
        Only portrait orientation files
  --scan-subtitles
        Scan for external subtitles during import
  --online-media-only
        Exclude local media
  --local-media-only
        Exclude online media
  --mime-type
        Filter by mimetype substring (e.g., video, mp4)
  --no-mime-type
        Exclude by mimetype substring
  --no-default-categories
        Disable default categories
  --created-after
        Created after date (YYYY-MM-DD)
  --created-before
        Created before date (YYYY-MM-DD)
  --modified-after
        Modified after date (YYYY-MM-DD)
  --modified-before
        Modified before date (YYYY-MM-DD)
  --downloaded-after
        Downloaded after date (YYYY-MM-DD)
  --downloaded-before
        Downloaded before date (YYYY-MM-DD)
  --deleted-after
        Deleted after date (YYYY-MM-DD)
  --deleted-before
        Deleted before date (YYYY-MM-DD)
  --played-after
        Last played after date (YYYY-MM-DD)
  --played-before
        Last played before date (YYYY-MM-DD)
  --hide-deleted
        Exclude deleted files from results
  --only-deleted
        Include only deleted files in results
  -c, --columns
        Columns to display
  -j, --json
        Output results as JSON
  --summarize
        Print aggregate statistics
  -f, --frequency
        Group statistics by time frequency (daily, weekly, monthly, yearly)
  --tui
        Interactive TUI mode
  --trash
        Trash files after action
  --post-action
        Post-action: none, delete, mark-deleted, move, copy
  --delete-files
        Delete files after action
  --delete-rows
        Delete rows from database
  --mark-deleted
        Mark as deleted in database
  --move-to
        Move files to directory
  --copy-to
        Copy files to directory
  --action-limit
        Stop after N files
  --action-size
        Stop after N bytes (e.g., 10GB)
  --track-history
        Track playback history
```

</details>

### media-check

Check media files for corruption

<details><summary>All Options</summary>

```bash
$ disco media-check --help

Flags:
  -v, --verbose
        Enable verbose logging
  --simulate
        Dry run; don't actually do anything
  -y, --no-confirm
        Don't ask for confirmation
  -T, --timeout
        Quit after N minutes/seconds
  -s, --include
        Include paths matching pattern
  -E, --exclude
        Exclude paths matching pattern
  --regex
        Filter paths by regex pattern
  --path-contains
        Path must contain all these strings
  --paths
        Exact paths to include
  --search
        Search terms (space-separated for AND, | for OR)
  -S, --size
        Size range (e.g., >100MB, 1GB%10)
  -d, --duration
        Duration range (e.g., >1hour, 30min%10)
  --modified
        Filter by modification time
  --created
        Filter by creation time
  --downloaded
        Filter by download time
  --duration-from-size
        Constrain media to duration of videos which match any size constraints
  --watched
        Filter by watched status (true/false)
  --unfinished
        Has playhead but not finished
  -P, --partial
        Filter by partial playback status
  --play-count-min
        Minimum play count
  --play-count-max
        Maximum play count
  --completed
        Show only completed items
  --in-progress
        Show only items in progress
  --with-captions
        Show only items with captions
  --flexible-search
        Flexible search (fuzzy)
  --exact
        Exact match for search
  -w, --where
        SQL where clause(s)
  --exists
        Filter out non-existent files
  -o, --fetch-siblings
        Fetch siblings of matched files (each, all, if-audiobook)
  --fetch-siblings-max
        Maximum number of siblings to fetch
  --category
        Filter by category
  --genre
        Filter by genre
  --language
        Filter by language
  -e, --ext
        Filter by extensions (e.g., .mp4,.mkv)
  --video-only
        Only video files
  --audio-only
        Only audio files
  --image-only
        Only image files
  --text-only
        Only text/ebook files
  --portrait
        Only portrait orientation files
  --scan-subtitles
        Scan for external subtitles during import
  --online-media-only
        Exclude local media
  --local-media-only
        Exclude online media
  --mime-type
        Filter by mimetype substring (e.g., video, mp4)
  --no-mime-type
        Exclude by mimetype substring
  --no-default-categories
        Disable default categories
  --hide-deleted
        Exclude deleted files from results
  --only-deleted
        Include only deleted files in results
  --chunk-size
        Chunk size in seconds. If set, recommended to use >0.1 seconds
  --gap
        Width between chunks to skip. Values greater than 1 are treated as number of seconds
  --delete-corrupt
        Delete media that is more corrupt or equal to this threshold. Values greater than 1 are treated as number of seconds
  --full-scan-if-corrupt
        Full scan as second pass if initial scan result more corruption or equal to this threshold. Values greater than 1 are treated as number of seconds
  --full-scan
        Decode the full media file
  --audio-scan
        Count errors in audio track only
```

</details>

### files-info

Show information about files

<details><summary>All Options</summary>

```bash
$ disco files-info --help

Flags:
  -v, --verbose
        Enable verbose logging
  --simulate
        Dry run; don't actually do anything
  -y, --no-confirm
        Don't ask for confirmation
  -T, --timeout
        Quit after N minutes/seconds
  --search
        Search terms (space-separated for AND, | for OR)
  -S, --size
        Size range (e.g., >100MB, 1GB%10)
  -d, --duration
        Duration range (e.g., >1hour, 30min%10)
  --modified
        Filter by modification time
  --created
        Filter by creation time
  --downloaded
        Filter by download time
  --duration-from-size
        Constrain media to duration of videos which match any size constraints
  --watched
        Filter by watched status (true/false)
  --unfinished
        Has playhead but not finished
  -P, --partial
        Filter by partial playback status
  --play-count-min
        Minimum play count
  --play-count-max
        Maximum play count
  --completed
        Show only completed items
  --in-progress
        Show only items in progress
  --with-captions
        Show only items with captions
  --flexible-search
        Flexible search (fuzzy)
  --exact
        Exact match for search
  -w, --where
        SQL where clause(s)
  --exists
        Filter out non-existent files
  -o, --fetch-siblings
        Fetch siblings of matched files (each, all, if-audiobook)
  --fetch-siblings-max
        Maximum number of siblings to fetch
  --created-after
        Created after date (YYYY-MM-DD)
  --created-before
        Created before date (YYYY-MM-DD)
  --modified-after
        Modified after date (YYYY-MM-DD)
  --modified-before
        Modified before date (YYYY-MM-DD)
  --downloaded-after
        Downloaded after date (YYYY-MM-DD)
  --downloaded-before
        Downloaded before date (YYYY-MM-DD)
  --deleted-after
        Deleted after date (YYYY-MM-DD)
  --deleted-before
        Deleted before date (YYYY-MM-DD)
  --played-after
        Last played after date (YYYY-MM-DD)
  --played-before
        Last played before date (YYYY-MM-DD)
  --category
        Filter by category
  --genre
        Filter by genre
  --language
        Filter by language
  -e, --ext
        Filter by extensions (e.g., .mp4,.mkv)
  --video-only
        Only video files
  --audio-only
        Only audio files
  --image-only
        Only image files
  --text-only
        Only text/ebook files
  --portrait
        Only portrait orientation files
  --scan-subtitles
        Scan for external subtitles during import
  --online-media-only
        Exclude local media
  --local-media-only
        Exclude online media
  --mime-type
        Filter by mimetype substring (e.g., video, mp4)
  --no-mime-type
        Exclude by mimetype substring
  --no-default-categories
        Disable default categories
  -c, --columns
        Columns to display
  -j, --json
        Output results as JSON
  --summarize
        Print aggregate statistics
  -f, --frequency
        Group statistics by time frequency (daily, weekly, monthly, yearly)
  --tui
        Interactive TUI mode
```

</details>

### disk-usage

Show disk usage aggregation

Examples:

```bash
$ disco du my_videos.db
$ disco du my_videos.db --depth 2
```

<details><summary>All Options</summary>

```bash
$ disco disk-usage --help

Flags:
  -v, --verbose
        Enable verbose logging
  --simulate
        Dry run; don't actually do anything
  -y, --no-confirm
        Don't ask for confirmation
  -T, --timeout
        Quit after N minutes/seconds
  -s, --include
        Include paths matching pattern
  -E, --exclude
        Exclude paths matching pattern
  --regex
        Filter paths by regex pattern
  --path-contains
        Path must contain all these strings
  --paths
        Exact paths to include
  --search
        Search terms (space-separated for AND, | for OR)
  -S, --size
        Size range (e.g., >100MB, 1GB%10)
  -d, --duration
        Duration range (e.g., >1hour, 30min%10)
  --modified
        Filter by modification time
  --created
        Filter by creation time
  --downloaded
        Filter by download time
  --duration-from-size
        Constrain media to duration of videos which match any size constraints
  --watched
        Filter by watched status (true/false)
  --unfinished
        Has playhead but not finished
  -P, --partial
        Filter by partial playback status
  --play-count-min
        Minimum play count
  --play-count-max
        Maximum play count
  --completed
        Show only completed items
  --in-progress
        Show only items in progress
  --with-captions
        Show only items with captions
  --flexible-search
        Flexible search (fuzzy)
  --exact
        Exact match for search
  -w, --where
        SQL where clause(s)
  --exists
        Filter out non-existent files
  -o, --fetch-siblings
        Fetch siblings of matched files (each, all, if-audiobook)
  --fetch-siblings-max
        Maximum number of siblings to fetch
  --category
        Filter by category
  --genre
        Filter by genre
  --language
        Filter by language
  -e, --ext
        Filter by extensions (e.g., .mp4,.mkv)
  --video-only
        Only video files
  --audio-only
        Only audio files
  --image-only
        Only image files
  --text-only
        Only text/ebook files
  --portrait
        Only portrait orientation files
  --scan-subtitles
        Scan for external subtitles during import
  --online-media-only
        Exclude local media
  --local-media-only
        Exclude online media
  --mime-type
        Filter by mimetype substring (e.g., video, mp4)
  --no-mime-type
        Exclude by mimetype substring
  --no-default-categories
        Disable default categories
  --created-after
        Created after date (YYYY-MM-DD)
  --created-before
        Created before date (YYYY-MM-DD)
  --modified-after
        Modified after date (YYYY-MM-DD)
  --modified-before
        Modified before date (YYYY-MM-DD)
  --downloaded-after
        Downloaded after date (YYYY-MM-DD)
  --downloaded-before
        Downloaded before date (YYYY-MM-DD)
  --deleted-after
        Deleted after date (YYYY-MM-DD)
  --deleted-before
        Deleted before date (YYYY-MM-DD)
  --played-after
        Last played after date (YYYY-MM-DD)
  --played-before
        Last played before date (YYYY-MM-DD)
  --hide-deleted
        Exclude deleted files from results
  --only-deleted
        Include only deleted files in results
  -u, --sort-by
        Sort by field
  -V, --reverse
        Reverse sort order
  -n, --nat-sort
        Use natural sorting
  -r, --random
        Random order
  -k, --re-rank
        Add key/value pairs re-rank sorting by multiple attributes (COLUMN=WEIGHT)
  -c, --columns
        Columns to display
  -j, --json
        Output results as JSON
  --summarize
        Print aggregate statistics
  -f, --frequency
        Group statistics by time frequency (daily, weekly, monthly, yearly)
  --tui
        Interactive TUI mode
  -B, --big-dirs
        Aggregate by parent directory
  --file-counts
        Filter by number of files in directory (e.g., >5, 10%1)
  --group-by-extensions
        Group by file extensions
  --group-by-mime-types
        Group by mimetypes
  --group-by-size
        Group by size buckets
  --group-by-parent
        Group media by parent directory with counts and totals
  -D, --depth
        Aggregate at specific directory depth
  --min-depth
        Minimum depth for aggregation
  --max-depth
        Maximum depth for aggregation
  --parents
        Include parent directories in aggregation
  --folders-only
        Only show folders
  --files-only
        Only show files
  --folder-sizes
        Filter folders by total size
  --folder-counts
        Filter folders by number of subfolders
  --fts
        Use FTS5 full-text search
  --fts-table
        FTS table name
  --no-fts
        Disable full-text search, use substring search only
  -R, --related
        Find media related to the first result
```

</details>

### dedupe

Dedupe similar media

<details><summary>All Options</summary>

```bash
$ disco dedupe --help

Flags:
  -v, --verbose
        Enable verbose logging
  --simulate
        Dry run; don't actually do anything
  -y, --no-confirm
        Don't ask for confirmation
  -T, --timeout
        Quit after N minutes/seconds
  -s, --include
        Include paths matching pattern
  -E, --exclude
        Exclude paths matching pattern
  --regex
        Filter paths by regex pattern
  --path-contains
        Path must contain all these strings
  --paths
        Exact paths to include
  --search
        Search terms (space-separated for AND, | for OR)
  -S, --size
        Size range (e.g., >100MB, 1GB%10)
  -d, --duration
        Duration range (e.g., >1hour, 30min%10)
  --modified
        Filter by modification time
  --created
        Filter by creation time
  --downloaded
        Filter by download time
  --duration-from-size
        Constrain media to duration of videos which match any size constraints
  --watched
        Filter by watched status (true/false)
  --unfinished
        Has playhead but not finished
  -P, --partial
        Filter by partial playback status
  --play-count-min
        Minimum play count
  --play-count-max
        Maximum play count
  --completed
        Show only completed items
  --in-progress
        Show only items in progress
  --with-captions
        Show only items with captions
  --flexible-search
        Flexible search (fuzzy)
  --exact
        Exact match for search
  -w, --where
        SQL where clause(s)
  --exists
        Filter out non-existent files
  -o, --fetch-siblings
        Fetch siblings of matched files (each, all, if-audiobook)
  --fetch-siblings-max
        Maximum number of siblings to fetch
  --category
        Filter by category
  --genre
        Filter by genre
  --language
        Filter by language
  -e, --ext
        Filter by extensions (e.g., .mp4,.mkv)
  --video-only
        Only video files
  --audio-only
        Only audio files
  --image-only
        Only image files
  --text-only
        Only text/ebook files
  --portrait
        Only portrait orientation files
  --scan-subtitles
        Scan for external subtitles during import
  --online-media-only
        Exclude local media
  --local-media-only
        Exclude online media
  --mime-type
        Filter by mimetype substring (e.g., video, mp4)
  --no-mime-type
        Exclude by mimetype substring
  --no-default-categories
        Disable default categories
  --created-after
        Created after date (YYYY-MM-DD)
  --created-before
        Created before date (YYYY-MM-DD)
  --modified-after
        Modified after date (YYYY-MM-DD)
  --modified-before
        Modified before date (YYYY-MM-DD)
  --downloaded-after
        Downloaded after date (YYYY-MM-DD)
  --downloaded-before
        Downloaded before date (YYYY-MM-DD)
  --deleted-after
        Deleted after date (YYYY-MM-DD)
  --deleted-before
        Deleted before date (YYYY-MM-DD)
  --played-after
        Last played after date (YYYY-MM-DD)
  --played-before
        Last played before date (YYYY-MM-DD)
  --hide-deleted
        Exclude deleted files from results
  --only-deleted
        Include only deleted files in results
  --audio
        Dedupe database by artist + album + title
  --extractor-id
        Dedupe database by extractor_id
  --title-only
        Dedupe database by title
  --duration-only
        Dedupe database by duration
  --filesystem
        Dedupe filesystem database (hash)
  --compare-dirs
        Compare directories
  --basename
        Match by basename similarity
  --dirname
        Match by dirname similarity
  --min-similarity-ratio
        Filter out matches with less than this ratio (0.7-0.9)
  --dedupe-cmd
        Command to run for deduplication (rmlint-style: cmd duplicate keep)
  --trash
        Trash files after action
  --post-action
        Post-action: none, delete, mark-deleted, move, copy
  --delete-files
        Delete files after action
  --delete-rows
        Delete rows from database
  --mark-deleted
        Mark as deleted in database
  --move-to
        Move files to directory
  --copy-to
        Copy files to directory
  --action-limit
        Stop after N files
  --action-size
        Stop after N bytes (e.g., 10GB)
  --track-history
        Track playback history
  --hash-gap
        Gap between segments (0.0-1.0 as percentage of file size, or absolute bytes if >1)
  --hash-chunk-size
        Size of each segment to hash
  --hash-threads
        Number of threads to use for hashing a single file
```

</details>

### big-dirs

Show big directories aggregation

<details><summary>All Options</summary>

```bash
$ disco big-dirs --help

Flags:
  -v, --verbose
        Enable verbose logging
  --simulate
        Dry run; don't actually do anything
  -y, --no-confirm
        Don't ask for confirmation
  -T, --timeout
        Quit after N minutes/seconds
  -s, --include
        Include paths matching pattern
  -E, --exclude
        Exclude paths matching pattern
  --regex
        Filter paths by regex pattern
  --path-contains
        Path must contain all these strings
  --paths
        Exact paths to include
  --search
        Search terms (space-separated for AND, | for OR)
  -S, --size
        Size range (e.g., >100MB, 1GB%10)
  -d, --duration
        Duration range (e.g., >1hour, 30min%10)
  --modified
        Filter by modification time
  --created
        Filter by creation time
  --downloaded
        Filter by download time
  --duration-from-size
        Constrain media to duration of videos which match any size constraints
  --watched
        Filter by watched status (true/false)
  --unfinished
        Has playhead but not finished
  -P, --partial
        Filter by partial playback status
  --play-count-min
        Minimum play count
  --play-count-max
        Maximum play count
  --completed
        Show only completed items
  --in-progress
        Show only items in progress
  --with-captions
        Show only items with captions
  --flexible-search
        Flexible search (fuzzy)
  --exact
        Exact match for search
  -w, --where
        SQL where clause(s)
  --exists
        Filter out non-existent files
  -o, --fetch-siblings
        Fetch siblings of matched files (each, all, if-audiobook)
  --fetch-siblings-max
        Maximum number of siblings to fetch
  --category
        Filter by category
  --genre
        Filter by genre
  --language
        Filter by language
  -e, --ext
        Filter by extensions (e.g., .mp4,.mkv)
  --video-only
        Only video files
  --audio-only
        Only audio files
  --image-only
        Only image files
  --text-only
        Only text/ebook files
  --portrait
        Only portrait orientation files
  --scan-subtitles
        Scan for external subtitles during import
  --online-media-only
        Exclude local media
  --local-media-only
        Exclude online media
  --mime-type
        Filter by mimetype substring (e.g., video, mp4)
  --no-mime-type
        Exclude by mimetype substring
  --no-default-categories
        Disable default categories
  --created-after
        Created after date (YYYY-MM-DD)
  --created-before
        Created before date (YYYY-MM-DD)
  --modified-after
        Modified after date (YYYY-MM-DD)
  --modified-before
        Modified before date (YYYY-MM-DD)
  --downloaded-after
        Downloaded after date (YYYY-MM-DD)
  --downloaded-before
        Downloaded before date (YYYY-MM-DD)
  --deleted-after
        Deleted after date (YYYY-MM-DD)
  --deleted-before
        Deleted before date (YYYY-MM-DD)
  --played-after
        Last played after date (YYYY-MM-DD)
  --played-before
        Last played before date (YYYY-MM-DD)
  --hide-deleted
        Exclude deleted files from results
  --only-deleted
        Include only deleted files in results
  -B, --big-dirs
        Aggregate by parent directory
  --file-counts
        Filter by number of files in directory (e.g., >5, 10%1)
  --group-by-extensions
        Group by file extensions
  --group-by-mime-types
        Group by mimetypes
  --group-by-size
        Group by size buckets
  --group-by-parent
        Group media by parent directory with counts and totals
  -D, --depth
        Aggregate at specific directory depth
  --min-depth
        Minimum depth for aggregation
  --max-depth
        Maximum depth for aggregation
  --parents
        Include parent directories in aggregation
  --folders-only
        Only show folders
  --files-only
        Only show files
  --folder-sizes
        Filter folders by total size
  --folder-counts
        Filter folders by number of subfolders
  -u, --sort-by
        Sort by field
  -V, --reverse
        Reverse sort order
  -n, --nat-sort
        Use natural sorting
  -r, --random
        Random order
  -k, --re-rank
        Add key/value pairs re-rank sorting by multiple attributes (COLUMN=WEIGHT)
  -c, --columns
        Columns to display
  -j, --json
        Output results as JSON
  --summarize
        Print aggregate statistics
  -f, --frequency
        Group statistics by time frequency (daily, weekly, monthly, yearly)
  --tui
        Interactive TUI mode
```

</details>

### categorize

Auto-group media into categories

<details><summary>All Options</summary>

```bash
$ disco categorize --help

Flags:
  -v, --verbose
        Enable verbose logging
  --simulate
        Dry run; don't actually do anything
  -y, --no-confirm
        Don't ask for confirmation
  -T, --timeout
        Quit after N minutes/seconds
  -q, --query
        Raw SQL query (overrides all query building)
  -L, --limit
        Limit results per database
  -a, --all
        Return all results (no limit)
  --offset
        Skip N results
  -s, --include
        Include paths matching pattern
  -E, --exclude
        Exclude paths matching pattern
  --regex
        Filter paths by regex pattern
  --path-contains
        Path must contain all these strings
  --paths
        Exact paths to include
  --search
        Search terms (space-separated for AND, | for OR)
  -S, --size
        Size range (e.g., >100MB, 1GB%10)
  -d, --duration
        Duration range (e.g., >1hour, 30min%10)
  --modified
        Filter by modification time
  --created
        Filter by creation time
  --downloaded
        Filter by download time
  --duration-from-size
        Constrain media to duration of videos which match any size constraints
  --watched
        Filter by watched status (true/false)
  --unfinished
        Has playhead but not finished
  -P, --partial
        Filter by partial playback status
  --play-count-min
        Minimum play count
  --play-count-max
        Maximum play count
  --completed
        Show only completed items
  --in-progress
        Show only items in progress
  --with-captions
        Show only items with captions
  --flexible-search
        Flexible search (fuzzy)
  --exact
        Exact match for search
  -w, --where
        SQL where clause(s)
  --exists
        Filter out non-existent files
  -o, --fetch-siblings
        Fetch siblings of matched files (each, all, if-audiobook)
  --fetch-siblings-max
        Maximum number of siblings to fetch
  --category
        Filter by category
  --genre
        Filter by genre
  --language
        Filter by language
  -e, --ext
        Filter by extensions (e.g., .mp4,.mkv)
  --video-only
        Only video files
  --audio-only
        Only audio files
  --image-only
        Only image files
  --text-only
        Only text/ebook files
  --portrait
        Only portrait orientation files
  --scan-subtitles
        Scan for external subtitles during import
  --online-media-only
        Exclude local media
  --local-media-only
        Exclude online media
  --mime-type
        Filter by mimetype substring (e.g., video, mp4)
  --no-mime-type
        Exclude by mimetype substring
  --no-default-categories
        Disable default categories
  --created-after
        Created after date (YYYY-MM-DD)
  --created-before
        Created before date (YYYY-MM-DD)
  --modified-after
        Modified after date (YYYY-MM-DD)
  --modified-before
        Modified before date (YYYY-MM-DD)
  --downloaded-after
        Downloaded after date (YYYY-MM-DD)
  --downloaded-before
        Downloaded before date (YYYY-MM-DD)
  --deleted-after
        Deleted after date (YYYY-MM-DD)
  --deleted-before
        Deleted before date (YYYY-MM-DD)
  --played-after
        Last played after date (YYYY-MM-DD)
  --played-before
        Last played before date (YYYY-MM-DD)
  --hide-deleted
        Exclude deleted files from results
  --only-deleted
        Include only deleted files in results
  --trash
        Trash files after action
  --post-action
        Post-action: none, delete, mark-deleted, move, copy
  --delete-files
        Delete files after action
  --delete-rows
        Delete rows from database
  --mark-deleted
        Mark as deleted in database
  --move-to
        Move files to directory
  --copy-to
        Copy files to directory
  --action-limit
        Stop after N files
  --action-size
        Stop after N bytes (e.g., 10GB)
  --track-history
        Track playback history
  --other
        Analyze 'other' category to find potential new categories
  --full-path
        Use full path for categorization suggestions instead of just filename
```

</details>

### similar-files

Find similar files

<details><summary>All Options</summary>

```bash
$ disco similar-files --help

Flags:
  -v, --verbose
        Enable verbose logging
  --simulate
        Dry run; don't actually do anything
  -y, --no-confirm
        Don't ask for confirmation
  -T, --timeout
        Quit after N minutes/seconds
  -s, --include
        Include paths matching pattern
  -E, --exclude
        Exclude paths matching pattern
  --regex
        Filter paths by regex pattern
  --path-contains
        Path must contain all these strings
  --paths
        Exact paths to include
  --search
        Search terms (space-separated for AND, | for OR)
  -S, --size
        Size range (e.g., >100MB, 1GB%10)
  -d, --duration
        Duration range (e.g., >1hour, 30min%10)
  --modified
        Filter by modification time
  --created
        Filter by creation time
  --downloaded
        Filter by download time
  --duration-from-size
        Constrain media to duration of videos which match any size constraints
  --watched
        Filter by watched status (true/false)
  --unfinished
        Has playhead but not finished
  -P, --partial
        Filter by partial playback status
  --play-count-min
        Minimum play count
  --play-count-max
        Maximum play count
  --completed
        Show only completed items
  --in-progress
        Show only items in progress
  --with-captions
        Show only items with captions
  --flexible-search
        Flexible search (fuzzy)
  --exact
        Exact match for search
  -w, --where
        SQL where clause(s)
  --exists
        Filter out non-existent files
  -o, --fetch-siblings
        Fetch siblings of matched files (each, all, if-audiobook)
  --fetch-siblings-max
        Maximum number of siblings to fetch
  --category
        Filter by category
  --genre
        Filter by genre
  --language
        Filter by language
  -e, --ext
        Filter by extensions (e.g., .mp4,.mkv)
  --video-only
        Only video files
  --audio-only
        Only audio files
  --image-only
        Only image files
  --text-only
        Only text/ebook files
  --portrait
        Only portrait orientation files
  --scan-subtitles
        Scan for external subtitles during import
  --online-media-only
        Exclude local media
  --local-media-only
        Exclude online media
  --mime-type
        Filter by mimetype substring (e.g., video, mp4)
  --no-mime-type
        Exclude by mimetype substring
  --no-default-categories
        Disable default categories
  --created-after
        Created after date (YYYY-MM-DD)
  --created-before
        Created before date (YYYY-MM-DD)
  --modified-after
        Modified after date (YYYY-MM-DD)
  --modified-before
        Modified before date (YYYY-MM-DD)
  --downloaded-after
        Downloaded after date (YYYY-MM-DD)
  --downloaded-before
        Downloaded before date (YYYY-MM-DD)
  --deleted-after
        Deleted after date (YYYY-MM-DD)
  --deleted-before
        Deleted before date (YYYY-MM-DD)
  --played-after
        Last played after date (YYYY-MM-DD)
  --played-before
        Last played before date (YYYY-MM-DD)
  -u, --sort-by
        Sort by field
  -V, --reverse
        Reverse sort order
  -n, --nat-sort
        Use natural sorting
  -r, --random
        Random order
  -k, --re-rank
        Add key/value pairs re-rank sorting by multiple attributes (COLUMN=WEIGHT)
  -c, --columns
        Columns to display
  -j, --json
        Output results as JSON
  --summarize
        Print aggregate statistics
  -f, --frequency
        Group statistics by time frequency (daily, weekly, monthly, yearly)
  --tui
        Interactive TUI mode
  --similar
        Find similar files or folders
  --sizes-delta
        Size difference threshold (%)
  --counts-delta
        File count difference threshold (%)
  --durations-delta
        Duration difference threshold (%)
  --filter-names
        Cluster by name similarity
  --filter-sizes
        Cluster by size similarity
  --filter-counts
        Cluster by count similarity
  --filter-durations
        Cluster by duration similarity
  --total-sizes
        Compare total sizes (folders only)
  --total-durations
        Compare total durations (folders only)
  --only-duplicates
        Only show duplicate items
  --only-originals
        Only show original items
  -C, --cluster-sort
        Group items by similarity
  --clusters
        Number of clusters
  --tfidf
        Use TF-IDF for clustering
  --move-groups
        Move grouped files into separate directories
  --print-groups
        Print clusters as JSON
```

</details>

### similar-folders

Find similar folders

<details><summary>All Options</summary>

```bash
$ disco similar-folders --help

Flags:
  -v, --verbose
        Enable verbose logging
  --simulate
        Dry run; don't actually do anything
  -y, --no-confirm
        Don't ask for confirmation
  -T, --timeout
        Quit after N minutes/seconds
  -s, --include
        Include paths matching pattern
  -E, --exclude
        Exclude paths matching pattern
  --regex
        Filter paths by regex pattern
  --path-contains
        Path must contain all these strings
  --paths
        Exact paths to include
  --search
        Search terms (space-separated for AND, | for OR)
  -S, --size
        Size range (e.g., >100MB, 1GB%10)
  -d, --duration
        Duration range (e.g., >1hour, 30min%10)
  --modified
        Filter by modification time
  --created
        Filter by creation time
  --downloaded
        Filter by download time
  --duration-from-size
        Constrain media to duration of videos which match any size constraints
  --watched
        Filter by watched status (true/false)
  --unfinished
        Has playhead but not finished
  -P, --partial
        Filter by partial playback status
  --play-count-min
        Minimum play count
  --play-count-max
        Maximum play count
  --completed
        Show only completed items
  --in-progress
        Show only items in progress
  --with-captions
        Show only items with captions
  --flexible-search
        Flexible search (fuzzy)
  --exact
        Exact match for search
  -w, --where
        SQL where clause(s)
  --exists
        Filter out non-existent files
  -o, --fetch-siblings
        Fetch siblings of matched files (each, all, if-audiobook)
  --fetch-siblings-max
        Maximum number of siblings to fetch
  --category
        Filter by category
  --genre
        Filter by genre
  --language
        Filter by language
  -e, --ext
        Filter by extensions (e.g., .mp4,.mkv)
  --video-only
        Only video files
  --audio-only
        Only audio files
  --image-only
        Only image files
  --text-only
        Only text/ebook files
  --portrait
        Only portrait orientation files
  --scan-subtitles
        Scan for external subtitles during import
  --online-media-only
        Exclude local media
  --local-media-only
        Exclude online media
  --mime-type
        Filter by mimetype substring (e.g., video, mp4)
  --no-mime-type
        Exclude by mimetype substring
  --no-default-categories
        Disable default categories
  --created-after
        Created after date (YYYY-MM-DD)
  --created-before
        Created before date (YYYY-MM-DD)
  --modified-after
        Modified after date (YYYY-MM-DD)
  --modified-before
        Modified before date (YYYY-MM-DD)
  --downloaded-after
        Downloaded after date (YYYY-MM-DD)
  --downloaded-before
        Downloaded before date (YYYY-MM-DD)
  --deleted-after
        Deleted after date (YYYY-MM-DD)
  --deleted-before
        Deleted before date (YYYY-MM-DD)
  --played-after
        Last played after date (YYYY-MM-DD)
  --played-before
        Last played before date (YYYY-MM-DD)
  -u, --sort-by
        Sort by field
  -V, --reverse
        Reverse sort order
  -n, --nat-sort
        Use natural sorting
  -r, --random
        Random order
  -k, --re-rank
        Add key/value pairs re-rank sorting by multiple attributes (COLUMN=WEIGHT)
  -c, --columns
        Columns to display
  -j, --json
        Output results as JSON
  --summarize
        Print aggregate statistics
  -f, --frequency
        Group statistics by time frequency (daily, weekly, monthly, yearly)
  --tui
        Interactive TUI mode
  --similar
        Find similar files or folders
  --sizes-delta
        Size difference threshold (%)
  --counts-delta
        File count difference threshold (%)
  --durations-delta
        Duration difference threshold (%)
  --filter-names
        Cluster by name similarity
  --filter-sizes
        Cluster by size similarity
  --filter-counts
        Cluster by count similarity
  --filter-durations
        Cluster by duration similarity
  --total-sizes
        Compare total sizes (folders only)
  --total-durations
        Compare total durations (folders only)
  --only-duplicates
        Only show duplicate items
  --only-originals
        Only show original items
  -C, --cluster-sort
        Group items by similarity
  --clusters
        Number of clusters
  --tfidf
        Use TF-IDF for clustering
  --move-groups
        Move grouped files into separate directories
  --print-groups
        Print clusters as JSON
```

</details>

### watch

Watch videos with mpv

Examples:

```bash
$ disco watch my_videos.db
$ disco watch my_videos.db -r --limit 10
$ disco watch my_videos.db --size '>1GB'
```

<details><summary>All Options</summary>

```bash
$ disco watch --help

Flags:
  -v, --verbose
        Enable verbose logging
  --simulate
        Dry run; don't actually do anything
  -y, --no-confirm
        Don't ask for confirmation
  -T, --timeout
        Quit after N minutes/seconds
  -q, --query
        Raw SQL query (overrides all query building)
  -L, --limit
        Limit results per database
  -a, --all
        Return all results (no limit)
  --offset
        Skip N results
  -s, --include
        Include paths matching pattern
  -E, --exclude
        Exclude paths matching pattern
  --regex
        Filter paths by regex pattern
  --path-contains
        Path must contain all these strings
  --paths
        Exact paths to include
  --search
        Search terms (space-separated for AND, | for OR)
  -S, --size
        Size range (e.g., >100MB, 1GB%10)
  -d, --duration
        Duration range (e.g., >1hour, 30min%10)
  --modified
        Filter by modification time
  --created
        Filter by creation time
  --downloaded
        Filter by download time
  --duration-from-size
        Constrain media to duration of videos which match any size constraints
  --watched
        Filter by watched status (true/false)
  --unfinished
        Has playhead but not finished
  -P, --partial
        Filter by partial playback status
  --play-count-min
        Minimum play count
  --play-count-max
        Maximum play count
  --completed
        Show only completed items
  --in-progress
        Show only items in progress
  --with-captions
        Show only items with captions
  --flexible-search
        Flexible search (fuzzy)
  --exact
        Exact match for search
  -w, --where
        SQL where clause(s)
  --exists
        Filter out non-existent files
  -o, --fetch-siblings
        Fetch siblings of matched files (each, all, if-audiobook)
  --fetch-siblings-max
        Maximum number of siblings to fetch
  --category
        Filter by category
  --genre
        Filter by genre
  --language
        Filter by language
  -e, --ext
        Filter by extensions (e.g., .mp4,.mkv)
  --video-only
        Only video files
  --audio-only
        Only audio files
  --image-only
        Only image files
  --text-only
        Only text/ebook files
  --portrait
        Only portrait orientation files
  --scan-subtitles
        Scan for external subtitles during import
  --online-media-only
        Exclude local media
  --local-media-only
        Exclude online media
  --mime-type
        Filter by mimetype substring (e.g., video, mp4)
  --no-mime-type
        Exclude by mimetype substring
  --no-default-categories
        Disable default categories
  --created-after
        Created after date (YYYY-MM-DD)
  --created-before
        Created before date (YYYY-MM-DD)
  --modified-after
        Modified after date (YYYY-MM-DD)
  --modified-before
        Modified before date (YYYY-MM-DD)
  --downloaded-after
        Downloaded after date (YYYY-MM-DD)
  --downloaded-before
        Downloaded before date (YYYY-MM-DD)
  --deleted-after
        Deleted after date (YYYY-MM-DD)
  --deleted-before
        Deleted before date (YYYY-MM-DD)
  --played-after
        Last played after date (YYYY-MM-DD)
  --played-before
        Last played before date (YYYY-MM-DD)
  --hide-deleted
        Exclude deleted files from results
  --only-deleted
        Include only deleted files in results
  -u, --sort-by
        Sort by field
  -V, --reverse
        Reverse sort order
  -n, --nat-sort
        Use natural sorting
  -r, --random
        Random order
  -k, --re-rank
        Add key/value pairs re-rank sorting by multiple attributes (COLUMN=WEIGHT)
  -c, --columns
        Columns to display
  -j, --json
        Output results as JSON
  --summarize
        Print aggregate statistics
  -f, --frequency
        Group statistics by time frequency (daily, weekly, monthly, yearly)
  --tui
        Interactive TUI mode
  --fts
        Use FTS5 full-text search
  --fts-table
        FTS table name
  --no-fts
        Disable full-text search, use substring search only
  -R, --related
        Find media related to the first result
  -O, --play-in-order
        Play media in order
  --no-play-in-order
        Don't play media in order
  --loop
        Loop playback
  -M, --mute
        Start playback muted
  --override-player
        Override default player (e.g. --player 'vlc')
  --start
        Start playback at specific time/percentage
  --end
        Stop playback at specific time/percentage
  --volume
        Set initial volume (0-100)
  --fullscreen
        Start in fullscreen
  --no-subtitles
        Disable subtitles
  --subtitle-mix
        Probability to play no-subtitle content
  -4, --interdimensional-cable
        Duration to play (in seconds) while changing the channel
  --speed
        Playback speed
  --save-playhead
        Save playback position on quit
  --mpv-socket
        Mpv socket path
  --watch-later-dir
        Mpv watch_later directory
  --player-args-sub
        Player arguments for videos with subtitles
  --player-args-no-sub
        Player arguments for videos without subtitles
  --cast
        Cast to chromecast groups
  --cast-device
        Chromecast device name
  --cast-with-local
        Play music locally at the same time as chromecast
  --cmd-0
        Command to run if mpv exits with code 0
  --cmd-1
        Command to run if mpv exits with code 1
  --cmd-2
        Command to run if mpv exits with code 2
  --cmd-3
        Command to run if mpv exits with code 3
  --cmd-4
        Command to run if mpv exits with code 4
  --cmd-5
        Command to run if mpv exits with code 5
  --cmd-6
        Command to run if mpv exits with code 6
  --cmd-7
        Command to run if mpv exits with code 7
  --cmd-8
        Command to run if mpv exits with code 8
  --cmd-9
        Command to run if mpv exits with code 9
  --cmd-10
        Command to run if mpv exits with code 10
  --cmd-11
        Command to run if mpv exits with code 11
  --cmd-12
        Command to run if mpv exits with code 12
  --cmd-13
        Command to run if mpv exits with code 13
  --cmd-14
        Command to run if mpv exits with code 14
  --cmd-15
        Command to run if mpv exits with code 15
  --cmd-20
        Command to run if mpv exits with code 20
  --cmd-127
        Command to run if mpv exits with code 127
  -I, --interactive
        Interactive decision making after playback
  --trash
        Trash files after action
  --post-action
        Post-action: none, delete, mark-deleted, move, copy
  --delete-files
        Delete files after action
  --delete-rows
        Delete rows from database
  --mark-deleted
        Mark as deleted in database
  --move-to
        Move files to directory
  --copy-to
        Copy files to directory
  --action-limit
        Stop after N files
  --action-size
        Stop after N bytes (e.g., 10GB)
  --track-history
        Track playback history
```

</details>

### listen

Listen to audio with mpv

Examples:

```bash
$ disco listen my_music.db
$ disco listen my_music.db --random
```

<details><summary>All Options</summary>

```bash
$ disco listen --help

Flags:
  -v, --verbose
        Enable verbose logging
  --simulate
        Dry run; don't actually do anything
  -y, --no-confirm
        Don't ask for confirmation
  -T, --timeout
        Quit after N minutes/seconds
  -q, --query
        Raw SQL query (overrides all query building)
  -L, --limit
        Limit results per database
  -a, --all
        Return all results (no limit)
  --offset
        Skip N results
  -s, --include
        Include paths matching pattern
  -E, --exclude
        Exclude paths matching pattern
  --regex
        Filter paths by regex pattern
  --path-contains
        Path must contain all these strings
  --paths
        Exact paths to include
  --search
        Search terms (space-separated for AND, | for OR)
  -S, --size
        Size range (e.g., >100MB, 1GB%10)
  -d, --duration
        Duration range (e.g., >1hour, 30min%10)
  --modified
        Filter by modification time
  --created
        Filter by creation time
  --downloaded
        Filter by download time
  --duration-from-size
        Constrain media to duration of videos which match any size constraints
  --watched
        Filter by watched status (true/false)
  --unfinished
        Has playhead but not finished
  -P, --partial
        Filter by partial playback status
  --play-count-min
        Minimum play count
  --play-count-max
        Maximum play count
  --completed
        Show only completed items
  --in-progress
        Show only items in progress
  --with-captions
        Show only items with captions
  --flexible-search
        Flexible search (fuzzy)
  --exact
        Exact match for search
  -w, --where
        SQL where clause(s)
  --exists
        Filter out non-existent files
  -o, --fetch-siblings
        Fetch siblings of matched files (each, all, if-audiobook)
  --fetch-siblings-max
        Maximum number of siblings to fetch
  --category
        Filter by category
  --genre
        Filter by genre
  --language
        Filter by language
  -e, --ext
        Filter by extensions (e.g., .mp4,.mkv)
  --video-only
        Only video files
  --audio-only
        Only audio files
  --image-only
        Only image files
  --text-only
        Only text/ebook files
  --portrait
        Only portrait orientation files
  --scan-subtitles
        Scan for external subtitles during import
  --online-media-only
        Exclude local media
  --local-media-only
        Exclude online media
  --mime-type
        Filter by mimetype substring (e.g., video, mp4)
  --no-mime-type
        Exclude by mimetype substring
  --no-default-categories
        Disable default categories
  --created-after
        Created after date (YYYY-MM-DD)
  --created-before
        Created before date (YYYY-MM-DD)
  --modified-after
        Modified after date (YYYY-MM-DD)
  --modified-before
        Modified before date (YYYY-MM-DD)
  --downloaded-after
        Downloaded after date (YYYY-MM-DD)
  --downloaded-before
        Downloaded before date (YYYY-MM-DD)
  --deleted-after
        Deleted after date (YYYY-MM-DD)
  --deleted-before
        Deleted before date (YYYY-MM-DD)
  --played-after
        Last played after date (YYYY-MM-DD)
  --played-before
        Last played before date (YYYY-MM-DD)
  --hide-deleted
        Exclude deleted files from results
  --only-deleted
        Include only deleted files in results
  -u, --sort-by
        Sort by field
  -V, --reverse
        Reverse sort order
  -n, --nat-sort
        Use natural sorting
  -r, --random
        Random order
  -k, --re-rank
        Add key/value pairs re-rank sorting by multiple attributes (COLUMN=WEIGHT)
  -c, --columns
        Columns to display
  -j, --json
        Output results as JSON
  --summarize
        Print aggregate statistics
  -f, --frequency
        Group statistics by time frequency (daily, weekly, monthly, yearly)
  --tui
        Interactive TUI mode
  --fts
        Use FTS5 full-text search
  --fts-table
        FTS table name
  --no-fts
        Disable full-text search, use substring search only
  -R, --related
        Find media related to the first result
  -O, --play-in-order
        Play media in order
  --no-play-in-order
        Don't play media in order
  --loop
        Loop playback
  -M, --mute
        Start playback muted
  --override-player
        Override default player (e.g. --player 'vlc')
  --start
        Start playback at specific time/percentage
  --end
        Stop playback at specific time/percentage
  --volume
        Set initial volume (0-100)
  --fullscreen
        Start in fullscreen
  --no-subtitles
        Disable subtitles
  --subtitle-mix
        Probability to play no-subtitle content
  -4, --interdimensional-cable
        Duration to play (in seconds) while changing the channel
  --speed
        Playback speed
  --save-playhead
        Save playback position on quit
  --mpv-socket
        Mpv socket path
  --watch-later-dir
        Mpv watch_later directory
  --player-args-sub
        Player arguments for videos with subtitles
  --player-args-no-sub
        Player arguments for videos without subtitles
  --cast
        Cast to chromecast groups
  --cast-device
        Chromecast device name
  --cast-with-local
        Play music locally at the same time as chromecast
  --cmd-0
        Command to run if mpv exits with code 0
  --cmd-1
        Command to run if mpv exits with code 1
  --cmd-2
        Command to run if mpv exits with code 2
  --cmd-3
        Command to run if mpv exits with code 3
  --cmd-4
        Command to run if mpv exits with code 4
  --cmd-5
        Command to run if mpv exits with code 5
  --cmd-6
        Command to run if mpv exits with code 6
  --cmd-7
        Command to run if mpv exits with code 7
  --cmd-8
        Command to run if mpv exits with code 8
  --cmd-9
        Command to run if mpv exits with code 9
  --cmd-10
        Command to run if mpv exits with code 10
  --cmd-11
        Command to run if mpv exits with code 11
  --cmd-12
        Command to run if mpv exits with code 12
  --cmd-13
        Command to run if mpv exits with code 13
  --cmd-14
        Command to run if mpv exits with code 14
  --cmd-15
        Command to run if mpv exits with code 15
  --cmd-20
        Command to run if mpv exits with code 20
  --cmd-127
        Command to run if mpv exits with code 127
  -I, --interactive
        Interactive decision making after playback
  --trash
        Trash files after action
  --post-action
        Post-action: none, delete, mark-deleted, move, copy
  --delete-files
        Delete files after action
  --delete-rows
        Delete rows from database
  --mark-deleted
        Mark as deleted in database
  --move-to
        Move files to directory
  --copy-to
        Copy files to directory
  --action-limit
        Stop after N files
  --action-size
        Stop after N bytes (e.g., 10GB)
  --track-history
        Track playback history
```

</details>

### stats

Show library statistics

<details><summary>All Options</summary>

```bash
$ disco stats --help

Flags:
  -v, --verbose
        Enable verbose logging
  --simulate
        Dry run; don't actually do anything
  -y, --no-confirm
        Don't ask for confirmation
  -T, --timeout
        Quit after N minutes/seconds
  -s, --include
        Include paths matching pattern
  -E, --exclude
        Exclude paths matching pattern
  --regex
        Filter paths by regex pattern
  --path-contains
        Path must contain all these strings
  --paths
        Exact paths to include
  --search
        Search terms (space-separated for AND, | for OR)
  -S, --size
        Size range (e.g., >100MB, 1GB%10)
  -d, --duration
        Duration range (e.g., >1hour, 30min%10)
  --modified
        Filter by modification time
  --created
        Filter by creation time
  --downloaded
        Filter by download time
  --duration-from-size
        Constrain media to duration of videos which match any size constraints
  --watched
        Filter by watched status (true/false)
  --unfinished
        Has playhead but not finished
  -P, --partial
        Filter by partial playback status
  --play-count-min
        Minimum play count
  --play-count-max
        Maximum play count
  --completed
        Show only completed items
  --in-progress
        Show only items in progress
  --with-captions
        Show only items with captions
  --flexible-search
        Flexible search (fuzzy)
  --exact
        Exact match for search
  -w, --where
        SQL where clause(s)
  --exists
        Filter out non-existent files
  -o, --fetch-siblings
        Fetch siblings of matched files (each, all, if-audiobook)
  --fetch-siblings-max
        Maximum number of siblings to fetch
  --category
        Filter by category
  --genre
        Filter by genre
  --language
        Filter by language
  -e, --ext
        Filter by extensions (e.g., .mp4,.mkv)
  --video-only
        Only video files
  --audio-only
        Only audio files
  --image-only
        Only image files
  --text-only
        Only text/ebook files
  --portrait
        Only portrait orientation files
  --scan-subtitles
        Scan for external subtitles during import
  --online-media-only
        Exclude local media
  --local-media-only
        Exclude online media
  --mime-type
        Filter by mimetype substring (e.g., video, mp4)
  --no-mime-type
        Exclude by mimetype substring
  --no-default-categories
        Disable default categories
  --created-after
        Created after date (YYYY-MM-DD)
  --created-before
        Created before date (YYYY-MM-DD)
  --modified-after
        Modified after date (YYYY-MM-DD)
  --modified-before
        Modified before date (YYYY-MM-DD)
  --downloaded-after
        Downloaded after date (YYYY-MM-DD)
  --downloaded-before
        Downloaded before date (YYYY-MM-DD)
  --deleted-after
        Deleted after date (YYYY-MM-DD)
  --deleted-before
        Deleted before date (YYYY-MM-DD)
  --played-after
        Last played after date (YYYY-MM-DD)
  --played-before
        Last played before date (YYYY-MM-DD)
  --hide-deleted
        Exclude deleted files from results
  --only-deleted
        Include only deleted files in results
  -c, --columns
        Columns to display
  -j, --json
        Output results as JSON
  --summarize
        Print aggregate statistics
  -f, --frequency
        Group statistics by time frequency (daily, weekly, monthly, yearly)
  --tui
        Interactive TUI mode
```

</details>

### history

Show playback history

Examples:

```bash
$ disco history my_videos.db
$ disco history my_videos.db --inprogress
```

<details><summary>All Options</summary>

```bash
$ disco history --help

Flags:
  -v, --verbose
        Enable verbose logging
  --simulate
        Dry run; don't actually do anything
  -y, --no-confirm
        Don't ask for confirmation
  -T, --timeout
        Quit after N minutes/seconds
  -s, --include
        Include paths matching pattern
  -E, --exclude
        Exclude paths matching pattern
  --regex
        Filter paths by regex pattern
  --path-contains
        Path must contain all these strings
  --paths
        Exact paths to include
  --search
        Search terms (space-separated for AND, | for OR)
  -S, --size
        Size range (e.g., >100MB, 1GB%10)
  -d, --duration
        Duration range (e.g., >1hour, 30min%10)
  --modified
        Filter by modification time
  --created
        Filter by creation time
  --downloaded
        Filter by download time
  --duration-from-size
        Constrain media to duration of videos which match any size constraints
  --watched
        Filter by watched status (true/false)
  --unfinished
        Has playhead but not finished
  -P, --partial
        Filter by partial playback status
  --play-count-min
        Minimum play count
  --play-count-max
        Maximum play count
  --completed
        Show only completed items
  --in-progress
        Show only items in progress
  --with-captions
        Show only items with captions
  --flexible-search
        Flexible search (fuzzy)
  --exact
        Exact match for search
  -w, --where
        SQL where clause(s)
  --exists
        Filter out non-existent files
  -o, --fetch-siblings
        Fetch siblings of matched files (each, all, if-audiobook)
  --fetch-siblings-max
        Maximum number of siblings to fetch
  --category
        Filter by category
  --genre
        Filter by genre
  --language
        Filter by language
  -e, --ext
        Filter by extensions (e.g., .mp4,.mkv)
  --video-only
        Only video files
  --audio-only
        Only audio files
  --image-only
        Only image files
  --text-only
        Only text/ebook files
  --portrait
        Only portrait orientation files
  --scan-subtitles
        Scan for external subtitles during import
  --online-media-only
        Exclude local media
  --local-media-only
        Exclude online media
  --mime-type
        Filter by mimetype substring (e.g., video, mp4)
  --no-mime-type
        Exclude by mimetype substring
  --no-default-categories
        Disable default categories
  --created-after
        Created after date (YYYY-MM-DD)
  --created-before
        Created before date (YYYY-MM-DD)
  --modified-after
        Modified after date (YYYY-MM-DD)
  --modified-before
        Modified before date (YYYY-MM-DD)
  --downloaded-after
        Downloaded after date (YYYY-MM-DD)
  --downloaded-before
        Downloaded before date (YYYY-MM-DD)
  --deleted-after
        Deleted after date (YYYY-MM-DD)
  --deleted-before
        Deleted before date (YYYY-MM-DD)
  --played-after
        Last played after date (YYYY-MM-DD)
  --played-before
        Last played before date (YYYY-MM-DD)
  --hide-deleted
        Exclude deleted files from results
  --only-deleted
        Include only deleted files in results
  -u, --sort-by
        Sort by field
  -V, --reverse
        Reverse sort order
  -n, --nat-sort
        Use natural sorting
  -r, --random
        Random order
  -k, --re-rank
        Add key/value pairs re-rank sorting by multiple attributes (COLUMN=WEIGHT)
  -c, --columns
        Columns to display
  -j, --json
        Output results as JSON
  --summarize
        Print aggregate statistics
  -f, --frequency
        Group statistics by time frequency (daily, weekly, monthly, yearly)
  --tui
        Interactive TUI mode
  --trash
        Trash files after action
  --post-action
        Post-action: none, delete, mark-deleted, move, copy
  --delete-files
        Delete files after action
  --delete-rows
        Delete rows from database
  --mark-deleted
        Mark as deleted in database
  --move-to
        Move files to directory
  --copy-to
        Copy files to directory
  --action-limit
        Stop after N files
  --action-size
        Stop after N bytes (e.g., 10GB)
  --track-history
        Track playback history
```

</details>

### history-add

Add paths to playback history

<details><summary>All Options</summary>

```bash
$ disco history-add --help

Flags:
  -v, --verbose
        Enable verbose logging
  --simulate
        Dry run; don't actually do anything
  -y, --no-confirm
        Don't ask for confirmation
  -T, --timeout
        Quit after N minutes/seconds
  --done
        Mark as done
```

</details>

### mpv-watchlater

Import mpv watchlater files to history

<details><summary>All Options</summary>

```bash
$ disco mpv-watchlater --help

Flags:
  -v, --verbose
        Enable verbose logging
  --simulate
        Dry run; don't actually do anything
  -y, --no-confirm
        Don't ask for confirmation
  -T, --timeout
        Quit after N minutes/seconds
  --category
        Filter by category
  --genre
        Filter by genre
  --language
        Filter by language
  -e, --ext
        Filter by extensions (e.g., .mp4,.mkv)
  --video-only
        Only video files
  --audio-only
        Only audio files
  --image-only
        Only image files
  --text-only
        Only text/ebook files
  --portrait
        Only portrait orientation files
  --scan-subtitles
        Scan for external subtitles during import
  --online-media-only
        Exclude local media
  --local-media-only
        Exclude online media
  --mime-type
        Filter by mimetype substring (e.g., video, mp4)
  --no-mime-type
        Exclude by mimetype substring
  --no-default-categories
        Disable default categories
  -s, --include
        Include paths matching pattern
  -E, --exclude
        Exclude paths matching pattern
  --regex
        Filter paths by regex pattern
  --path-contains
        Path must contain all these strings
  --paths
        Exact paths to include
  --created-after
        Created after date (YYYY-MM-DD)
  --created-before
        Created before date (YYYY-MM-DD)
  --modified-after
        Modified after date (YYYY-MM-DD)
  --modified-before
        Modified before date (YYYY-MM-DD)
  --downloaded-after
        Downloaded after date (YYYY-MM-DD)
  --downloaded-before
        Downloaded before date (YYYY-MM-DD)
  --deleted-after
        Deleted after date (YYYY-MM-DD)
  --deleted-before
        Deleted before date (YYYY-MM-DD)
  --played-after
        Last played after date (YYYY-MM-DD)
  --played-before
        Last played before date (YYYY-MM-DD)
  --hide-deleted
        Exclude deleted files from results
  --only-deleted
        Include only deleted files in results
  -O, --play-in-order
        Play media in order
  --no-play-in-order
        Don't play media in order
  --loop
        Loop playback
  -M, --mute
        Start playback muted
  --override-player
        Override default player (e.g. --player 'vlc')
  --start
        Start playback at specific time/percentage
  --end
        Stop playback at specific time/percentage
  --volume
        Set initial volume (0-100)
  --fullscreen
        Start in fullscreen
  --no-subtitles
        Disable subtitles
  --subtitle-mix
        Probability to play no-subtitle content
  -4, --interdimensional-cable
        Duration to play (in seconds) while changing the channel
  --speed
        Playback speed
  --save-playhead
        Save playback position on quit
  --mpv-socket
        Mpv socket path
  --watch-later-dir
        Mpv watch_later directory
  --player-args-sub
        Player arguments for videos with subtitles
  --player-args-no-sub
        Player arguments for videos without subtitles
  --cast
        Cast to chromecast groups
  --cast-device
        Chromecast device name
  --cast-with-local
        Play music locally at the same time as chromecast
```

</details>

### serve

Start Web UI server

Examples:

```bash
$ disco serve my_videos.db my_music.db
$ disco serve --readonly my_videos.db
```

<details><summary>All Options</summary>

```bash
$ disco serve --help

Flags:
  -v, --verbose
        Enable verbose logging
  --simulate
        Dry run; don't actually do anything
  -y, --no-confirm
        Don't ask for confirmation
  -T, --timeout
        Quit after N minutes/seconds
  -q, --query
        Raw SQL query (overrides all query building)
  -L, --limit
        Limit results per database
  -a, --all
        Return all results (no limit)
  --offset
        Skip N results
  -s, --include
        Include paths matching pattern
  -E, --exclude
        Exclude paths matching pattern
  --regex
        Filter paths by regex pattern
  --path-contains
        Path must contain all these strings
  --paths
        Exact paths to include
  --search
        Search terms (space-separated for AND, | for OR)
  -S, --size
        Size range (e.g., >100MB, 1GB%10)
  -d, --duration
        Duration range (e.g., >1hour, 30min%10)
  --modified
        Filter by modification time
  --created
        Filter by creation time
  --downloaded
        Filter by download time
  --duration-from-size
        Constrain media to duration of videos which match any size constraints
  --watched
        Filter by watched status (true/false)
  --unfinished
        Has playhead but not finished
  -P, --partial
        Filter by partial playback status
  --play-count-min
        Minimum play count
  --play-count-max
        Maximum play count
  --completed
        Show only completed items
  --in-progress
        Show only items in progress
  --with-captions
        Show only items with captions
  --flexible-search
        Flexible search (fuzzy)
  --exact
        Exact match for search
  -w, --where
        SQL where clause(s)
  --exists
        Filter out non-existent files
  -o, --fetch-siblings
        Fetch siblings of matched files (each, all, if-audiobook)
  --fetch-siblings-max
        Maximum number of siblings to fetch
  --category
        Filter by category
  --genre
        Filter by genre
  --language
        Filter by language
  -e, --ext
        Filter by extensions (e.g., .mp4,.mkv)
  --video-only
        Only video files
  --audio-only
        Only audio files
  --image-only
        Only image files
  --text-only
        Only text/ebook files
  --portrait
        Only portrait orientation files
  --scan-subtitles
        Scan for external subtitles during import
  --online-media-only
        Exclude local media
  --local-media-only
        Exclude online media
  --mime-type
        Filter by mimetype substring (e.g., video, mp4)
  --no-mime-type
        Exclude by mimetype substring
  --no-default-categories
        Disable default categories
  --created-after
        Created after date (YYYY-MM-DD)
  --created-before
        Created before date (YYYY-MM-DD)
  --modified-after
        Modified after date (YYYY-MM-DD)
  --modified-before
        Modified before date (YYYY-MM-DD)
  --downloaded-after
        Downloaded after date (YYYY-MM-DD)
  --downloaded-before
        Downloaded before date (YYYY-MM-DD)
  --deleted-after
        Deleted after date (YYYY-MM-DD)
  --deleted-before
        Deleted before date (YYYY-MM-DD)
  --played-after
        Last played after date (YYYY-MM-DD)
  --played-before
        Last played before date (YYYY-MM-DD)
  --hide-deleted
        Exclude deleted files from results
  --only-deleted
        Include only deleted files in results
  -u, --sort-by
        Sort by field
  -V, --reverse
        Reverse sort order
  -n, --nat-sort
        Use natural sorting
  -r, --random
        Random order
  -k, --re-rank
        Add key/value pairs re-rank sorting by multiple attributes (COLUMN=WEIGHT)
  -c, --columns
        Columns to display
  -j, --json
        Output results as JSON
  --summarize
        Print aggregate statistics
  -f, --frequency
        Group statistics by time frequency (daily, weekly, monthly, yearly)
  --tui
        Interactive TUI mode
  -B, --big-dirs
        Aggregate by parent directory
  --file-counts
        Filter by number of files in directory (e.g., >5, 10%1)
  --group-by-extensions
        Group by file extensions
  --group-by-mime-types
        Group by mimetypes
  --group-by-size
        Group by size buckets
  --group-by-parent
        Group media by parent directory with counts and totals
  -D, --depth
        Aggregate at specific directory depth
  --min-depth
        Minimum depth for aggregation
  --max-depth
        Maximum depth for aggregation
  --parents
        Include parent directories in aggregation
  --folders-only
        Only show folders
  --files-only
        Only show files
  --folder-sizes
        Filter folders by total size
  --folder-counts
        Filter folders by number of subfolders
  -O, --play-in-order
        Play media in order
  --no-play-in-order
        Don't play media in order
  --loop
        Loop playback
  -M, --mute
        Start playback muted
  --override-player
        Override default player (e.g. --player 'vlc')
  --start
        Start playback at specific time/percentage
  --end
        Stop playback at specific time/percentage
  --volume
        Set initial volume (0-100)
  --fullscreen
        Start in fullscreen
  --no-subtitles
        Disable subtitles
  --subtitle-mix
        Probability to play no-subtitle content
  -4, --interdimensional-cable
        Duration to play (in seconds) while changing the channel
  --speed
        Playback speed
  --save-playhead
        Save playback position on quit
  --mpv-socket
        Mpv socket path
  --watch-later-dir
        Mpv watch_later directory
  --player-args-sub
        Player arguments for videos with subtitles
  --player-args-no-sub
        Player arguments for videos without subtitles
  --cast
        Cast to chromecast groups
  --cast-device
        Chromecast device name
  --cast-with-local
        Play music locally at the same time as chromecast
  --trash
        Trash files after action
  --post-action
        Post-action: none, delete, mark-deleted, move, copy
  --delete-files
        Delete files after action
  --delete-rows
        Delete rows from database
  --mark-deleted
        Mark as deleted in database
  --move-to
        Move files to directory
  --copy-to
        Copy files to directory
  --action-limit
        Stop after N files
  --action-size
        Stop after N bytes (e.g., 10GB)
  --track-history
        Track playback history
  --fts
        Use FTS5 full-text search
  --fts-table
        FTS table name
  --no-fts
        Disable full-text search, use substring search only
  -R, --related
        Find media related to the first result
  -p, --port
        Port to listen on
  --public-dir
        Override embedded web assets with local directory
  --dev
        Enable development mode (auto-reload)
  --read-only
        Disable write operations (progress tracking, playlist modifications, deletions)
  --no-browser
        Don't open browser on startup
```

</details>

### optimize

Optimize database (VACUUM, ANALYZE, FTS optimize)

Examples:

```bash
$ disco optimize my_videos.db
```

<details><summary>All Options</summary>

```bash
$ disco optimize --help

Flags:
  -v, --verbose
        Enable verbose logging
  --simulate
        Dry run; don't actually do anything
  -y, --no-confirm
        Don't ask for confirmation
  -T, --timeout
        Quit after N minutes/seconds
```

</details>

### repair

Repair malformed database using sqlite3

<details><summary>All Options</summary>

```bash
$ disco repair --help

Flags:
  -v, --verbose
        Enable verbose logging
  --simulate
        Dry run; don't actually do anything
  -y, --no-confirm
        Don't ask for confirmation
  -T, --timeout
        Quit after N minutes/seconds
```

</details>

### tui

Interactive TUI media picker

<details><summary>All Options</summary>

```bash
$ disco tui --help

Flags:
  -v, --verbose
        Enable verbose logging
  --simulate
        Dry run; don't actually do anything
  -y, --no-confirm
        Don't ask for confirmation
  -T, --timeout
        Quit after N minutes/seconds
  -q, --query
        Raw SQL query (overrides all query building)
  -L, --limit
        Limit results per database
  -a, --all
        Return all results (no limit)
  --offset
        Skip N results
  -s, --include
        Include paths matching pattern
  -E, --exclude
        Exclude paths matching pattern
  --regex
        Filter paths by regex pattern
  --path-contains
        Path must contain all these strings
  --paths
        Exact paths to include
  --search
        Search terms (space-separated for AND, | for OR)
  -S, --size
        Size range (e.g., >100MB, 1GB%10)
  -d, --duration
        Duration range (e.g., >1hour, 30min%10)
  --modified
        Filter by modification time
  --created
        Filter by creation time
  --downloaded
        Filter by download time
  --duration-from-size
        Constrain media to duration of videos which match any size constraints
  --watched
        Filter by watched status (true/false)
  --unfinished
        Has playhead but not finished
  -P, --partial
        Filter by partial playback status
  --play-count-min
        Minimum play count
  --play-count-max
        Maximum play count
  --completed
        Show only completed items
  --in-progress
        Show only items in progress
  --with-captions
        Show only items with captions
  --flexible-search
        Flexible search (fuzzy)
  --exact
        Exact match for search
  -w, --where
        SQL where clause(s)
  --exists
        Filter out non-existent files
  -o, --fetch-siblings
        Fetch siblings of matched files (each, all, if-audiobook)
  --fetch-siblings-max
        Maximum number of siblings to fetch
  --category
        Filter by category
  --genre
        Filter by genre
  --language
        Filter by language
  -e, --ext
        Filter by extensions (e.g., .mp4,.mkv)
  --video-only
        Only video files
  --audio-only
        Only audio files
  --image-only
        Only image files
  --text-only
        Only text/ebook files
  --portrait
        Only portrait orientation files
  --scan-subtitles
        Scan for external subtitles during import
  --online-media-only
        Exclude local media
  --local-media-only
        Exclude online media
  --mime-type
        Filter by mimetype substring (e.g., video, mp4)
  --no-mime-type
        Exclude by mimetype substring
  --no-default-categories
        Disable default categories
  --created-after
        Created after date (YYYY-MM-DD)
  --created-before
        Created before date (YYYY-MM-DD)
  --modified-after
        Modified after date (YYYY-MM-DD)
  --modified-before
        Modified before date (YYYY-MM-DD)
  --downloaded-after
        Downloaded after date (YYYY-MM-DD)
  --downloaded-before
        Downloaded before date (YYYY-MM-DD)
  --deleted-after
        Deleted after date (YYYY-MM-DD)
  --deleted-before
        Deleted before date (YYYY-MM-DD)
  --played-after
        Last played after date (YYYY-MM-DD)
  --played-before
        Last played before date (YYYY-MM-DD)
  --hide-deleted
        Exclude deleted files from results
  --only-deleted
        Include only deleted files in results
  -u, --sort-by
        Sort by field
  -V, --reverse
        Reverse sort order
  -n, --nat-sort
        Use natural sorting
  -r, --random
        Random order
  -k, --re-rank
        Add key/value pairs re-rank sorting by multiple attributes (COLUMN=WEIGHT)
  -c, --columns
        Columns to display
  -j, --json
        Output results as JSON
  --summarize
        Print aggregate statistics
  -f, --frequency
        Group statistics by time frequency (daily, weekly, monthly, yearly)
  --tui
        Interactive TUI mode
  --fts
        Use FTS5 full-text search
  --fts-table
        FTS table name
  --no-fts
        Disable full-text search, use substring search only
  -R, --related
        Find media related to the first result
```

</details>

### readme

Generate README.md content

<details><summary>All Options</summary>

```bash
$ disco readme --help

Flags:
  -v, --verbose
        Enable verbose logging
  --simulate
        Dry run; don't actually do anything
  -y, --no-confirm
        Don't ask for confirmation
  -T, --timeout
        Quit after N minutes/seconds
  -q, --query
        Raw SQL query (overrides all query building)
  -L, --limit
        Limit results per database
  -a, --all
        Return all results (no limit)
  --offset
        Skip N results
  -s, --include
        Include paths matching pattern
  -E, --exclude
        Exclude paths matching pattern
  --regex
        Filter paths by regex pattern
  --path-contains
        Path must contain all these strings
  --paths
        Exact paths to include
  --search
        Search terms (space-separated for AND, | for OR)
  -S, --size
        Size range (e.g., >100MB, 1GB%10)
  -d, --duration
        Duration range (e.g., >1hour, 30min%10)
  --modified
        Filter by modification time
  --created
        Filter by creation time
  --downloaded
        Filter by download time
  --duration-from-size
        Constrain media to duration of videos which match any size constraints
  --watched
        Filter by watched status (true/false)
  --unfinished
        Has playhead but not finished
  -P, --partial
        Filter by partial playback status
  --play-count-min
        Minimum play count
  --play-count-max
        Maximum play count
  --completed
        Show only completed items
  --in-progress
        Show only items in progress
  --with-captions
        Show only items with captions
  --flexible-search
        Flexible search (fuzzy)
  --exact
        Exact match for search
  -w, --where
        SQL where clause(s)
  --exists
        Filter out non-existent files
  -o, --fetch-siblings
        Fetch siblings of matched files (each, all, if-audiobook)
  --fetch-siblings-max
        Maximum number of siblings to fetch
  --category
        Filter by category
  --genre
        Filter by genre
  --language
        Filter by language
  -e, --ext
        Filter by extensions (e.g., .mp4,.mkv)
  --video-only
        Only video files
  --audio-only
        Only audio files
  --image-only
        Only image files
  --text-only
        Only text/ebook files
  --portrait
        Only portrait orientation files
  --scan-subtitles
        Scan for external subtitles during import
  --online-media-only
        Exclude local media
  --local-media-only
        Exclude online media
  --mime-type
        Filter by mimetype substring (e.g., video, mp4)
  --no-mime-type
        Exclude by mimetype substring
  --no-default-categories
        Disable default categories
  --created-after
        Created after date (YYYY-MM-DD)
  --created-before
        Created before date (YYYY-MM-DD)
  --modified-after
        Modified after date (YYYY-MM-DD)
  --modified-before
        Modified before date (YYYY-MM-DD)
  --downloaded-after
        Downloaded after date (YYYY-MM-DD)
  --downloaded-before
        Downloaded before date (YYYY-MM-DD)
  --deleted-after
        Deleted after date (YYYY-MM-DD)
  --deleted-before
        Deleted before date (YYYY-MM-DD)
  --played-after
        Last played after date (YYYY-MM-DD)
  --played-before
        Last played before date (YYYY-MM-DD)
  --hide-deleted
        Exclude deleted files from results
  --only-deleted
        Include only deleted files in results
  -u, --sort-by
        Sort by field
  -V, --reverse
        Reverse sort order
  -n, --nat-sort
        Use natural sorting
  -r, --random
        Random order
  -k, --re-rank
        Add key/value pairs re-rank sorting by multiple attributes (COLUMN=WEIGHT)
  -c, --columns
        Columns to display
  -j, --json
        Output results as JSON
  --summarize
        Print aggregate statistics
  -f, --frequency
        Group statistics by time frequency (daily, weekly, monthly, yearly)
  --tui
        Interactive TUI mode
  -B, --big-dirs
        Aggregate by parent directory
  --file-counts
        Filter by number of files in directory (e.g., >5, 10%1)
  --group-by-extensions
        Group by file extensions
  --group-by-mime-types
        Group by mimetypes
  --group-by-size
        Group by size buckets
  --group-by-parent
        Group media by parent directory with counts and totals
  -D, --depth
        Aggregate at specific directory depth
  --min-depth
        Minimum depth for aggregation
  --max-depth
        Maximum depth for aggregation
  --parents
        Include parent directories in aggregation
  --folders-only
        Only show folders
  --files-only
        Only show files
  --folder-sizes
        Filter folders by total size
  --folder-counts
        Filter folders by number of subfolders
  --regex-sort
        Sort by splitting lines and sorting words
  --regexs
        Regex patterns for line splitting
  --word-sorts
        Word sorting strategies
  --line-sorts
        Line sorting strategies
  --compat
        Use natsort compat mode
  --preprocess
        Remove junk common to filenames and URLs
  --stop-words
        List of words to ignore
  --duplicates
        Filter for duplicate words (true/false)
  --unique-only
        Filter for unique words (true/false)
  --similar
        Find similar files or folders
  --sizes-delta
        Size difference threshold (%)
  --counts-delta
        File count difference threshold (%)
  --durations-delta
        Duration difference threshold (%)
  --filter-names
        Cluster by name similarity
  --filter-sizes
        Cluster by size similarity
  --filter-counts
        Cluster by count similarity
  --filter-durations
        Cluster by duration similarity
  --total-sizes
        Compare total sizes (folders only)
  --total-durations
        Compare total durations (folders only)
  --only-duplicates
        Only show duplicate items
  --only-originals
        Only show original items
  -C, --cluster-sort
        Group items by similarity
  --clusters
        Number of clusters
  --tfidf
        Use TF-IDF for clustering
  --move-groups
        Move grouped files into separate directories
  --print-groups
        Print clusters as JSON
  --audio
        Dedupe database by artist + album + title
  --extractor-id
        Dedupe database by extractor_id
  --title-only
        Dedupe database by title
  --duration-only
        Dedupe database by duration
  --filesystem
        Dedupe filesystem database (hash)
  --compare-dirs
        Compare directories
  --basename
        Match by basename similarity
  --dirname
        Match by dirname similarity
  --min-similarity-ratio
        Filter out matches with less than this ratio (0.7-0.9)
  --dedupe-cmd
        Command to run for deduplication (rmlint-style: cmd duplicate keep)
  --fts
        Use FTS5 full-text search
  --fts-table
        FTS table name
  --no-fts
        Disable full-text search, use substring search only
  -R, --related
        Find media related to the first result
  -O, --play-in-order
        Play media in order
  --no-play-in-order
        Don't play media in order
  --loop
        Loop playback
  -M, --mute
        Start playback muted
  --override-player
        Override default player (e.g. --player 'vlc')
  --start
        Start playback at specific time/percentage
  --end
        Stop playback at specific time/percentage
  --volume
        Set initial volume (0-100)
  --fullscreen
        Start in fullscreen
  --no-subtitles
        Disable subtitles
  --subtitle-mix
        Probability to play no-subtitle content
  -4, --interdimensional-cable
        Duration to play (in seconds) while changing the channel
  --speed
        Playback speed
  --save-playhead
        Save playback position on quit
  --mpv-socket
        Mpv socket path
  --watch-later-dir
        Mpv watch_later directory
  --player-args-sub
        Player arguments for videos with subtitles
  --player-args-no-sub
        Player arguments for videos without subtitles
  --cast
        Cast to chromecast groups
  --cast-device
        Chromecast device name
  --cast-with-local
        Play music locally at the same time as chromecast
  --cmd-0
        Command to run if mpv exits with code 0
  --cmd-1
        Command to run if mpv exits with code 1
  --cmd-2
        Command to run if mpv exits with code 2
  --cmd-3
        Command to run if mpv exits with code 3
  --cmd-4
        Command to run if mpv exits with code 4
  --cmd-5
        Command to run if mpv exits with code 5
  --cmd-6
        Command to run if mpv exits with code 6
  --cmd-7
        Command to run if mpv exits with code 7
  --cmd-8
        Command to run if mpv exits with code 8
  --cmd-9
        Command to run if mpv exits with code 9
  --cmd-10
        Command to run if mpv exits with code 10
  --cmd-11
        Command to run if mpv exits with code 11
  --cmd-12
        Command to run if mpv exits with code 12
  --cmd-13
        Command to run if mpv exits with code 13
  --cmd-14
        Command to run if mpv exits with code 14
  --cmd-15
        Command to run if mpv exits with code 15
  --cmd-20
        Command to run if mpv exits with code 20
  --cmd-127
        Command to run if mpv exits with code 127
  -I, --interactive
        Interactive decision making after playback
  --trash
        Trash files after action
  --post-action
        Post-action: none, delete, mark-deleted, move, copy
  --delete-files
        Delete files after action
  --delete-rows
        Delete rows from database
  --mark-deleted
        Mark as deleted in database
  --move-to
        Move files to directory
  --copy-to
        Copy files to directory
  --action-limit
        Stop after N files
  --action-size
        Stop after N bytes (e.g., 10GB)
  --track-history
        Track playback history
  --hash-gap
        Gap between segments (0.0-1.0 as percentage of file size, or absolute bytes if >1)
  --hash-chunk-size
        Size of each segment to hash
  --hash-threads
        Number of threads to use for hashing a single file
  -t, --only-tables
        Comma separated specific table(s)
  --primary-keys
        Comma separated primary keys
  --business-keys
        Comma separated business keys
  --upsert
        Upsert rows on conflict
  --ignore
        Ignore rows on conflict (only-new-rows)
  --only-target-columns
        Only copy columns that exist in target
  --skip-columns
        Columns to skip during merge
  --databases
        Specific database paths to query (must be in server's allowed list). Can be specified multiple times.
```

</details>

### regex-sort

Sort by splitting lines and sorting words

<details><summary>All Options</summary>

```bash
$ disco regex-sort --help

Flags:
  -v, --verbose
        Enable verbose logging
  --simulate
        Dry run; don't actually do anything
  -y, --no-confirm
        Don't ask for confirmation
  -T, --timeout
        Quit after N minutes/seconds
  -u, --sort-by
        Sort by field
  -V, --reverse
        Reverse sort order
  -n, --nat-sort
        Use natural sorting
  -r, --random
        Random order
  -k, --re-rank
        Add key/value pairs re-rank sorting by multiple attributes (COLUMN=WEIGHT)
  --regex-sort
        Sort by splitting lines and sorting words
  --regexs
        Regex patterns for line splitting
  --word-sorts
        Word sorting strategies
  --line-sorts
        Line sorting strategies
  --compat
        Use natsort compat mode
  --preprocess
        Remove junk common to filenames and URLs
  --stop-words
        List of words to ignore
  --duplicates
        Filter for duplicate words (true/false)
  --unique-only
        Filter for unique words (true/false)
  --output-path
        Output file path (default stdout)
```

</details>

### cluster-sort

Group items by similarity

<details><summary>All Options</summary>

```bash
$ disco cluster-sort --help

Flags:
  -v, --verbose
        Enable verbose logging
  --simulate
        Dry run; don't actually do anything
  -y, --no-confirm
        Don't ask for confirmation
  -T, --timeout
        Quit after N minutes/seconds
  --similar
        Find similar files or folders
  --sizes-delta
        Size difference threshold (%)
  --counts-delta
        File count difference threshold (%)
  --durations-delta
        Duration difference threshold (%)
  --filter-names
        Cluster by name similarity
  --filter-sizes
        Cluster by size similarity
  --filter-counts
        Cluster by count similarity
  --filter-durations
        Cluster by duration similarity
  --total-sizes
        Compare total sizes (folders only)
  --total-durations
        Compare total durations (folders only)
  --only-duplicates
        Only show duplicate items
  --only-originals
        Only show original items
  -C, --cluster-sort
        Group items by similarity
  --clusters
        Number of clusters
  --tfidf
        Use TF-IDF for clustering
  --move-groups
        Move grouped files into separate directories
  --print-groups
        Print clusters as JSON
  --regex-sort
        Sort by splitting lines and sorting words
  --regexs
        Regex patterns for line splitting
  --word-sorts
        Word sorting strategies
  --line-sorts
        Line sorting strategies
  --compat
        Use natsort compat mode
  --preprocess
        Remove junk common to filenames and URLs
  --stop-words
        List of words to ignore
  --duplicates
        Filter for duplicate words (true/false)
  --unique-only
        Filter for unique words (true/false)
  --output-path
        Output file path (default stdout)
```

</details>

### sample-hash

Calculate a hash based on small file segments

<details><summary>All Options</summary>

```bash
$ disco sample-hash --help

Flags:
  -v, --verbose
        Enable verbose logging
  --simulate
        Dry run; don't actually do anything
  -y, --no-confirm
        Don't ask for confirmation
  -T, --timeout
        Quit after N minutes/seconds
  --hash-gap
        Gap between segments (0.0-1.0 as percentage of file size, or absolute bytes if >1)
  --hash-chunk-size
        Size of each segment to hash
  --hash-threads
        Number of threads to use for hashing a single file
  -c, --columns
        Columns to display
  -j, --json
        Output results as JSON
  --summarize
        Print aggregate statistics
  -f, --frequency
        Group statistics by time frequency (daily, weekly, monthly, yearly)
  --tui
        Interactive TUI mode
```

</details>

### open

Open files with default application

<details><summary>All Options</summary>

```bash
$ disco open --help

Flags:
  -v, --verbose
        Enable verbose logging
  --simulate
        Dry run; don't actually do anything
  -y, --no-confirm
        Don't ask for confirmation
  -T, --timeout
        Quit after N minutes/seconds
  -s, --include
        Include paths matching pattern
  -E, --exclude
        Exclude paths matching pattern
  --regex
        Filter paths by regex pattern
  --path-contains
        Path must contain all these strings
  --paths
        Exact paths to include
  --search
        Search terms (space-separated for AND, | for OR)
  -S, --size
        Size range (e.g., >100MB, 1GB%10)
  -d, --duration
        Duration range (e.g., >1hour, 30min%10)
  --modified
        Filter by modification time
  --created
        Filter by creation time
  --downloaded
        Filter by download time
  --duration-from-size
        Constrain media to duration of videos which match any size constraints
  --watched
        Filter by watched status (true/false)
  --unfinished
        Has playhead but not finished
  -P, --partial
        Filter by partial playback status
  --play-count-min
        Minimum play count
  --play-count-max
        Maximum play count
  --completed
        Show only completed items
  --in-progress
        Show only items in progress
  --with-captions
        Show only items with captions
  --flexible-search
        Flexible search (fuzzy)
  --exact
        Exact match for search
  -w, --where
        SQL where clause(s)
  --exists
        Filter out non-existent files
  -o, --fetch-siblings
        Fetch siblings of matched files (each, all, if-audiobook)
  --fetch-siblings-max
        Maximum number of siblings to fetch
  --category
        Filter by category
  --genre
        Filter by genre
  --language
        Filter by language
  -e, --ext
        Filter by extensions (e.g., .mp4,.mkv)
  --video-only
        Only video files
  --audio-only
        Only audio files
  --image-only
        Only image files
  --text-only
        Only text/ebook files
  --portrait
        Only portrait orientation files
  --scan-subtitles
        Scan for external subtitles during import
  --online-media-only
        Exclude local media
  --local-media-only
        Exclude online media
  --mime-type
        Filter by mimetype substring (e.g., video, mp4)
  --no-mime-type
        Exclude by mimetype substring
  --no-default-categories
        Disable default categories
  -u, --sort-by
        Sort by field
  -V, --reverse
        Reverse sort order
  -n, --nat-sort
        Use natural sorting
  -r, --random
        Random order
  -k, --re-rank
        Add key/value pairs re-rank sorting by multiple attributes (COLUMN=WEIGHT)
  --trash
        Trash files after action
  --post-action
        Post-action: none, delete, mark-deleted, move, copy
  --delete-files
        Delete files after action
  --delete-rows
        Delete rows from database
  --mark-deleted
        Mark as deleted in database
  --move-to
        Move files to directory
  --copy-to
        Copy files to directory
  --action-limit
        Stop after N files
  --action-size
        Stop after N bytes (e.g., 10GB)
  --track-history
        Track playback history
```

</details>

### browse

Open URLs in browser

<details><summary>All Options</summary>

```bash
$ disco browse --help

Flags:
  -v, --verbose
        Enable verbose logging
  --simulate
        Dry run; don't actually do anything
  -y, --no-confirm
        Don't ask for confirmation
  -T, --timeout
        Quit after N minutes/seconds
  -s, --include
        Include paths matching pattern
  -E, --exclude
        Exclude paths matching pattern
  --regex
        Filter paths by regex pattern
  --path-contains
        Path must contain all these strings
  --paths
        Exact paths to include
  --search
        Search terms (space-separated for AND, | for OR)
  -S, --size
        Size range (e.g., >100MB, 1GB%10)
  -d, --duration
        Duration range (e.g., >1hour, 30min%10)
  --modified
        Filter by modification time
  --created
        Filter by creation time
  --downloaded
        Filter by download time
  --duration-from-size
        Constrain media to duration of videos which match any size constraints
  --watched
        Filter by watched status (true/false)
  --unfinished
        Has playhead but not finished
  -P, --partial
        Filter by partial playback status
  --play-count-min
        Minimum play count
  --play-count-max
        Maximum play count
  --completed
        Show only completed items
  --in-progress
        Show only items in progress
  --with-captions
        Show only items with captions
  --flexible-search
        Flexible search (fuzzy)
  --exact
        Exact match for search
  -w, --where
        SQL where clause(s)
  --exists
        Filter out non-existent files
  -o, --fetch-siblings
        Fetch siblings of matched files (each, all, if-audiobook)
  --fetch-siblings-max
        Maximum number of siblings to fetch
  --category
        Filter by category
  --genre
        Filter by genre
  --language
        Filter by language
  -e, --ext
        Filter by extensions (e.g., .mp4,.mkv)
  --video-only
        Only video files
  --audio-only
        Only audio files
  --image-only
        Only image files
  --text-only
        Only text/ebook files
  --portrait
        Only portrait orientation files
  --scan-subtitles
        Scan for external subtitles during import
  --online-media-only
        Exclude local media
  --local-media-only
        Exclude online media
  --mime-type
        Filter by mimetype substring (e.g., video, mp4)
  --no-mime-type
        Exclude by mimetype substring
  --no-default-categories
        Disable default categories
  -u, --sort-by
        Sort by field
  -V, --reverse
        Reverse sort order
  -n, --nat-sort
        Use natural sorting
  -r, --random
        Random order
  -k, --re-rank
        Add key/value pairs re-rank sorting by multiple attributes (COLUMN=WEIGHT)
  --browser
        Browser to use
```

</details>

### now

Show current mpv playback status

<details><summary>All Options</summary>

```bash
$ disco now --help

Flags:
  --mpv-socket
        Mpv socket path
  --cast-device
        Chromecast device name
  -v, --verbose
        Enable verbose logging
```

</details>

### next

Skip to next file in mpv

<details><summary>All Options</summary>

```bash
$ disco next --help

Flags:
  --mpv-socket
        Mpv socket path
  --cast-device
        Chromecast device name
  -v, --verbose
        Enable verbose logging
```

</details>

### stop

Stop mpv playback

<details><summary>All Options</summary>

```bash
$ disco stop --help

Flags:
  --mpv-socket
        Mpv socket path
  --cast-device
        Chromecast device name
  -v, --verbose
        Enable verbose logging
```

</details>

### pause

Toggle mpv pause state

<details><summary>All Options</summary>

```bash
$ disco pause --help

Flags:
  --mpv-socket
        Mpv socket path
  --cast-device
        Chromecast device name
  -v, --verbose
        Enable verbose logging
```

</details>

### seek

Seek mpv playback

<details><summary>All Options</summary>

```bash
$ disco seek --help

Flags:
  --mpv-socket
        Mpv socket path
  --cast-device
        Chromecast device name
  -v, --verbose
        Enable verbose logging
```

</details>

### merge-dbs

Merge multiple SQLite databases

<details><summary>All Options</summary>

```bash
$ disco merge-dbs --help

Flags:
  -v, --verbose
        Enable verbose logging
  --simulate
        Dry run; don't actually do anything
  -y, --no-confirm
        Don't ask for confirmation
  -T, --timeout
        Quit after N minutes/seconds
  --search
        Search terms (space-separated for AND, | for OR)
  -S, --size
        Size range (e.g., >100MB, 1GB%10)
  -d, --duration
        Duration range (e.g., >1hour, 30min%10)
  --modified
        Filter by modification time
  --created
        Filter by creation time
  --downloaded
        Filter by download time
  --duration-from-size
        Constrain media to duration of videos which match any size constraints
  --watched
        Filter by watched status (true/false)
  --unfinished
        Has playhead but not finished
  -P, --partial
        Filter by partial playback status
  --play-count-min
        Minimum play count
  --play-count-max
        Maximum play count
  --completed
        Show only completed items
  --in-progress
        Show only items in progress
  --with-captions
        Show only items with captions
  --flexible-search
        Flexible search (fuzzy)
  --exact
        Exact match for search
  -w, --where
        SQL where clause(s)
  --exists
        Filter out non-existent files
  -o, --fetch-siblings
        Fetch siblings of matched files (each, all, if-audiobook)
  --fetch-siblings-max
        Maximum number of siblings to fetch
  -t, --only-tables
        Comma separated specific table(s)
  --primary-keys
        Comma separated primary keys
  --business-keys
        Comma separated business keys
  --upsert
        Upsert rows on conflict
  --ignore
        Ignore rows on conflict (only-new-rows)
  --only-target-columns
        Only copy columns that exist in target
  --skip-columns
        Columns to skip during merge
```

</details>

### explode

Create symlinks for all subcommands (busybox-style)

<details><summary>All Options</summary>

```bash
$ disco explode --help
```

</details>

### update

Check for and install updates from GitHub

<details><summary>All Options</summary>

```bash
$ disco update --help
```

</details>

### version

Show version and build information

<details><summary>All Options</summary>

```bash
$ disco version --help

Flags:
  -v, --verbose
        Show verbose output including dependencies
```

</details>

