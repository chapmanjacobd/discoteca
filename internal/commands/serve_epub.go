package commands

import (
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/chapmanjacobd/discoteca/internal/models"
	"github.com/chapmanjacobd/discoteca/internal/utils"
)

// HandleEpubConvert serves converted EPUB/text documents as HTML format
// URL format: /api/epub/{path} serves index.html with custom TOC header
// URL format: /api/epub/{path}/{asset} serves CSS/images from the HTML directory
func (c *ServeCmd) HandleEpubConvert(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	pathParts := c.extractEpubPath(w, r)
	if pathParts == "" {
		return // error already written
	}
	models.Log.Info("handleEpubConvert request", "pathParts", pathParts)

	docPath, assetPath := c.resolveDocAndAssetPath(pathParts)
	models.Log.Info("Final resolved paths", "docPath", docPath, "assetPath", assetPath)

	if c.isPathBlocklisted(docPath) {
		http.Error(w, "Access denied: sensitive path", http.StatusForbidden)
		return
	}

	fileInfo, err := os.Stat(docPath)
	if err != nil {
		folderPath := c.normalizeFolderPath(docPath)
		if folderPath == "" {
			models.Log.Error("EPUB file not found on disk", "path", docPath, "error", err)
			http.Error(w, "File not found", http.StatusNotFound)
			return
		}
		docPath = folderPath
		indexHTMLPath := filepath.Join(docPath, "index.html")
		if _, err2 := os.Stat(indexHTMLPath); err2 != nil {
			models.Log.Error("Folder does not contain index.html", "path", docPath, "error", err2)
			http.Error(w, "Folder does not contain index.html", http.StatusNotFound)
			return
		}
		models.Log.Info("Serving HTML folder", "path", docPath)
		c.serveHTMLFolder(w, r, docPath, assetPath)
		return
	}

	if fileInfo.IsDir() {
		indexHTMLPath := filepath.Join(docPath, "index.html")
		if _, err2 := os.Stat(indexHTMLPath); err2 != nil {
			models.Log.Error("Folder does not contain index.html", "path", docPath, "error", err2)
			http.Error(w, "Folder does not contain index.html", http.StatusNotFound)
			return
		}
		models.Log.Info("Serving HTML folder", "path", docPath)
		c.serveHTMLFolder(w, r, docPath, assetPath)
		return
	}

	ext := strings.ToLower(filepath.Ext(docPath))
	if ext == ".pdf" {
		models.Log.Debug("Serving PDF directly", "path", docPath)
		http.ServeFile(w, r, docPath)
		return
	}

	c.serveConvertedDocument(w, r, docPath, assetPath)
}

// extractEpubPath extracts and validates the path from the request URL.
// Returns empty string if path is missing (error already written to response).
func (c *ServeCmd) extractEpubPath(w http.ResponseWriter, r *http.Request) string {
	pathParts := r.PathValue("path")
	if pathParts == "" {
		pathParts = strings.TrimPrefix(r.URL.Path, "/api/epub/")
	}

	if strings.Contains(pathParts, "%") {
		unescaped, err := unescapePath(pathParts)
		if err == nil {
			pathParts = unescaped
		}
	}

	if pathParts == "" || pathParts == "/" {
		http.Error(w, "Path required", http.StatusBadRequest)
		return ""
	}

	return pathParts
}

// resolveDocAndAssetPath splits the URL path into document path and optional asset path
// based on known ebook file extensions.
func (c *ServeCmd) resolveDocAndAssetPath(pathParts string) (docPath, assetPath string) {
	docPath = pathParts

	ebookExts := []string{
		".epub", ".mobi", ".azw", ".azw3", ".fb2", ".djvu",
		".cbz", ".cbr", ".docx", ".odt", ".rtf", ".txt",
		".md", ".html", ".htm", ".pdf", ".xhtml", ".opf", ".ncx",
		".oeb", ".oebps", ".xml",
	}

	for _, ext := range ebookExts {
		lowerParts := strings.ToLower(pathParts)
		extIdx := strings.Index(lowerParts, ext)
		if extIdx != -1 {
			endIdx := extIdx + len(ext)
			if endIdx == len(pathParts) || pathParts[endIdx] == '/' {
				docPath = pathParts[:endIdx]
				if endIdx < len(pathParts) {
					assetPath = strings.TrimPrefix(pathParts[endIdx:], "/")
				}
				break
			}
		}
	}

	if !strings.HasPrefix(docPath, "/") && !strings.HasPrefix(docPath, "C:") {
		docPath = "/" + docPath
	}

	return docPath, assetPath
}

// normalizeFolderPath removes a trailing slash from docPath if it exists and the path is a valid directory.
// Returns the folder path if valid, empty string otherwise.
func (c *ServeCmd) normalizeFolderPath(docPath string) string {
	folderPath, _ := strings.CutSuffix(docPath, "/")
	folderInfo, err := os.Stat(folderPath)
	if err != nil || !folderInfo.IsDir() {
		return ""
	}
	return folderPath
}

// serveHTMLFolder serves an HTML folder directly, with optional asset or TOC wrapper.
func (c *ServeCmd) serveHTMLFolder(w http.ResponseWriter, r *http.Request, docPath, assetPath string) {
	if assetPath != "" {
		assetFile := filepath.Join(docPath, assetPath)
		if !strings.HasPrefix(assetFile, docPath) {
			http.Error(w, "Invalid asset path", http.StatusBadRequest)
			return
		}
		serveFileWithMimeType(w, r, assetFile)
		return
	}
	serveHTMLWithTOC(w, r, docPath, docPath)
}

// serveConvertedDocument converts the document via calibre and serves the result.
func (c *ServeCmd) serveConvertedDocument(w http.ResponseWriter, r *http.Request, docPath, assetPath string) {
	models.Log.Info("Converting document to HTML", "path", docPath)
	htmlDir, err := utils.ConvertEpubToOEB(r.Context(), docPath)
	if err != nil {
		models.Log.Error("EPUB conversion failed", "path", docPath, "error", err)
		http.Error(w, fmt.Sprintf("Conversion failed: %v", err), http.StatusInternalServerError)
		return
	}
	models.Log.Debug("Conversion successful", "htmlDir", htmlDir)

	if assetPath != "" {
		assetFile := filepath.Join(htmlDir, assetPath)
		if !strings.HasPrefix(assetFile, htmlDir) {
			http.Error(w, "Invalid asset path", http.StatusBadRequest)
			return
		}
		models.Log.Debug("Serving asset", "assetFile", assetFile)
		serveFileWithMimeType(w, r, assetFile)
		return
	}

	models.Log.Info("Serving converted HTML with TOC", "htmlDir", htmlDir)
	serveHTMLWithTOC(w, r, htmlDir, docPath)
}

// serveHTMLWithTOC serves calibre HTML output with a sticky TOC header
func serveHTMLWithTOC(w http.ResponseWriter, _ *http.Request, htmlDir, originalPath string) {
	// Get list of HTML files for TOC
	htmlFiles, err := utils.GetHTMLFiles(htmlDir)
	if err != nil {
		http.Error(w, "Failed to get HTML files", http.StatusInternalServerError)
		return
	}

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
	_, _ = w.Write([]byte(wrapperHTML))
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
