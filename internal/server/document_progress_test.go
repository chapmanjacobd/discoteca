package server_test

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/chapmanjacobd/discoteca/internal/commands"
	"github.com/chapmanjacobd/discoteca/internal/models"
	"github.com/chapmanjacobd/discoteca/internal/server"
	"github.com/chapmanjacobd/discoteca/internal/testutils"
)

func TestServer_DocumentProgress(t *testing.T) {
	fixture := testutils.Setup(t)
	defer fixture.Cleanup()

	bookPath := fixture.CreateDummyFile("book.epub")
	addCmd := &commands.AddCmd{Args: []string{fixture.DBPath, bookPath}}
	addCmd.AfterApply()
	if err := addCmd.Run(context.Background()); err != nil {
		t.Fatalf("add failed: %v", err)
	}

	cmd := &server.Server{Databases: []string{fixture.DBPath}}
	defer cmd.Close()

	request := func(method, target string, body any, readOnly bool) *httptest.ResponseRecorder {
		t.Helper()
		cmd.ReadOnly = readOnly
		var reader *bytes.Reader
		if body == nil {
			reader = bytes.NewReader(nil)
		} else {
			payload, err := json.Marshal(body)
			if err != nil {
				t.Fatalf("json.Marshal failed: %v", err)
			}
			reader = bytes.NewReader(payload)
		}
		req := httptest.NewRequest(method, target, reader)
		w := httptest.NewRecorder()
		cmd.HandleDocumentProgress(w, req)
		return w
	}

	t.Run("missing progress returns zero position", func(t *testing.T) {
		w := request(http.MethodGet, "/api/document-progress?path="+bookPath, nil, false)
		if w.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200: %s", w.Code, w.Body.String())
		}
		var got models.DocumentProgress
		if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
			t.Fatalf("decode failed: %v", err)
		}
		if got.Path != bookPath || got.ScrollPercent != 0 || got.TimeUpdated != 0 {
			t.Errorf("unexpected default progress: %+v", got)
		}
	})

	t.Run("post stores progress and get returns it", func(t *testing.T) {
		w := request(http.MethodPost, "/api/document-progress", map[string]any{
			"path":           bookPath,
			"scroll_percent": 0.42,
			"chapter":        "chapter-2.xhtml",
		}, false)
		if w.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200: %s", w.Code, w.Body.String())
		}
		var posted models.DocumentProgress
		if err := json.Unmarshal(w.Body.Bytes(), &posted); err != nil {
			t.Fatalf("decode failed: %v", err)
		}
		if posted.ScrollPercent != 0.42 || posted.Chapter != "chapter-2.xhtml" || posted.TimeUpdated == 0 {
			t.Errorf("unexpected post response: %+v", posted)
		}

		w = request(http.MethodGet, "/api/document-progress?path="+bookPath, nil, false)
		if w.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200: %s", w.Code, w.Body.String())
		}
		var got models.DocumentProgress
		if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
			t.Fatalf("decode failed: %v", err)
		}
		if got.ScrollPercent != 0.42 || got.Chapter != "chapter-2.xhtml" || got.TimeUpdated != posted.TimeUpdated {
			t.Errorf("get = %+v, want posted %+v", got, posted)
		}

		dbConn := fixture.GetDB()
		defer dbConn.Close()
		var stored float64
		var chapter sql.NullString
		if err := dbConn.QueryRow(
			"SELECT scroll_percent, chapter FROM document_progress WHERE media_path = ?",
			bookPath,
		).Scan(&stored, &chapter); err != nil {
			t.Fatalf("query failed: %v", err)
		}
		if stored != 0.42 || chapter.String != "chapter-2.xhtml" {
			t.Errorf("stored = %v/%q, want 0.42/chapter-2.xhtml", stored, chapter.String)
		}
	})

	t.Run("scroll percent is clamped", func(t *testing.T) {
		w := request(http.MethodPost, "/api/document-progress", map[string]any{
			"path":           bookPath,
			"scroll_percent": 1.7,
		}, false)
		if w.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200: %s", w.Code, w.Body.String())
		}
		var got models.DocumentProgress
		if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
			t.Fatalf("decode failed: %v", err)
		}
		if got.ScrollPercent != 1 {
			t.Errorf("scroll_percent = %v, want 1", got.ScrollPercent)
		}

		w = request(http.MethodPost, "/api/document-progress", map[string]any{
			"path":           bookPath,
			"scroll_percent": -2,
		}, false)
		if w.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200: %s", w.Code, w.Body.String())
		}
		if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
			t.Fatalf("decode failed: %v", err)
		}
		if got.ScrollPercent != 0 {
			t.Errorf("scroll_percent = %v, want 0", got.ScrollPercent)
		}
	})

	t.Run("unknown path is rejected", func(t *testing.T) {
		w := request(http.MethodGet, "/api/document-progress?path=/missing.epub", nil, false)
		if w.Code != http.StatusNotFound {
			t.Errorf("GET status = %d, want 404", w.Code)
		}
		w = request(http.MethodPost, "/api/document-progress", map[string]any{
			"path":           "/missing.epub",
			"scroll_percent": 0.5,
		}, false)
		if w.Code != http.StatusNotFound {
			t.Errorf("POST status = %d, want 404", w.Code)
		}
	})

	t.Run("read-only mode rejects writes but allows reads", func(t *testing.T) {
		w := request(http.MethodPost, "/api/document-progress", map[string]any{
			"path":           bookPath,
			"scroll_percent": 0.9,
		}, true)
		if w.Code != http.StatusForbidden {
			t.Errorf("POST status = %d, want 403", w.Code)
		}
		w = request(http.MethodGet, "/api/document-progress?path="+bookPath, nil, true)
		if w.Code != http.StatusOK {
			t.Errorf("GET status = %d, want 200", w.Code)
		}
	})

	t.Run("unsupported method is rejected", func(t *testing.T) {
		w := request(http.MethodPut, "/api/document-progress", nil, false)
		if w.Code != http.StatusMethodNotAllowed {
			t.Errorf("status = %d, want 405", w.Code)
		}
	})
}

func TestServer_DocumentProgress_ChapterIsBounded(t *testing.T) {
	fixture := testutils.Setup(t)
	defer fixture.Cleanup()

	bookPath := fixture.CreateDummyFile("book.epub")
	addCmd := &commands.AddCmd{Args: []string{fixture.DBPath, bookPath}}
	addCmd.AfterApply()
	if err := addCmd.Run(context.Background()); err != nil {
		t.Fatalf("add failed: %v", err)
	}

	cmd := &server.Server{Databases: []string{fixture.DBPath}}
	defer cmd.Close()

	const maxChapterLength = 2048
	chapter := strings.Repeat("x", maxChapterLength+50)
	payload, err := json.Marshal(map[string]any{
		"path":           bookPath,
		"scroll_percent": 0.1,
		"chapter":        chapter,
	})
	if err != nil {
		t.Fatalf("json.Marshal failed: %v", err)
	}
	req := httptest.NewRequest(http.MethodPost, "/api/document-progress", bytes.NewReader(payload))
	w := httptest.NewRecorder()
	cmd.HandleDocumentProgress(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200: %s", w.Code, w.Body.String())
	}

	var got models.DocumentProgress
	if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode failed: %v", err)
	}
	if len(got.Chapter) != maxChapterLength {
		t.Errorf("chapter length = %d, want %d", len(got.Chapter), maxChapterLength)
	}
}
