package commands_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/chapmanjacobd/discoteca/internal/commands"
	"github.com/chapmanjacobd/discoteca/internal/db"
	"github.com/chapmanjacobd/discoteca/internal/testutils"
)

// TestDatabaseFiltering_Security tests that the server rejects queries for databases
// that were not configured at startup
func TestDatabaseFiltering_Security(t *testing.T) {
	fixture := testutils.Setup(t)
	defer fixture.Cleanup()

	// Initialize the DB
	sqlDB := fixture.GetDB()
	db.InitDB(context.Background(), sqlDB)
	sqlDB.Close()

	// Create commands.ServeCmd with only the fixture database
	cmd := &commands.ServeCmd{
		Databases: []string{fixture.DBPath},
	}
	handler := cmd.Mux()

	t.Run("RejectUnauthorizedDatabase", func(t *testing.T) {
		// Try to query a database path that wasn't configured
		req := httptest.NewRequest(http.MethodGet, "/api/query?db=/etc/malicious.db", nil)
		req.Header.Set("X-Disco-Token", cmd.APIToken)
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)

		// Should return 400 Bad Request
		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected 400 for unauthorized database, got %d", w.Code)
		}

		// Response should mention the security violation
		body := w.Body.String()
		if !strings.Contains(body, "not in allowed list") {
			t.Errorf("Expected error about unauthorized database, got: %s", body)
		}
	})

	t.Run("RejectMultipleUnauthorizedDatabases", func(t *testing.T) {
		// Try to query multiple unauthorized databases
		req := httptest.NewRequest(http.MethodGet, "/api/query?db=/etc/passwd.db&db=/tmp/evil.db", nil)
		req.Header.Set("X-Disco-Token", cmd.APIToken)
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected 400 for unauthorized databases, got %d", w.Code)
		}
	})

	t.Run("RejectMixedAuthorizedAndUnauthorized", func(t *testing.T) {
		// Try to query one valid and one invalid database
		req := httptest.NewRequest(http.MethodGet, "/api/query?db="+fixture.DBPath+"&db=/tmp/evil.db", nil)
		req.Header.Set("X-Disco-Token", cmd.APIToken)
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected 400 for mixed databases, got %d", w.Code)
		}
	})

	t.Run("AllowAuthorizedDatabase", func(t *testing.T) {
		// Query the configured database - should work
		req := httptest.NewRequest(http.MethodGet, "/api/query?db="+fixture.DBPath, nil)
		req.Header.Set("X-Disco-Token", cmd.APIToken)
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)

		// Should return 200 OK (even if no results)
		if w.Code != http.StatusOK {
			t.Errorf("Expected 200 for authorized database, got %d: %s", w.Code, w.Body.String())
		}
	})

	t.Run("AllowNoDatabaseFilter", func(t *testing.T) {
		// Query without database filter - should use all configured databases
		req := httptest.NewRequest(http.MethodGet, "/api/query", nil)
		req.Header.Set("X-Disco-Token", cmd.APIToken)
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected 200 for no database filter, got %d: %s", w.Code, w.Body.String())
		}
	})
}

// TestDatabaseFiltering_FilterBins tests that filter bins endpoint also respects database filtering
func TestDatabaseFiltering_FilterBins(t *testing.T) {
	fixture := testutils.Setup(t)
	defer fixture.Cleanup()

	// Initialize the DB with some test data
	sqlDB := fixture.GetDB()
	db.InitDB(context.Background(), sqlDB)
	_, err := sqlDB.Exec(`
		INSERT INTO media (path, size, duration, media_type, time_deleted)
		VALUES ('/test/video.mp4', 1000000, 120, 'video', 0)
	`)
	if err != nil {
		t.Fatal(err)
	}
	sqlDB.Close()

	cmd := &commands.ServeCmd{
		Databases: []string{fixture.DBPath},
	}
	handler := cmd.Mux()

	t.Run("FilterBins_RejectUnauthorizedDatabase", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/filter-bins?db=/etc/evil.db", nil)
		req.Header.Set("X-Disco-Token", cmd.APIToken)
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected 400 for unauthorized database in filter-bins, got %d", w.Code)
		}
	})

	t.Run("FilterBins_AllowAuthorizedDatabase", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/filter-bins?db="+fixture.DBPath, nil)
		req.Header.Set("X-Disco-Token", cmd.APIToken)
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected 200 for authorized database in filter-bins, got %d: %s", w.Code, w.Body.String())
		}
	})
}

// TestDatabaseFiltering_WithMultipleDatabases tests filtering with multiple configured databases
func TestDatabaseFiltering_WithMultipleDatabases(t *testing.T) {
	// Create two test fixtures
	fixture1 := testutils.Setup(t)
	defer fixture1.Cleanup()

	fixture2 := testutils.Setup(t)
	defer fixture2.Cleanup()

	// Initialize both databases with schema
	db1 := fixture1.GetDB()
	db.InitDB(context.Background(), db1)
	db1.Close()

	db2 := fixture2.GetDB()
	db.InitDB(context.Background(), db2)
	db2.Close()

	cmd := &commands.ServeCmd{
		Databases: []string{fixture1.DBPath, fixture2.DBPath},
	}
	handler := cmd.Mux()

	t.Run("AllowSubsetOfDatabases", func(t *testing.T) {
		// Query only the first database
		req := httptest.NewRequest(http.MethodGet, "/api/query?db="+fixture1.DBPath, nil)
		req.Header.Set("X-Disco-Token", cmd.APIToken)
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected 200 for subset of databases, got %d: %s", w.Code, w.Body.String())
		}
	})

	t.Run("AllowAllConfiguredDatabases", func(t *testing.T) {
		// Query both configured databases
		req := httptest.NewRequest(http.MethodGet, "/api/query?db="+fixture1.DBPath+"&db="+fixture2.DBPath, nil)
		req.Header.Set("X-Disco-Token", cmd.APIToken)
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected 200 for all configured databases, got %d: %s", w.Code, w.Body.String())
		}
	})

	t.Run("RejectNonConfiguredDatabase", func(t *testing.T) {
		// Try to query a database that exists but wasn't configured
		otherFixture := testutils.Setup(t)
		defer otherFixture.Cleanup()

		// Initialize the other database too
		otherDB := otherFixture.GetDB()
		db.InitDB(context.Background(), otherDB)
		otherDB.Close()

		req := httptest.NewRequest(http.MethodGet, "/api/query?db="+otherFixture.DBPath, nil)
		req.Header.Set("X-Disco-Token", cmd.APIToken)
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected 400 for non-configured database, got %d: %s", w.Code, w.Body.String())
		}
	})
}
