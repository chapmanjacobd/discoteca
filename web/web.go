package web

import (
	"embed"
	"io/fs"
)

// FS_RAW embeds the static web assets from the dist folder
//
//go:embed dist/*
var FS_RAW embed.FS

// FS is the web asset file system with "dist" prefix removed
var FS, _ = fs.Sub(FS_RAW, "dist")
