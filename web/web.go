package web

import (
	"embed"
	"io/fs"
)

// FsRaw embeds the static web assets from the dist folder
//
//go:embed dist/*
var FsRaw embed.FS

// FS is the web asset file system with "dist" prefix removed
var FS, _ = fs.Sub(FsRaw, "dist")
