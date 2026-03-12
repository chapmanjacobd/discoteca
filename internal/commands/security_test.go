package commands

import (
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/chapmanjacobd/discoteca/internal/testutils"
)

func TestSecurity_Blacklist(t *testing.T) {
	fixture := testutils.Setup(t)
	defer fixture.Cleanup()

	cmd := &ServeCmd{
		Databases: []string{fixture.DBPath},
	}
	defer cmd.Close()
	cmd.APIToken = "test-token"

	testCases := []struct {
		path     string
		expected int
	}{
		{filepath.FromSlash("/etc/passwd"), http.StatusForbidden},
		{filepath.FromSlash("/home/user/.ssh/id_rsa"), http.StatusForbidden},
		{filepath.FromSlash("/media/video.mp4"), http.StatusNotFound}, // Returns 404 when not in DB
	}

	for _, tc := range testCases {
		req := httptest.NewRequest(http.MethodGet, "/api/raw?path="+tc.path, nil)
		req.Header.Set("X-Disco-Token", cmd.APIToken)
		w := httptest.NewRecorder()
		cmd.handleRaw(w, req)
		if w.Code != tc.expected {
			t.Errorf("Path %s: expected status %d, got %d", tc.path, tc.expected, w.Code)
		}
	}
}
