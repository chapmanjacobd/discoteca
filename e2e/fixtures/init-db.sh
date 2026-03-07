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

# Clean up old database
if [ -f "$DB_FILE" ]; then
    echo "Removing old database..."
    rm -f "$DB_FILE" "${DB_FILE}-wal" "${DB_FILE}-shm"
fi

# Create fixtures directory if needed
mkdir -p "$FIXTURES_DIR"
mkdir -p "$MEDIA_DIR"/{videos,audio,images,documents}

# Create fake media files if they don't exist
if [ ! -f "$MEDIA_DIR/videos/movie1.mp4" ]; then
    echo "Creating fake media files..."
    
    # Minimal MP4 (ftyp box)
    printf '\x00\x00\x00\x18ftypisom\x00\x00\x00\x00isom\x00\x00\x00\x08free' > "$MEDIA_DIR/videos/movie1.mp4"
    printf '\x00\x00\x00\x18ftypisom\x00\x00\x00\x00isom\x00\x00\x00\x08free' > "$MEDIA_DIR/videos/movie2.mp4"
    printf '\x00\x00\x00\x18ftypisom\x00\x00\x00\x00isom\x00\x00\x00\x08free' > "$MEDIA_DIR/videos/clip1.mp4"
    printf '\x00\x00\x00\x18ftypisom\x00\x00\x00\x00isom\x00\x00\x00\x08free' > "$MEDIA_DIR/videos/clip2.mp4"
    
    # Minimal MP3 (ID3 header)
    printf '\x49\x44\x33\x03\x00\x00\x00\x00\x00\x00' > "$MEDIA_DIR/audio/song1.mp3"
    printf '\x49\x44\x33\x03\x00\x00\x00\x00\x00\x00' > "$MEDIA_DIR/audio/song2.mp3"
    printf '\x49\x44\x33\x03\x00\x00\x00\x00\x00\x00' > "$MEDIA_DIR/audio/podcast.mp3"
    
    # Minimal JPEG
    printf '\xFF\xD8\xFF\xE0\x00\x10JFIF\x00\x01\x01\x00\x00\x01\x00\x01\x00\x00\xFF\xD9' > "$MEDIA_DIR/images/photo1.jpg"
    printf '\xFF\xD8\xFF\xE0\x00\x10JFIF\x00\x01\x01\x00\x00\x01\x00\x01\x00\x00\xFF\xD9' > "$MEDIA_DIR/images/photo2.jpg"
    
    # Minimal PDF
    printf '%%PDF-1.4\n1 0 obj\n<< /Type /Catalog >>\nendobj\ntrailer\n<< /Root 1 0 R >>\n%%%%EOF' > "$MEDIA_DIR/documents/doc1.pdf"
    
    # VTT subtitle files
    cat > "$MEDIA_DIR/videos/movie1.vtt" << 'VTT'
WEBVTT

00:00:15.500 --> 00:00:20.000
Welcome to the movie

00:00:30.000 --> 00:00:35.000
This is an exciting scene

00:01:00.000 --> 00:01:05.000
The plot thickens
VTT

    cat > "$MEDIA_DIR/videos/movie2.vtt" << 'VTT'
WEBVTT

00:00:20.000 --> 00:00:25.000
Opening scene

00:00:45.000 --> 00:00:50.000
Main character appears
VTT

    cat > "$MEDIA_DIR/videos/clip1.vtt" << 'VTT'
WEBVTT

00:00:12.000 --> 00:00:15.000
Short clip caption
VTT

    cat > "$MEDIA_DIR/videos/clip2.vtt" << 'VTT'
WEBVTT

00:00:15.000 --> 00:00:18.000
Another short clip
VTT
fi

# Build disco binary if needed
if [ ! -f "$REPO_ROOT/disco" ]; then
    echo "Building disco binary..."
    cd "$REPO_ROOT" && go build -o disco ./cmd/disco
fi

# Create database with disco
echo "Creating database with disco..."
cd "$REPO_ROOT"
./disco add "$DB_FILE" "$MEDIA_DIR"

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
