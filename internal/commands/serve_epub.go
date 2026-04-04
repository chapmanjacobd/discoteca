package commands

import (
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/chapmanjacobd/discoteca/internal/utils"
)

// handleEpubConvert serves converted EPUB/text documents as HTML format
// URL format: /api/epub/{path} serves index.html with custom TOC header
// URL format: /api/epub/{path}/{asset} serves CSS/images from the HTML directory
func (c *ServeCmd) handleEpubConvert(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract path from URL using path value (captures {path...} from mux)
	pathParts := r.PathValue("path")
	if pathParts == "" {
		// Fallback for older Go versions or if path value is not set
		pathParts = strings.TrimPrefix(r.URL.Path, "/api/epub/")
	}

	// We ONLY unescape if it contains %, otherwise we might mess up valid paths
	if strings.Contains(pathParts, "%") {
		unescaped, err := unescapePath(pathParts)
		if err == nil {
			pathParts = unescaped
		}
	}
	slog.Info("handleEpubConvert request", "pathParts", pathParts)

	if pathParts == "" || pathParts == "/" {
		http.Error(w, "Path required", http.StatusBadRequest)
		return
	}

	// Split into document path and optional asset path
	// Find the part of the path that ends with a known ebook extension
	docPath := pathParts
	assetPath := ""

	ebookExts := []string{
		".epub",
		".mobi",
		".azw",
		".azw3",
		".fb2",
		".djvu",
		".cbz",
		".cbr",
		".docx",
		".odt",
		".rtf",
		".txt",
		".md",
		".html",
		".htm",
		".pdf",
	}

	for _, ext := range ebookExts {
		lowerParts := strings.ToLower(pathParts)
		extIdx := strings.Index(lowerParts, ext)
		if extIdx != -1 {
			// Found the extension. The document path ends here.
			endIdx := extIdx + len(ext)
			// Ensure it's either the end of the string or followed by a slash
			if endIdx == len(pathParts) || pathParts[endIdx] == '/' {
				docPath = pathParts[:endIdx]
				if endIdx < len(pathParts) {
					assetPath = strings.TrimPrefix(pathParts[endIdx:], "/")
				}
				break
			}
		}
	}

	// Ensure docPath starts with / for absolute paths
	if !strings.HasPrefix(docPath, "/") && !strings.HasPrefix(docPath, "C:") {
		docPath = "/" + docPath
	}

	slog.Info("Final resolved paths", "docPath", docPath, "assetPath", assetPath)

	// Verify file access
	if c.isPathBlocklisted(docPath) {
		http.Error(w, "Access denied: sensitive path", http.StatusForbidden)
		return
	}

	// Check if file exists
	fileInfo, err := os.Stat(docPath)
	if err != nil {
		// Check if it's a folder (might have trailing slash or not)
		folderPath := docPath
		if before, ok := strings.CutSuffix(folderPath, "/"); ok {
			folderPath = before
		}
		folderInfo, folderErr := os.Stat(folderPath)
		if folderErr != nil || !folderInfo.IsDir() {
			slog.Error("EPUB file not found on disk", "path", docPath, "error", err)
			http.Error(w, "File not found", http.StatusNotFound)
			return
		}
		// It's a folder, check if it has index.html
		docPath = folderPath
		fileInfo = folderInfo
	}

	// If it's a folder, check if it has index.html (HTML folder)
	if fileInfo.IsDir() {
		indexHTMLPath := filepath.Join(docPath, "index.html")
		if _, err := os.Stat(indexHTMLPath); err != nil {
			slog.Error("Folder does not contain index.html", "path", docPath, "error", err)
			http.Error(w, "Folder does not contain index.html", http.StatusNotFound)
			return
		}
		// Serve HTML folder directly without calibre conversion
		slog.Info("Serving HTML folder", "path", docPath)
		if assetPath != "" {
			// Serve asset from HTML folder
			assetFile := filepath.Join(docPath, assetPath)
			if !strings.HasPrefix(assetFile, docPath) {
				http.Error(w, "Invalid asset path", http.StatusBadRequest)
				return
			}
			serveFileWithMimeType(w, r, assetFile)
			return
		}
		// Serve wrapper HTML for the folder
		serveHTMLFolderWithTOC(w, r, docPath, docPath)
		return
	}

	// Skip calibre conversion for PDFs
	ext := strings.ToLower(filepath.Ext(docPath))
	if ext == ".pdf" {
		slog.Debug("Serving PDF directly", "path", docPath)
		http.ServeFile(w, r, docPath)
		return
	}

	// Convert EPUB/text to HTML
	slog.Info("Converting document to HTML", "path", docPath)
	htmlDir, err := utils.ConvertEpubToOEB(docPath)
	if err != nil {
		slog.Error("EPUB conversion failed", "path", docPath, "error", err)
		http.Error(w, fmt.Sprintf("Conversion failed: %v", err), http.StatusInternalServerError)
		return
	}

	slog.Debug("Conversion successful", "htmlDir", htmlDir)

	// If asset path specified, serve that file from HTML directory
	if assetPath != "" {
		assetFile := filepath.Join(htmlDir, assetPath)
		if !strings.HasPrefix(assetFile, htmlDir) {
			http.Error(w, "Invalid asset path", http.StatusBadRequest)
			return
		}
		slog.Debug("Serving asset", "assetFile", assetFile)
		serveFileWithMimeType(w, r, assetFile)
		return
	}

	// Serve wrapper HTML with TOC header
	slog.Info("Serving converted HTML with TOC", "htmlDir", htmlDir)
	serveHTMLWithTOC(w, r, htmlDir, docPath)
}

// serveHTMLWithTOC serves calibre HTML output with a sticky TOC header
func serveHTMLWithTOC(w http.ResponseWriter, _ *http.Request, htmlDir, originalPath string) {
	// Get list of HTML files for TOC
	htmlFiles := utils.GetHTMLFiles(htmlDir)

	// Find actual book content HTML for the main frame (relative to htmlDir)
	initialSrc := ""
	contentFile := utils.FindMainContentFile(htmlDir)
	if contentFile != "" {
		rel, err := filepath.Rel(htmlDir, contentFile)
		if err == nil {
			initialSrc = fmt.Sprintf("/api/epub/%s/%s",
				strings.TrimPrefix(originalPath, "/"),
				rel)
		}
	}

	// Create wrapper HTML with sticky TOC
	wrapperHTML := createWrapperHTML(initialSrc, htmlFiles, originalPath)

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write([]byte(wrapperHTML))
}

// createWrapperHTML creates HTML with sticky TOC header
func createWrapperHTML(initialSrc string, htmlFiles []string, originalPath string) string {
	// Extract title from originalPath or use filename
	title := filepath.Base(originalPath)

	// Build TOC options
	var tocOptions strings.Builder
	for i, file := range htmlFiles {
		baseName := filepath.Base(file)
		baseName = strings.TrimSuffix(baseName, filepath.Ext(baseName))
		// Clean up filename for display
		baseName = strings.ReplaceAll(baseName, "_", " ")
		baseName = strings.ReplaceAll(baseName, "-", " ")
		if baseName == "index" {
			baseName = "Start"
		}

		val := fmt.Sprintf("/api/epub/%s/%s",
			strings.TrimPrefix(originalPath, "/"),
			file)

		fmt.Fprintf(&tocOptions, `<option value="%s">%s</option>`, val, baseName)
		if i == 0 {
			tocOptions.WriteString("\n")
		}
	}

	wrapper := fmt.Sprintf(`<!DOCTYPE html>
<html>
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width, initial-scale=1.0">
<title>%s</title>
<style>
* { box-sizing: border-box; margin: 0; padding: 0; }
html, body { height: 100%%; overflow: hidden; }
.toc-header {
    position: fixed;
    top: 0;
    left: 0;
    right: 0;
    height: 50px;
    background: #2c3e50;
    color: white;
    display: flex;
    align-items: center;
    padding: 0 15px;
    z-index: 1000;
    box-shadow: 0 2px 5px rgba(0,0,0,0.2);
}
.toc-header h1 {
    font-size: 16px;
    font-weight: normal;
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
    margin-right: 15px;
}
.toc-nav {
    display: flex;
    align-items: center;
}
.toc-nav select {
    padding: 5px 10px;
    font-size: 14px;
    border: none;
    border-radius: 4px;
    background: #34495e;
    color: white;
    cursor: pointer;
    max-width: 300px;
}
.toc-nav select option {
    background: #2c3e50;
    color: white;
}
.content-frame {
    position: absolute;
    top: 50px;
    left: 0;
    right: 0;
    bottom: 0;
    border: none;
    width: 100%%;
    height: calc(100%% - 50px);
}
</style>
</head>
<body>
<div class="toc-header">
    <h1>%s</h1>
    <nav class="toc-nav">
        <select onchange="document.getElementById('content-frame').src = this.value">
            <option value="">Select chapter...</option>
            %s
        </select>
    </nav>
</div>
<iframe id="content-frame" name="content-frame" class="content-frame" src="%s"></iframe>
</body>
</html>`, title, title, tocOptions.String(), initialSrc)

	return wrapper
}

func unescapePath(path string) (string, error) {
	return url.PathUnescape(path)
}

// serveHTMLFolderWithTOC serves an HTML folder with a sticky TOC header
func serveHTMLFolderWithTOC(w http.ResponseWriter, _ *http.Request, htmlDir, originalPath string) {
	// Get list of HTML files for TOC
	htmlFiles := utils.GetHTMLFiles(htmlDir)

	// Find actual book content HTML for the main frame (relative to htmlDir)
	initialSrc := ""
	contentFile := utils.FindMainContentFile(htmlDir)
	if contentFile != "" {
		rel, err := filepath.Rel(htmlDir, contentFile)
		if err == nil {
			initialSrc = fmt.Sprintf("/api/epub/%s/%s",
				strings.TrimPrefix(originalPath, "/"),
				rel)
		}
	}

	// Create wrapper HTML with sticky TOC
	wrapperHTML := createWrapperHTML(initialSrc, htmlFiles, originalPath)

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write([]byte(wrapperHTML))
}
