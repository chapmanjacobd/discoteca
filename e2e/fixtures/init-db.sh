#!/bin/bash
# E2E Test Database Initialization Script
# This script creates a fresh test database with deterministic test data
# Run this whenever the schema changes or test data needs updating

set -e

# Get the repository root (parent of e2e directory)
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
FIXTURES_DIR="$SCRIPT_DIR"
MEDIA_DIR="$FIXTURES_DIR/media"
DB_FILE="$FIXTURES_DIR/test.db"

echo "=== E2E Test Database Initialization ==="
echo "Repository root: $REPO_ROOT"
echo "Fixtures directory: $FIXTURES_DIR"
echo "Media directory: $MEDIA_DIR"
echo "Database file: $DB_FILE"
echo ""

# Clean up old database and media
if [ -f "$DB_FILE" ]; then
    echo "Removing old database..."
    rm -f "$DB_FILE" "${DB_FILE}-wal" "${DB_FILE}-shm"
fi

if [ -d "$MEDIA_DIR" ]; then
    echo "Removing old media files..."
    rm -rf "$MEDIA_DIR"
fi

# Create fixtures directory if needed
mkdir -p "$FIXTURES_DIR"
mkdir -p "$MEDIA_DIR"/{videos,audio,images,documents}

# Generate playable media files with ffmpeg
echo "Generating test media files with ffmpeg..."

# Video files with audio tracks (playable in browser)
# Using common prefixes so keyword suggestions can find common words
ffmpeg -y -f lavfi -i testsrc=duration=10:size=320x240:rate=30 \
    -f lavfi -i sine=frequency=440:duration=10 \
    -c:v libx264 -c:a aac -movflags +faststart \
    "$MEDIA_DIR/videos/test_video1.mp4" 2>/dev/null

ffmpeg -y -f lavfi -i testsrc=duration=8:size=320x240:rate=30 \
    -f lavfi -i sine=frequency=523:duration=8 \
    -c:v libx264 -c:a aac -movflags +faststart \
    "$MEDIA_DIR/videos/test_video2.mp4" 2>/dev/null

ffmpeg -y -f lavfi -i testsrc=duration=5:size=320x240:rate=30 \
    -f lavfi -i sine=frequency=659:duration=5 \
    -c:v libx264 -c:a aac -movflags +faststart \
    "$MEDIA_DIR/videos/test_clip1.mp4" 2>/dev/null

ffmpeg -y -f lavfi -i testsrc=duration=3:size=320x240:rate=30 \
    -f lavfi -i sine=frequency=784:duration=3 \
    -c:v libx264 -c:a aac -movflags +faststart \
    "$MEDIA_DIR/videos/test_clip2.mp4" 2>/dev/null

# Audio files (playable in browser)
ffmpeg -y -f lavfi -i sine=frequency=440:duration=30 \
    -c:a libmp3lame -b:a 128k \
    "$MEDIA_DIR/audio/test_audio1.mp3" 2>/dev/null

ffmpeg -y -f lavfi -i sine=frequency=523:duration=25 \
    -c:a libmp3lame -b:a 128k \
    "$MEDIA_DIR/audio/test_audio2.mp3" 2>/dev/null

ffmpeg -y -f lavfi -i sine=frequency=330:duration=60 \
    -c:a libmp3lame -b:a 128k \
    "$MEDIA_DIR/audio/podcast.mp3" 2>/dev/null

# Valid PNG images for slideshow testing
ffmpeg -y -f lavfi -i color=c=red:s=640x480:d=1 \
    -vf "drawtext=fontfile=/usr/share/fonts/truetype/dejavu/DejaVuSans.ttf:fontsize=48:fontcolor=white:x=(w-text_w)/2:y=(h-text_h)/2:text='Image 1'" \
    -frames:v 1 "$MEDIA_DIR/images/test_image1.png" 2>/dev/null

ffmpeg -y -f lavfi -i color=c=green:s=640x480:d=1 \
    -vf "drawtext=fontfile=/usr/share/fonts/truetype/dejavu/DejaVuSans.ttf:fontsize=48:fontcolor=white:x=(w-text_w)/2:y=(h-text_h)/2:text='Image 2'" \
    -frames:v 1 "$MEDIA_DIR/images/test_image2.png" 2>/dev/null

ffmpeg -y -f lavfi -i color=c=blue:s=640x480:d=1 \
    -vf "drawtext=fontfile=/usr/share/fonts/truetype/dejavu/DejaVuSans.ttf:fontsize=48:fontcolor=white:x=(w-text_w)/2:y=(h-text_h)/2:text='Image 3'" \
    -frames:v 1 "$MEDIA_DIR/images/test_image3.png" 2>/dev/null

# Create valid PDF using pandoc
echo "# Test Document

This is a test PDF document for e2e testing.

It contains multiple lines to ensure it's a valid, readable PDF." | pandoc -f markdown -t pdf -o "$MEDIA_DIR/documents/test-document.pdf"

# Create EPUB using pandoc for testing calibre conversion
cat > "$MEDIA_DIR/documents/test-book.md" << 'MD'
---
title: Test Book
author: Test Author
language: en
---

# Chapter 1: Introduction

This is the first chapter of our test book. It contains some sample text to verify that EPUB conversion works correctly.

## Section 1.1

This is a subsection with more content. We need enough text to make sure the conversion produces multiple HTML files.

Lorem ipsum dolor sit amet, consectetur adipiscing elit. Sed do eiusmod tempor incididunt ut labore et dolore magna aliqua.

# Chapter 2: Main Content

This is the second chapter with the main content of the book. It should be converted to a separate HTML file by calibre.

## Section 2.1

More subsections to test the navigation. The calibre conversion should preserve the document structure.

Ut enim ad minim veniam, quis nostrud exercitation ullamco laboris nisi ut aliquip ex ea commodo consequat.

# Chapter 3: Conclusion

This is the final chapter. It wraps up the test book content.

Duis aute irure dolor in reprehenderit in voluptate velit esse cillum dolore eu fugiat nulla pariatur.
MD

# Convert markdown to EPUB using pandoc
pandoc -f markdown -t epub -o "$MEDIA_DIR/documents/test-book.epub" "$MEDIA_DIR/documents/test-book.md"
echo "Created EPUB: $MEDIA_DIR/documents/test-book.epub"

# VTT subtitle files (external subtitles for caption scanning)
cat > "$MEDIA_DIR/videos/test_video1.vtt" << 'VTT'
WEBVTT

00:00:15.500 --> 00:00:20.000
Welcome to the movie

00:00:30.000 --> 00:00:35.000
This is an exciting scene

00:01:00.000 --> 00:01:05.000
The plot thickens
VTT

cat > "$MEDIA_DIR/videos/test_video2.vtt" << 'VTT'
WEBVTT

00:00:20.000 --> 00:00:25.000
Opening scene

00:00:45.000 --> 00:00:50.000
Main character appears
VTT

cat > "$MEDIA_DIR/videos/test_clip1.vtt" << 'VTT'
WEBVTT

00:00:12.000 --> 00:00:15.000
Short clip caption
VTT

cat > "$MEDIA_DIR/videos/test_clip2.vtt" << 'VTT'
WEBVTT

00:00:15.000 --> 00:00:18.000
Another short clip
VTT

echo "Test media files generated successfully"

# Build disco binary if needed
if [ ! -f "$REPO_ROOT/disco" ]; then
    echo "Building disco binary..."
    cd "$REPO_ROOT" && go build -o disco ./cmd/disco
fi

# Create database with disco
echo "Creating database with disco..."
cd "$REPO_ROOT"
./disco add --scan-subtitles "$DB_FILE" "$MEDIA_DIR"

# Verify database
echo ""
echo "=== Database Summary ==="
sqlite3 "$DB_FILE" "SELECT 'Media count: ' || COUNT(*) FROM media;"
sqlite3 "$DB_FILE" "SELECT 'Caption count: ' || COUNT(*) FROM captions;" 2>/dev/null || echo "Captions table not found"

echo ""
echo "=== Exporting Schema ==="
# Export schema for tracking migrations
sqlite3 "$DB_FILE" ".schema" > "$FIXTURES_DIR/schema.sql"
echo "Schema exported to: $FIXTURES_DIR/schema.sql"

# Show schema hash for quick comparison
SCHEMA_HASH=$(md5sum "$FIXTURES_DIR/schema.sql" | cut -d' ' -f1)
echo "Schema hash: $SCHEMA_HASH"

echo ""
echo "=== Database created successfully! ==="
echo "Location: $DB_FILE"
echo "Schema exported to: $FIXTURES_DIR/schema.sql"
echo ""
echo "To regenerate (after schema changes):"
echo "  make e2e-init"
echo ""
echo "To review schema changes:"
echo "  git diff e2e/fixtures/schema.sql"
