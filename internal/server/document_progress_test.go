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

type documentProgressTestEnv struct {
	fixture  *testutils.TestFixture
	bookPath string
	server   *server.Server
}

func newDocumentProgressTestEnv(t *testing.T) *documentProgressTestEnv {
	t.Helper()

	fixture := testutils.Setup(t)
	t.Cleanup(fixture.Cleanup)

	bookPath := fixture.CreateDummyFile("book.epub")
	addCmd := &commands.AddCmd{Args: []string{fixture.DBPath, bookPath}}
	addCmd.AfterApply()
	if err := addCmd.Run(context.Background()); err != nil {
		t.Fatalf("add failed: %v", err)
	}

	srv := &server.Server{Databases: []string{fixture.DBPath}}
	t.Cleanup(func() {
		if err := srv.Close(); err != nil {
			t.Errorf("close failed: %v", err)
		}
	})

	return &documentProgressTestEnv{
		fixture:  fixture,
		bookPath: bookPath,
		server:   srv,
	}
}

func (e *documentProgressTestEnv) request(
	t *testing.T,
	method, target string,
	body any,
	readOnly bool,
) *httptest.ResponseRecorder {
	t.Helper()

	e.server.ReadOnly = readOnly
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
	e.server.HandleDocumentProgress(w, req)
	return w
}

func TestServer_DocumentProgress(t *testing.T) {
	env := newDocumentProgressTestEnv(t)

	t.Run("missing progress returns zero position", func(t *testing.T) {
		checkDocumentProgressDefault(t, env)
	})

	t.Run("post stores progress and get returns it", func(t *testing.T) {
		checkDocumentProgressRoundTrip(t, env)
	})

	t.Run("scroll percent is clamped", func(t *testing.T) {
		checkDocumentProgressClamping(t, env)
	})

	t.Run("unknown path is rejected", func(t *testing.T) {
		checkDocumentProgressUnknownPath(t, env)
	})

	t.Run("read-only mode rejects writes but allows reads", func(t *testing.T) {
		checkDocumentProgressReadOnly(t, env)
	})

	t.Run("unsupported method is rejected", func(t *testing.T) {
		checkDocumentProgressUnsupportedMethod(t, env)
	})
}

func checkDocumentProgressDefault(t *testing.T, env *documentProgressTestEnv) {
	t.Helper()

	w := env.request(t, http.MethodGet, "/api/document-progress?path="+env.bookPath, nil, false)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200: %s", w.Code, w.Body.String())
	}
	var got models.DocumentProgress
	if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode failed: %v", err)
	}
	if got.Path != env.bookPath || got.ScrollPercent != 0 || got.TimeUpdated != 0 {
		t.Errorf("unexpected default progress: %+v", got)
	}
}

func checkDocumentProgressRoundTrip(t *testing.T, env *documentProgressTestEnv) {
	t.Helper()

	w := env.request(t, http.MethodPost, "/api/document-progress", map[string]any{
		"path":           env.bookPath,
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

	w = env.request(t, http.MethodGet, "/api/document-progress?path="+env.bookPath, nil, false)
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

	dbConn := env.fixture.GetDB()
	defer dbConn.Close()
	var stored float64
	var chapter sql.NullString
	if err := dbConn.QueryRow(
		"SELECT scroll_percent, chapter FROM document_progress WHERE media_path = ?",
		env.bookPath,
	).Scan(&stored, &chapter); err != nil {
		t.Fatalf("query failed: %v", err)
	}
	if stored != 0.42 || chapter.String != "chapter-2.xhtml" {
		t.Errorf("stored = %v/%q, want 0.42/chapter-2.xhtml", stored, chapter.String)
	}
}

func checkDocumentProgressClamping(t *testing.T, env *documentProgressTestEnv) {
	t.Helper()

	w := env.request(t, http.MethodPost, "/api/document-progress", map[string]any{
		"path":           env.bookPath,
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

	w = env.request(t, http.MethodPost, "/api/document-progress", map[string]any{
		"path":           env.bookPath,
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
}

func checkDocumentProgressUnknownPath(t *testing.T, env *documentProgressTestEnv) {
	t.Helper()

	w := env.request(t, http.MethodGet, "/api/document-progress?path=/missing.epub", nil, false)
	if w.Code != http.StatusNotFound {
		t.Errorf("GET status = %d, want 404", w.Code)
	}
	w = env.request(t, http.MethodPost, "/api/document-progress", map[string]any{
		"path":           "/missing.epub",
		"scroll_percent": 0.5,
	}, false)
	if w.Code != http.StatusNotFound {
		t.Errorf("POST status = %d, want 404", w.Code)
	}
}

func checkDocumentProgressReadOnly(t *testing.T, env *documentProgressTestEnv) {
	t.Helper()

	w := env.request(t, http.MethodPost, "/api/document-progress", map[string]any{
		"path":           env.bookPath,
		"scroll_percent": 0.9,
	}, true)
	if w.Code != http.StatusForbidden {
		t.Errorf("POST status = %d, want 403", w.Code)
	}
	w = env.request(t, http.MethodGet, "/api/document-progress?path="+env.bookPath, nil, true)
	if w.Code != http.StatusOK {
		t.Errorf("GET status = %d, want 200", w.Code)
	}
}

func checkDocumentProgressUnsupportedMethod(t *testing.T, env *documentProgressTestEnv) {
	t.Helper()

	w := env.request(t, http.MethodPut, "/api/document-progress", nil, false)
	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("status = %d, want 405", w.Code)
	}
}

func TestServer_DocumentProgress_ChapterIsBounded(t *testing.T) {
	env := newDocumentProgressTestEnv(t)

	const maxChapterLength = 2048
	chapter := strings.Repeat("x", maxChapterLength+50)
	w := env.request(t, http.MethodPost, "/api/document-progress", map[string]any{
		"path":           env.bookPath,
		"scroll_percent": 0.1,
		"chapter":        chapter,
	}, false)
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
