package server_test

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"

	_ "github.com/mattn/go-sqlite3"

	"github.com/chapmanjacobd/discoteca/internal/db"
	"github.com/chapmanjacobd/discoteca/internal/logging"
	"github.com/chapmanjacobd/discoteca/internal/server"
)

func setupPhase7DB(t *testing.T) (string, *sql.DB) {
	t.Helper()
	logging.SetupLogging(0)
	dbPath := filepath.Join(t.TempDir(), "security-integration.db")
	sqlDB, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := db.InitDB(context.Background(), sqlDB); err != nil {
		t.Fatalf("failed to initialize DB: %v", err)
	}
	return dbPath, sqlDB
}

func securityTestServer(dbPath string) *server.Server {
	return &server.Server{Databases: []string{dbPath}, APIToken: "test-token"}
}

func authRequest(method, target string, body *bytes.Reader, token string) *http.Request {
	var req *http.Request
	if body == nil {
		req = httptest.NewRequest(method, target, nil)
	} else {
		req = httptest.NewRequest(method, target, body)
	}
	req.Header.Set("X-Disco-Token", token)
	return req
}

func TestSessionTokenIsNotExposedByStaticAssets(t *testing.T) {
	cmd := &server.Server{APIToken: "super-secret-token"}
	mux := cmd.Mux()

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	if strings.Contains(w.Body.String(), "super-secret-token") {
		t.Fatal("static HTML must not contain the API token")
	}
	for _, cookie := range w.Result().Cookies() {
		if cookie.Name == "disco_token" {
			t.Fatal("GET / must not set the API token cookie")
		}
	}

	js := httptest.NewRecorder()
	mux.ServeHTTP(js, httptest.NewRequest(http.MethodGet, "/index.js", nil))
	if js.Code != http.StatusOK {
		t.Fatalf("expected JS asset 200, got %d", js.Code)
	}
	if strings.Contains(js.Body.String(), "super-secret-token") {
		t.Fatal("static JavaScript must not contain the API token")
	}
}

func TestSessionEndpoint(t *testing.T) {
	cmd := &server.Server{APIToken: "session-token"}
	mux := cmd.Mux()

	t.Run("wrong token returns 401", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/session", nil)
		req.Header.Set("Authorization", "Bearer wrong")
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)
		if w.Code != http.StatusUnauthorized {
			t.Fatalf("expected 401, got %d", w.Code)
		}
		if len(w.Result().Cookies()) != 0 {
			t.Fatal("failed session must not set a cookie")
		}
	})

	t.Run("right token sets secure HttpOnly cookie", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/session", nil)
		req.Header.Set("Authorization", "Bearer session-token")
		req.Header.Set("X-Forwarded-Proto", "https")
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)
		if w.Code != http.StatusNoContent {
			t.Fatalf("expected 204, got %d", w.Code)
		}
		cookies := w.Result().Cookies()
		if len(cookies) != 1 {
			t.Fatalf("expected one cookie, got %d", len(cookies))
		}
		cookie := cookies[0]
		if cookie.Name != "disco_token" || cookie.Value != "session-token" {
			t.Fatalf("unexpected session cookie %#v", cookie)
		}
		if !cookie.HttpOnly || !cookie.Secure || cookie.SameSite != http.SameSiteStrictMode {
			t.Fatalf("session cookie is missing security flags: %#v", cookie)
		}
	})

	t.Run("cookie authenticates API requests", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/databases", nil)
		req.AddCookie(&http.Cookie{Name: "disco_token", Value: "session-token"})
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", w.Code)
		}
	})

	t.Run("bearer authenticates API requests", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/databases", nil)
		req.Header.Set("Authorization", "Bearer session-token")
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", w.Code)
		}
	})
}

func TestQueryRejectsUnsafeScoreAndSort(t *testing.T) {
	dbPath, sqlDB := setupPhase7DB(t)
	defer sqlDB.Close()
	if _, err := sqlDB.Exec(
		`INSERT INTO media (path, title, media_type, time_deleted) VALUES ('/media/a.mp4', 'A', 'video', 0)`,
	); err != nil {
		t.Fatal(err)
	}

	cmd := securityTestServer(dbPath)
	defer cmd.Close()
	mux := cmd.Mux()

	for _, target := range []string{
		"/api/query?min_score=1;DROP%20TABLE%20media",
		"/api/query?max_score=1%20OR%201=1",
		"/api/query?min_score=NaN",
		"/api/query?max_score=Inf",
		"/api/query?sort=1;DROP%20TABLE%20media",
		"/api/query?sort=(SELECT%201)",
		"/api/query?sort_fields=[%22path%22,%221;DROP%20TABLE%20media%22]",
		"/api/query?sort_fields=path%20desc,1;DROP%20TABLE%20media",
		"/api/query?sort_desc=1;DROP%20TABLE%20media",
	} {
		t.Run(target, func(t *testing.T) {
			w := httptest.NewRecorder()
			mux.ServeHTTP(w, authRequest(http.MethodGet, target, nil, cmd.APIToken))
			if w.Code != http.StatusBadRequest {
				t.Fatalf("expected 400, got %d - %s", w.Code, w.Body.String())
			}
		})
	}

	t.Run("valid score and sort still work", func(t *testing.T) {
		w := httptest.NewRecorder()
		mux.ServeHTTP(
			w,
			authRequest(http.MethodGet, "/api/query?min_score=0&sort=path%20desc", nil, cmd.APIToken),
		)
		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d - %s", w.Code, w.Body.String())
		}
	})

	var count int
	if err := sqlDB.QueryRow("SELECT COUNT(*) FROM media").Scan(&count); err != nil {
		t.Fatalf("media table is gone: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected media row to survive, got %d", count)
	}
}

func TestMetadataRejectsDatabaseOutsideAllowedList(t *testing.T) {
	dbPath, sqlDB := setupPhase7DB(t)
	defer sqlDB.Close()
	if _, err := sqlDB.Exec(
		`INSERT INTO media (path, title, media_type, time_deleted) VALUES ('/media/a.mp4', 'A', 'video', 0)`,
	); err != nil {
		t.Fatal(err)
	}

	cmd := securityTestServer(dbPath)
	defer cmd.Close()
	mux := cmd.Mux()

	allowed := authRequest(
		http.MethodGet,
		"/api/metadata?db="+url.QueryEscape(dbPath)+"&path=/media/a.mp4",
		nil,
		cmd.APIToken,
	)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, allowed)
	if w.Code != http.StatusOK {
		t.Fatalf("expected allowed DB to return 200, got %d - %s", w.Code, w.Body.String())
	}

	otherDB := filepath.Join(t.TempDir(), "other.db")
	rejected := authRequest(
		http.MethodGet,
		"/api/metadata?db="+url.QueryEscape(otherDB)+"&path=/media/a.mp4",
		nil,
		cmd.APIToken,
	)
	w = httptest.NewRecorder()
	mux.ServeHTTP(w, rejected)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for an unlisted DB, got %d", w.Code)
	}
	if _, err := os.Stat(otherDB); !os.IsNotExist(err) {
		t.Fatalf("metadata must not create the unlisted SQLite file")
	}
}

func TestEmptyBinRejectsUnlistedAndUntrashedPaths(t *testing.T) {
	dbPath, sqlDB := setupPhase7DB(t)
	defer sqlDB.Close()
	if _, err := sqlDB.Exec(
		`INSERT INTO media (path, media_type, time_deleted) VALUES ('/trash/a.mp4', 'video', 1), ('/keep/b.mp4', 'video', 0)`,
	); err != nil {
		t.Fatal(err)
	}

	cmd := securityTestServer(dbPath)
	defer cmd.Close()
	mux := cmd.Mux()

	body, _ := json.Marshal(map[string][]string{"paths": {"/keep/b.mp4"}})
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, authRequest(http.MethodPost, "/api/empty-bin", bytes.NewReader(body), cmd.APIToken))
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for an untrashed path, got %d - %s", w.Code, w.Body.String())
	}
	var result server.EmptyBinResult
	if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if result.Skipped != 1 || result.Deleted != 0 {
		t.Fatalf("unexpected result %+v", result)
	}

	var remaining int
	if err := sqlDB.QueryRow("SELECT COUNT(*) FROM media WHERE path = '/keep/b.mp4'").Scan(&remaining); err != nil {
		t.Fatal(err)
	}
	if remaining != 1 {
		t.Fatal("unlisted path must not be deleted")
	}
}

func TestEmptyBinRejectsPathFromAnotherDatabase(t *testing.T) {
	configuredDB, configuredSQL := setupPhase7DB(t)
	defer configuredSQL.Close()

	otherDB := filepath.Join(t.TempDir(), "other.db")
	otherSQL, err := sql.Open("sqlite3", otherDB)
	if err != nil {
		t.Fatal(err)
	}
	defer otherSQL.Close()
	if err := db.InitDB(context.Background(), otherSQL); err != nil {
		t.Fatal(err)
	}
	otherPath := filepath.Join(t.TempDir(), "other-trash.mp4")
	if _, err := otherSQL.Exec(
		`INSERT INTO media (path, media_type, time_deleted) VALUES (?, 'video', 1)`,
		otherPath,
	); err != nil {
		t.Fatal(err)
	}

	cmd := securityTestServer(configuredDB)
	defer cmd.Close()
	mux := cmd.Mux()

	body, _ := json.Marshal(map[string][]string{"paths": {otherPath}})
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, authRequest(http.MethodPost, "/api/empty-bin", bytes.NewReader(body), cmd.APIToken))
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for a path outside the configured DBs, got %d - %s", w.Code, w.Body.String())
	}
}

func TestEmptyBinListedPathAndReadOnly(t *testing.T) {
	dbPath, sqlDB := setupPhase7DB(t)
	defer sqlDB.Close()

	filePath := filepath.Join(filepath.Dir(dbPath), "trashed.mp4")
	if err := os.WriteFile(filePath, []byte("data"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := sqlDB.Exec(
		`INSERT INTO media (path, media_type, time_deleted) VALUES (?, 'video', 1)`,
		filePath,
	); err != nil {
		t.Fatal(err)
	}

	cmd := securityTestServer(dbPath)
	defer cmd.Close()
	mux := cmd.Mux()

	body, _ := json.Marshal(map[string][]string{"paths": {filePath}})
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, authRequest(http.MethodPost, "/api/empty-bin", bytes.NewReader(body), cmd.APIToken))
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d - %s", w.Code, w.Body.String())
	}
	var result server.EmptyBinResult
	if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if result.Deleted != 1 || result.Failed != 0 {
		t.Fatalf("unexpected result %+v", result)
	}
	if _, err := os.Stat(filePath); !os.IsNotExist(err) {
		t.Fatal("expected file to be removed")
	}

	cmd.ReadOnly = true
	w = httptest.NewRecorder()
	mux.ServeHTTP(w, authRequest(http.MethodPost, "/api/empty-bin", bytes.NewReader(body), cmd.APIToken))
	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403 in read-only mode, got %d", w.Code)
	}
}

func TestEmptyBinPartialFailure(t *testing.T) {
	dbPath, sqlDB := setupPhase7DB(t)
	defer sqlDB.Close()

	dir := filepath.Dir(dbPath)
	goodFile := filepath.Join(dir, "good.mp4")
	if err := os.WriteFile(goodFile, []byte("data"), 0o644); err != nil {
		t.Fatal(err)
	}
	badDir := filepath.Join(dir, "unremovable")
	if err := os.MkdirAll(badDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(badDir, "child"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := sqlDB.Exec(
		`INSERT INTO media (path, media_type, time_deleted) VALUES (?, 'video', 1), (?, 'video', 1)`,
		goodFile,
		badDir,
	); err != nil {
		t.Fatal(err)
	}

	cmd := securityTestServer(dbPath)
	defer cmd.Close()
	mux := cmd.Mux()

	body, _ := json.Marshal(map[string][]string{"paths": {goodFile, badDir}})
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, authRequest(http.MethodPost, "/api/empty-bin", bytes.NewReader(body), cmd.APIToken))
	if w.Code != http.StatusMultiStatus {
		t.Fatalf("expected 207 partial success, got %d - %s", w.Code, w.Body.String())
	}
	var result server.EmptyBinResult
	if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if result.Deleted != 1 || result.Failed != 1 {
		t.Fatalf("unexpected result %+v", result)
	}

	var badRows int
	if err := sqlDB.QueryRow("SELECT COUNT(*) FROM media WHERE path = ?", badDir).Scan(&badRows); err != nil {
		t.Fatal(err)
	}
	if badRows != 1 {
		t.Fatal("failed deletion must keep its trash row for retry")
	}
}

func TestEpubRequiresDatabaseMembershipAndContainsAssets(t *testing.T) {
	dbPath, sqlDB := setupPhase7DB(t)
	defer sqlDB.Close()

	dir := filepath.Dir(dbPath)
	pdfPath := filepath.Join(dir, "book.pdf")
	epubPath := filepath.Join(dir, "book.epub")
	unlistedPath := filepath.Join(dir, "unlisted.epub")
	if err := os.WriteFile(pdfPath, []byte("%PDF-1.4\n%%EOF\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(epubPath, []byte("dummy epub"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(unlistedPath, []byte("dummy epub"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := sqlDB.Exec(
		`INSERT INTO media (path, media_type, time_deleted) VALUES (?, 'text', 0), (?, 'text', 0)`,
		pdfPath,
		epubPath,
	); err != nil {
		t.Fatal(err)
	}

	// Fake ebook-convert so the EPUB path is exercised without calibre.
	binDir := filepath.Join(t.TempDir(), "bin")
	if err := os.MkdirAll(binDir, 0o755); err != nil {
		t.Fatal(err)
	}
	script := "#!/bin/sh\nmkdir -p \"$2\"\nprintf '<html><body>book</body></html>' > \"$2/index.html\"\n"
	if err := os.WriteFile(filepath.Join(binDir, "ebook-convert"), []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))
	t.Setenv("HOME", t.TempDir())

	cmd := securityTestServer(dbPath)
	defer cmd.Close()

	t.Run("unlisted document is denied", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/epub/"+strings.TrimPrefix(unlistedPath, "/"), nil)
		w := httptest.NewRecorder()
		cmd.HandleEpubConvert(w, req)
		if w.Code != http.StatusForbidden {
			t.Fatalf("expected 403, got %d - %s", w.Code, w.Body.String())
		}
	})

	t.Run("listed PDF is served", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/epub/"+strings.TrimPrefix(pdfPath, "/"), nil)
		w := httptest.NewRecorder()
		cmd.HandleEpubConvert(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d - %s", w.Code, w.Body.String())
		}
		if w.Header().Get("X-Content-Type-Options") != "nosniff" {
			t.Error("expected nosniff")
		}
	})

	t.Run("listed EPUB returns sandboxed wrapper", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/epub/"+strings.TrimPrefix(epubPath, "/"), nil)
		w := httptest.NewRecorder()
		cmd.HandleEpubConvert(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d - %s", w.Code, w.Body.String())
		}
		csp := w.Header().Get("Content-Security-Policy")
		if !strings.Contains(csp, "default-src 'none'") || !strings.Contains(csp, "frame-src 'self'") {
			t.Fatalf("wrapper should have a restrictive CSP; got %q", csp)
		}
		if !strings.Contains(w.Body.String(), `sandbox="allow-same-origin"`) {
			t.Error("book iframe should be sandboxed")
		}
	})

	t.Run("traversal is rejected", func(t *testing.T) {
		for _, target := range []string{
			"/api/epub/../../etc/passwd",
			"/api/epub/%2e%2e/%2e%2e/etc/passwd",
			"/api/epub/" + strings.TrimPrefix(epubPath, "/") + "/../../etc/passwd",
		} {
			req := httptest.NewRequest(http.MethodGet, target, nil)
			w := httptest.NewRecorder()
			cmd.HandleEpubConvert(w, req)
			if w.Code == http.StatusOK {
				t.Fatalf("traversal %q must not be served", target)
			}
		}
	})

	t.Run("sibling prefix asset is rejected", func(t *testing.T) {
		req := httptest.NewRequest(
			http.MethodGet,
			"/api/epub/"+strings.TrimPrefix(epubPath, "/")+"/../book.epub.evil",
			nil,
		)
		w := httptest.NewRecorder()
		cmd.HandleEpubConvert(w, req)
		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected 400 for sibling-prefix asset, got %d - %s", w.Code, w.Body.String())
		}
	})
}

func TestStreamingRequiresDatabaseMembership(t *testing.T) {
	dbPath, sqlDB := setupPhase7DB(t)
	defer sqlDB.Close()

	dir := filepath.Dir(dbPath)
	mediaPath := filepath.Join(dir, "movie.mkv")
	unlistedPath := filepath.Join(dir, "unlisted.mkv")
	if err := os.WriteFile(mediaPath, []byte("data"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(unlistedPath, []byte("data"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := sqlDB.Exec(
		`INSERT INTO media (path, title, media_type, duration, time_deleted) VALUES (?, 'Movie', 'video', 60, 0)`,
		mediaPath,
	); err != nil {
		t.Fatal(err)
	}

	cmd := securityTestServer(dbPath)
	defer cmd.Close()
	mux := cmd.Mux()

	t.Run("HLS playlist works for a listed file", func(t *testing.T) {
		w := httptest.NewRecorder()
		mux.ServeHTTP(
			w,
			authRequest(http.MethodGet, "/api/hls/playlist?path="+url.QueryEscape(mediaPath), nil, cmd.APIToken),
		)
		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d - %s", w.Code, w.Body.String())
		}
	})

	t.Run("HLS playlist denies an unlisted file", func(t *testing.T) {
		w := httptest.NewRecorder()
		mux.ServeHTTP(
			w,
			authRequest(http.MethodGet, "/api/hls/playlist?path="+url.QueryEscape(unlistedPath), nil, cmd.APIToken),
		)
		if w.Code != http.StatusForbidden {
			t.Fatalf("expected 403, got %d", w.Code)
		}
	})

	t.Run("HLS segment denies an unlisted file", func(t *testing.T) {
		w := httptest.NewRecorder()
		mux.ServeHTTP(
			w,
			authRequest(
				http.MethodGet,
				"/api/hls/segment?path="+url.QueryEscape(unlistedPath)+"&index=0",
				nil,
				cmd.APIToken,
			),
		)
		if w.Code != http.StatusForbidden {
			t.Fatalf("expected 403, got %d", w.Code)
		}
	})

	t.Run("play denies an unlisted local file", func(t *testing.T) {
		body, _ := json.Marshal(map[string]string{"path": unlistedPath})
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, authRequest(http.MethodPost, "/api/play", bytes.NewReader(body), cmd.APIToken))
		if w.Code != http.StatusForbidden {
			t.Fatalf("expected 403, got %d", w.Code)
		}
	})

	t.Run("missing listed file is not marked deleted", func(t *testing.T) {
		missing := filepath.Join(dir, "missing.mkv")
		if _, err := sqlDB.Exec(
			`INSERT INTO media (path, title, media_type, duration, time_deleted) VALUES (?, 'Missing', 'video', 60, 0)`,
			missing,
		); err != nil {
			t.Fatal(err)
		}
		w := httptest.NewRecorder()
		mux.ServeHTTP(
			w,
			authRequest(
				http.MethodGet,
				"/api/hls/segment?path="+url.QueryEscape(missing)+"&index=0",
				nil,
				cmd.APIToken,
			),
		)
		if w.Code != http.StatusNotFound {
			t.Fatalf("expected 404, got %d", w.Code)
		}
		var deleted sql.NullInt64
		if err := sqlDB.QueryRow("SELECT time_deleted FROM media WHERE path = ?", missing).Scan(&deleted); err != nil {
			t.Fatal(err)
		}
		if deleted.Valid && deleted.Int64 != 0 {
			t.Fatal("playback probe must not mark the file deleted")
		}
	})
}

func TestSubtitleSidecarUsesIndexedLookup(t *testing.T) {
	dbPath, sqlDB := setupPhase7DB(t)
	defer sqlDB.Close()

	dir := filepath.Dir(dbPath)
	mediaPath := filepath.Join(dir, "movie.mkv")
	sidecarPath := filepath.Join(dir, "movie.en.vtt")
	if err := os.WriteFile(mediaPath, []byte("data"), 0o644); err != nil {
		t.Fatal(err)
	}
	sidecar := "WEBVTT\n\n00:00:01.000 --> 00:00:02.000\nhello\n"
	if err := os.WriteFile(sidecarPath, []byte(sidecar), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := sqlDB.Exec(
		`INSERT INTO media (path, title, media_type, subtitle_count, time_deleted) VALUES (?, 'Movie', 'video', 1, 0)`,
		mediaPath,
	); err != nil {
		t.Fatal(err)
	}

	cmd := securityTestServer(dbPath)
	defer cmd.Close()
	mux := cmd.Mux()

	w := httptest.NewRecorder()
	mux.ServeHTTP(
		w,
		authRequest(http.MethodGet, "/api/subtitles?path="+url.QueryEscape(sidecarPath), nil, cmd.APIToken),
	)
	if w.Code != http.StatusOK {
		t.Fatalf("expected sidecar subtitle to be served, got %d - %s", w.Code, w.Body.String())
	}
	if w.Header().Get("Content-Type") != "text/vtt" {
		t.Fatalf("expected text/vtt, got %q", w.Header().Get("Content-Type"))
	}
}

func TestWriteHandlersReportFailures(t *testing.T) {
	dbPath, sqlDB := setupPhase7DB(t)
	defer sqlDB.Close()
	mediaPath := filepath.Join(filepath.Dir(dbPath), "rated.mp4")
	if _, err := sqlDB.Exec(
		`INSERT INTO media (path, title, media_type, score, time_deleted) VALUES (?, 'Rated', 'video', 0, 0)`,
		mediaPath,
	); err != nil {
		t.Fatal(err)
	}

	invalidDB := filepath.Join(t.TempDir(), "missing", "db.sqlite")
	makeBody := func() *bytes.Reader {
		body, _ := json.Marshal(map[string]any{"path": mediaPath, "score": 4})
		return bytes.NewReader(body)
	}

	t.Run("total failure returns 500", func(t *testing.T) {
		cmd := &server.Server{Databases: []string{invalidDB}, APIToken: "test-token"}
		defer cmd.Close()
		w := httptest.NewRecorder()
		cmd.Mux().ServeHTTP(w, authRequest(http.MethodPost, "/api/rate", makeBody(), cmd.APIToken))
		if w.Code != http.StatusInternalServerError {
			t.Fatalf("expected 500, got %d - %s", w.Code, w.Body.String())
		}
	})

	t.Run("partial failure returns 207 and still updates", func(t *testing.T) {
		cmd := &server.Server{Databases: []string{dbPath, invalidDB}, APIToken: "test-token"}
		defer cmd.Close()
		w := httptest.NewRecorder()
		cmd.Mux().ServeHTTP(w, authRequest(http.MethodPost, "/api/rate", makeBody(), cmd.APIToken))
		if w.Code != http.StatusMultiStatus {
			t.Fatalf("expected 207, got %d - %s", w.Code, w.Body.String())
		}
		var score float64
		if err := sqlDB.QueryRow("SELECT score FROM media WHERE path = ?", mediaPath).Scan(&score); err != nil {
			t.Fatal(err)
		}
		if score != 4 {
			t.Fatalf("expected score 4, got %v", score)
		}
	})

	t.Run("unknown path returns 404", func(t *testing.T) {
		cmd := securityTestServer(dbPath)
		defer cmd.Close()
		body, _ := json.Marshal(map[string]any{"path": "/media/unknown.mp4", "score": 3})
		w := httptest.NewRecorder()
		cmd.Mux().ServeHTTP(w, authRequest(http.MethodPost, "/api/rate", bytes.NewReader(body), cmd.APIToken))
		if w.Code != http.StatusNotFound {
			t.Fatalf("expected 404, got %d - %s", w.Code, w.Body.String())
		}
	})
}

func TestRequireWriteMethods(t *testing.T) {
	dbPath, sqlDB := setupPhase7DB(t)
	defer sqlDB.Close()
	cmd := securityTestServer(dbPath)
	defer cmd.Close()
	mux := cmd.Mux()

	tests := []struct {
		method string
		target string
		want   int
	}{
		{http.MethodGet, "/api/categorize/apply", http.StatusMethodNotAllowed},
		{http.MethodGet, "/api/categorize/keyword", http.StatusMethodNotAllowed},
		{http.MethodPut, "/api/categorize/keyword", http.StatusMethodNotAllowed},
		{http.MethodGet, "/api/categorize/keywords", http.StatusOK},
	}

	for _, tt := range tests {
		t.Run(tt.method+" "+tt.target, func(t *testing.T) {
			w := httptest.NewRecorder()
			mux.ServeHTTP(w, authRequest(tt.method, tt.target, nil, cmd.APIToken))
			if w.Code != tt.want {
				t.Fatalf("expected %d, got %d - %s", tt.want, w.Code, w.Body.String())
			}
		})
	}
}
