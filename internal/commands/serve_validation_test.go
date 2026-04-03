package commands

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/chapmanjacobd/discoteca/internal/testutils"
)

func newAuthRequest(method, target string, body io.Reader, token string) *http.Request {
	req := httptest.NewRequest(method, target, body)
	req.Header.Set("X-Disco-Token", token)
	return req
}

func TestServeCmd_Validation(t *testing.T) {
	fixture := testutils.Setup(t)
	defer fixture.Cleanup()

	cmd := &ServeCmd{
		Databases: []string{fixture.DBPath},
	}
	handler := cmd.Mux()

	t.Run("HandleRate_InvalidBody", func(t *testing.T) {
		req := newAuthRequest(http.MethodPost, "/api/rate", bytes.NewBufferString("invalid json"), cmd.APIToken)
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)
		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected 400, got %d", w.Code)
		}
	})

	t.Run("HandleProgress_InvalidBody", func(t *testing.T) {
		req := newAuthRequest(http.MethodPost, "/api/progress", bytes.NewBufferString("invalid json"), cmd.APIToken)
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)
		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected 400, got %d", w.Code)
		}
	})

	t.Run("HandleMarkPlayed_InvalidBody", func(t *testing.T) {
		req := newAuthRequest(http.MethodPost, "/api/mark-played", bytes.NewBufferString("invalid json"), cmd.APIToken)
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)
		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected 400, got %d", w.Code)
		}
	})

	t.Run("HandleMarkUnplayed_InvalidBody", func(t *testing.T) {
		req := newAuthRequest(
			http.MethodPost,
			"/api/mark-unplayed",
			bytes.NewBufferString("invalid json"),
			cmd.APIToken,
		)
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)
		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected 400, got %d", w.Code)
		}
	})

	t.Run("HandleDelete_InvalidBody", func(t *testing.T) {
		req := newAuthRequest(http.MethodPost, "/api/delete", bytes.NewBufferString("invalid json"), cmd.APIToken)
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)
		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected 400, got %d", w.Code)
		}
	})

	t.Run("HandlePlay_InvalidMethod", func(t *testing.T) {
		req := newAuthRequest(http.MethodGet, "/api/play", nil, cmd.APIToken)
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)
		if w.Code != http.StatusMethodNotAllowed {
			t.Errorf("Expected 405, got %d", w.Code)
		}
	})

	t.Run("HandlePlay_InvalidBody", func(t *testing.T) {
		req := newAuthRequest(http.MethodPost, "/api/play", bytes.NewBufferString("invalid json"), cmd.APIToken)
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)
		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected 400, got %d", w.Code)
		}
	})

	t.Run("HandlePlaylistItems_MissingTitle", func(t *testing.T) {
		req := newAuthRequest(http.MethodGet, "/api/playlists/items", nil, cmd.APIToken)
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)
		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected 400, got %d", w.Code)
		}
	})

	t.Run("HandlePlaylistItems_InvalidBody", func(t *testing.T) {
		req := newAuthRequest(
			http.MethodPost,
			"/api/playlists/items",
			bytes.NewBufferString("invalid json"),
			cmd.APIToken,
		)
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)
		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected 400, got %d", w.Code)
		}
	})

	t.Run("HandlePlaylistReorder_InvalidBody", func(t *testing.T) {
		req := newAuthRequest(
			http.MethodPost,
			"/api/playlists/reorder",
			bytes.NewBufferString("invalid json"),
			cmd.APIToken,
		)
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)
		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected 400, got %d", w.Code)
		}
	})

	t.Run("HandlePlaylistReorder_InvalidMethod", func(t *testing.T) {
		req := newAuthRequest(http.MethodGet, "/api/playlists/reorder", nil, cmd.APIToken)
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)
		if w.Code != http.StatusMethodNotAllowed {
			t.Errorf("Expected 405, got %d", w.Code)
		}
	})
}

func TestServeCmd_HandleHLSSegment_Validation(t *testing.T) {
	fixture := testutils.Setup(t)
	defer fixture.Cleanup()

	cmd := &ServeCmd{
		Databases: []string{fixture.DBPath},
	}
	handler := cmd.Mux()

	t.Run("MissingPath", func(t *testing.T) {
		req := newAuthRequest(http.MethodGet, "/api/hls/segment?index=0", nil, cmd.APIToken)
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)
		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected 400, got %d", w.Code)
		}
	})

	t.Run("MissingIndex", func(t *testing.T) {
		req := newAuthRequest(http.MethodGet, "/api/hls/segment?path=test.mp4", nil, cmd.APIToken)
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)
		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected 400, got %d", w.Code)
		}
	})

	t.Run("InvalidIndex", func(t *testing.T) {
		req := newAuthRequest(http.MethodGet, "/api/hls/segment?path=test.mp4&index=abc", nil, cmd.APIToken)
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)
		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected 400, got %d", w.Code)
		}
	})
}

func TestServeCmd_HandleHLSPlaylist_Validation(t *testing.T) {
	fixture := testutils.Setup(t)
	defer fixture.Cleanup()

	cmd := &ServeCmd{
		Databases: []string{fixture.DBPath},
	}
	handler := cmd.Mux()

	t.Run("MissingPath", func(t *testing.T) {
		req := newAuthRequest(http.MethodGet, "/api/hls/playlist", nil, cmd.APIToken)
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)
		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected 400, got %d", w.Code)
		}
	})

	t.Run("NotFound", func(t *testing.T) {
		req := newAuthRequest(http.MethodGet, "/api/hls/playlist?path=nonexistent.mp4", nil, cmd.APIToken)
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)
		if w.Code != http.StatusNotFound {
			t.Errorf("Expected 404, got %d", w.Code)
		}
	})
}
