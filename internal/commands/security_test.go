package commands_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/chapmanjacobd/discoteca/internal/commands"
	"github.com/chapmanjacobd/discoteca/internal/testutils"
)

func TestSecurity_Blocklist(t *testing.T) {
	fixture := testutils.Setup(t)
	defer fixture.Cleanup()

	cmd := &commands.ServeCmd{
		Databases: []string{fixture.DBPath},
	}
	defer cmd.Close()
	cmd.APIToken = "test-token"

	testCases := []struct {
		path     string
		expected int
	}{
		// Unix paths
		{"/etc/passwd", http.StatusForbidden},
		{"/home/user/.ssh/id_rsa", http.StatusForbidden},
		{"/media/video.mp4", http.StatusNotFound},

		// Windows paths (should also be blocked)
		{"\\etc\\passwd", http.StatusForbidden},
		{"\\home\\user\\.ssh\\id_rsa", http.StatusForbidden},
		{"C:\\Windows\\System32\\config\\sam", http.StatusForbidden},
		{"D:\\Windows\\System32\\config\\sam", http.StatusForbidden},
		{"C:\\Users\\user\\.ssh\\id_rsa", http.StatusForbidden},
		{"\\\\server\\share\\etc\\passwd", http.StatusForbidden},

		// Mixed separators and case sensitivity
		{"/home\\user\\.ssh/id_rsa", http.StatusForbidden},
		{"\\etc/passwd", http.StatusForbidden},
		{"c:/windows/system32/config/SAM", http.StatusForbidden},
		{"C:\\USERS\\USER\\.SSH\\ID_RSA", http.StatusForbidden},
	}

	for _, tc := range testCases {
		t.Run(tc.path, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/api/raw?path="+tc.path, nil)
			req.Header.Set("X-Disco-Token", cmd.APIToken)
			w := httptest.NewRecorder()
			cmd.HandleRaw(w, req)
			if w.Code != tc.expected {
				t.Errorf("Path %s: expected status %d, got %d", tc.path, tc.expected, w.Code)
			}
		})
	}
}
