# Installation Guide

This guide covers installing discoteca on Linux, macOS, and Windows, including all optional dependencies for full functionality.

## Quick Install

### Pre-built Binaries (Recommended)

Download the latest release for your platform from [GitHub Releases](https://github.com/chapmanjacobd/discoteca/releases):

- **Linux**: `disco-linux-amd64` or `disco-linux-arm64`
- **macOS**: `disco-darwin-amd64` or `disco-darwin-arm64` (Apple Silicon)
- **Windows**: `disco-windows-amd64.zip`

Extract and add to your PATH, or run directly.

### Go Install

```bash
go install github.com/chapmanjacobd/discoteca/cmd/disco@latest
```

**Note**: This requires CGO and appropriate compilers for your platform.

---

## Dependencies

Discoteca works without any dependencies for basic functionality. The following optional dependencies enable additional features:

### Required for Core Features

| Dependency | Purpose | Required For |
|------------|---------|--------------|
| **SQLite3** | Database engine | All operations (CGO dependency) |
| **Go 1.26+** | Build toolchain | Building from source |

### Optional Dependencies

#### Media Processing (Recommended)

| Dependency | Purpose | Features Enabled |
|------------|---------|------------------|
| **ffmpeg** | Media transcoding, duration detection, HLS streaming | Video/audio playback, streaming, media inspection |
| **ffprobe** | Media metadata extraction | Accurate duration, codec info, stream detection |

**Install on Linux:**
```bash
# Debian/Ubuntu
sudo apt install ffmpeg

# Fedora/RHEL
sudo dnf install ffmpeg

# Arch
sudo pacman -S ffmpeg
```

**Install on macOS:**
```bash
brew install ffmpeg
```

**Install on Windows:**

**Option 1: Using Scoop (Recommended)**
```powershell
# Install Scoop if not already installed
Set-ExecutionPolicy -ExecutionPolicy RemoteSigned -Scope CurrentUser
Invoke-RestMethod -Uri https://get.scoop.sh | Invoke-Expression

# Install ffmpeg
scoop install ffmpeg
```

**Option 2: Manual Installation**
1. Download from [ffmpeg.org](https://ffmpeg.org/download.html)
2. Extract to `C:\ffmpeg`
3. Add `C:\ffmpeg\bin` to your PATH environment variable
4. Verify: `ffmpeg -version`

#### Text & Ebook Processing

| Dependency | Purpose | Features Enabled |
|------------|---------|------------------|
| **pandoc** | Document conversion | Text file processing (dev/test only) |
| **calibre** | Ebook conversion (EPUB, MOBI, etc.) | Ebook library management, TTS generation |

**Install on Linux:**
```bash
# Debian/Ubuntu
sudo apt install pandoc calibre

# Fedora/RHEL
sudo dnf install pandoc calibre

# Arch
sudo pacman -S pandoc calibre
```

**Install on macOS:**
```bash
brew install pandoc calibre
```

**Install on Windows:**

**Option 1: Using Scoop (Recommended)**
```powershell
scoop install pandoc
```

**Option 2: Manual Installation**
- **pandoc**: Download from [pandoc.org](https://pandoc.org/installing.html)
- **calibre**: Download from [calibre-ebook.com](https://calibre-ebook.com/download_windows)

#### Text-to-Speech

| Dependency | Purpose | Features Enabled |
|------------|---------|------------------|
| **espeak-ng** | Audio generation from text | TTS caption generation |

**Install on Linux:**
```bash
# Debian/Ubuntu
sudo apt install espeak-ng

# Fedora/RHEL
sudo dnf install espeak-ng

# Arch
sudo pacman -S espeak-ng
```

**Install on macOS:**
```bash
brew install espeak-ng
```

**Install on Windows:**

**Option 1: Using Scoop (Recommended)**
```powershell
scoop install espeak-ng
```

**Option 2: Manual Installation**
1. Download from [espeak-ng.github.io](https://espeak-ng.github.io/)
2. Run installer
3. Add to PATH if needed

#### Media Players

| Dependency | Purpose | Features Enabled |
|------------|---------|------------------|
| **mpv** | Video/audio playback control | `disco play`, `disco now`, playback control |

**Install on Linux:**
```bash
sudo apt install mpv    # Debian/Ubuntu
sudo dnf install mpv    # Fedora/RHEL
sudo pacman -S mpv      # Arch
```

**Install on macOS:**
```bash
brew install mpv
```

**Install on Windows:**

**Option 1: Using Scoop (Recommended)**
```powershell
scoop install mpv
```

**Option 2: Manual Installation**
1. Download from [mpv.io](https://mpv.io/installation/)
2. Add to PATH for CLI integration

---

## Platform-Specific Notes

### Linux

#### Desktop Integration

The `disco open` command uses `xdg-open` by default, which is available on most desktop Linux distributions.

```bash
# Install xdg-utils if not present
sudo apt install xdg-utils    # Debian/Ubuntu
sudo dnf install xdg-utils    # Fedora/RHEL
```

#### Server/Headless Installation

For server deployments without GUI:

```bash
# Minimal installation
sudo apt install ffmpeg sqlite3

# Optional: for ebook support
sudo apt install calibre

# Optional: for TTS
sudo apt install espeak-ng
```

### Windows

#### PowerShell vs Command Prompt

Both work, but note:
- Use `disco.exe` explicitly in PowerShell if there are naming conflicts
- Paths with spaces should be quoted: `disco add my.db "C:\My Videos"`

#### Windows Defender

If Windows Defender blocks the binary:
1. Right-click the executable → Properties
2. Check "Unblock" at the bottom
3. Click OK

#### Environment Variables

Add disco to your PATH:
1. Open System Properties → Advanced → Environment Variables
2. Add the disco installation directory to `Path`
3. Restart your terminal

#### Windows Subsystem for Linux (WSL)

You can also run discoteca in WSL2:
```bash
# Install in WSL2
sudo apt install ffmpeg sqlite3
go install github.com/chapmanjacobd/discoteca/cmd/disco@latest

# Access Windows files from WSL
disco add my.db /mnt/c/Users/YourName/Videos
```

### macOS

#### Apple Silicon (M1/M2/M3)

Discoteca provides native `darwin_arm64` builds. All dependencies work natively:

```bash
# Install Homebrew if needed
/bin/bash -c "$(curl -fsSL https://raw.githubusercontent.com/Homebrew/install/HEAD/install.sh)"

# Install dependencies
brew install ffmpeg calibre espeak-ng mpv
```

#### Gatekeeper

If macOS prevents running the binary:

```bash
# Remove quarantine attribute
xattr -d com.apple.quarantine /path/to/disco
```

Or right-click the app and select "Open" once to approve.

---

## Build from Source

### Prerequisites

**All Platforms:**
- Go 1.26 or later
- Git
- CGO compiler (varies by platform)

**Linux:**
```bash
sudo apt install build-essential
```

**macOS:**
```bash
xcode-select --install
```

**Windows:**
1. Install [MinGW-w64](https://www.mingw-w64.org/) or use WSL
2. Or install Visual Studio Build Tools with C++ support

### Standard Build

```bash
git clone https://github.com/chapmanjacobd/discoteca.git
cd discoteca
go mod tidy
go build -tags "fts5" -o disco ./cmd/disco
```

### Build Variants

See [BUILD_MODES.md](BUILD_MODES.md) for different build options:

```bash
# FTS5 support (default, recommended)
make build-fts5

# Bleve full-text search
make build-bleve

# No full-text search (minimal)
make build-nofts
```

### Cross-Compilation

Discoteca uses GoReleaser for cross-platform builds:

```bash
# Install GoReleaser
go install github.com/goreleaser/goreleaser@latest

# Build all platforms
goreleaser release --snapshot --clean
```

Or manually:

```bash
# Linux
GOOS=linux GOARCH=amd64 go build -tags "fts5" -o disco-linux-amd64 ./cmd/disco

# macOS
GOOS=darwin GOARCH=arm64 go build -tags "fts5" -o disco-darwin-arm64 ./cmd/disco

# Windows
GOOS=windows GOARCH=amd64 go build -tags "fts5" -o disco-windows-amd64.exe ./cmd/disco
```

---

## Verifying Installation

### Check Version

```bash
disco --version
```

### Test Core Functionality

```bash
# Create a test database
disco add test.db /path/to/media

# List media
disco print test.db
```

### Test Optional Features

```bash
# Test ffmpeg integration (should show version)
ffmpeg -version

# Test calibre (should convert a test file)
ebook-convert --version

# Test espeak-ng
espeak-ng "Hello" --stdout > /dev/null && echo "espeak-ng works"
```

---

## Troubleshooting

### "sqlite3: unable to open database file"

Ensure SQLite3 is installed and accessible:
```bash
# Linux
sudo apt install libsqlite3-dev

# macOS (usually pre-installed)
sqlite3 --version

# Windows
# SQLite is embedded in the binary via CGO
```

### "ffmpeg not found"

Add ffmpeg to your PATH or install it (see dependencies above).

### "permission denied" when opening files

**Linux/macOS:**
```bash
chmod +x /path/to/disco
```

**Windows:**
Run as Administrator or check file properties.

### CGO Build Errors

**Linux:**
```bash
sudo apt install gcc libc6-dev
```

**macOS:**
```bash
xcode-select --install
```

**Windows:**
Use WSL or install MinGW-w64.

### Cross-Platform Path Issues

Discoteca handles paths correctly on all platforms, but be aware:
- Use forward slashes `/` or escaped backslashes `\\` in paths
- Quote paths with spaces
- UNC paths on Windows: `\\server\share\file.mp4`

---

## Docker Installation

Coming soon. See GitHub issues for containerization progress.

---

## Package Managers

### Homebrew (macOS/Linux)

Not yet available. Contributions welcome!

### Chocolatey (Windows)

Not yet available. Contributions welcome!

### AUR (Arch Linux)

Not yet available. Contributions welcome!

---

## Updating

### Pre-built Binaries

Download the latest release and replace the old binary.

### Go Install

```bash
go install github.com/chapmanjacobd/discoteca/cmd/disco@latest
```

### Built-in Update Command

```bash
disco update
```

This checks GitHub for the latest release and updates automatically.

---

## Uninstalling

### Pre-built Binary

Simply delete the binary file.

### Go Install

```bash
rm $(go env GOPATH)/bin/disco
```

### Remove Configuration

```bash
# Linux/macOS
rm -rf ~/.config/disco
rm -rf ~/.cache/disco

# Windows
rmdir %APPDATA%\disco
rmdir %LOCALAPPDATA%\disco
```

---

## Next Steps

- See [README.md](README.md) for usage examples
- See [BUILD_MODES.md](BUILD_MODES.md) for build configuration options
- Run `disco --help` for command reference
