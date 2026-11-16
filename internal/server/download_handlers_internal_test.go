package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/chapmanjacobd/discoteca/internal/downloader"
)

type stubDownloadService struct {
	cfg      downloader.Config
	status   *downloader.Status
	startErr error
	started  *downloader.Options
	canceled bool
}

func (s *stubDownloadService) Config() downloader.Config { return s.cfg }

func (s *stubDownloadService) Start(opts downloader.Options) (*downloader.Status, error) {
	s.started = &opts
	if s.startErr != nil {
		return nil, s.startErr
	}
	return &downloader.Status{State: downloader.StateRunning, Progress: -1}, nil
}

func (s *stubDownloadService) Status() *downloader.Status { return s.status }
func (s *stubDownloadService) Cancel() bool               { s.canceled = true; return true }
func (s *stubDownloadService) Close()                     {}

func doDownloadRequest(t *testing.T, cmd *Server, method, path, body string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(method, path, bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Disco-Token", cmd.APIToken)
	w := httptest.NewRecorder()
	cmd.Mux().ServeHTTP(w, req)
	return w
}

func TestHandleDownloadReadOnly(t *testing.T) {
	cmd := &Server{
		Databases:       []string{"/tmp/library.db"},
		ReadOnly:        true,
		downloadManager: &stubDownloadService{},
	}
	defer cmd.Close()

	w := doDownloadRequest(t, cmd, http.MethodPost, "/api/download", `{"url":"https://example.test/video"}`)
	if w.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403: %s", w.Code, w.Body.String())
	}
}

func TestHandleDownloadUnavailable(t *testing.T) {
	cmd := &Server{Databases: []string{"/tmp/library.db"}}
	defer cmd.Close()

	w := doDownloadRequest(t, cmd, http.MethodPost, "/api/download", `{"url":"https://example.test/video"}`)
	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want 503: %s", w.Code, w.Body.String())
	}
}

func TestHandleDownloadStart(t *testing.T) {
	stub := &stubDownloadService{}
	cmd := &Server{Databases: []string{"/tmp/library.db"}, downloadManager: stub}
	defer cmd.Close()

	w := doDownloadRequest(
		t,
		cmd,
		http.MethodPost,
		"/api/download",
		`{"url":"https://example.test/video","database":"/tmp/library.db","media_type":"video"}`,
	)
	if w.Code != http.StatusAccepted {
		t.Fatalf("status = %d, want 202: %s", w.Code, w.Body.String())
	}
	if stub.started == nil || stub.started.Database != "/tmp/library.db" {
		t.Errorf("started options = %#v", stub.started)
	}

	var status downloader.Status
	if err := json.Unmarshal(w.Body.Bytes(), &status); err != nil {
		t.Fatal(err)
	}
	if status.State != downloader.StateRunning {
		t.Errorf("state = %q", status.State)
	}
}

func TestHandleDownloadRejectsUnknownDatabase(t *testing.T) {
	stub := &stubDownloadService{}
	cmd := &Server{Databases: []string{"/tmp/library.db"}, downloadManager: stub}
	defer cmd.Close()

	w := doDownloadRequest(
		t,
		cmd,
		http.MethodPost,
		"/api/download",
		`{"url":"https://example.test/video","database":"/tmp/other.db"}`,
	)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400: %s", w.Code, w.Body.String())
	}
	if stub.started != nil {
		t.Error("Start should not be called for an unknown database")
	}
}

func TestHandleDownloadBusy(t *testing.T) {
	stub := &stubDownloadService{
		startErr: downloader.ErrBusy,
		status:   &downloader.Status{State: downloader.StateRunning, Progress: 12},
	}
	cmd := &Server{Databases: []string{"/tmp/library.db"}, downloadManager: stub}
	defer cmd.Close()

	w := doDownloadRequest(t, cmd, http.MethodPost, "/api/download", `{"url":"https://example.test/video"}`)
	if w.Code != http.StatusConflict {
		t.Fatalf("status = %d, want 409: %s", w.Code, w.Body.String())
	}
	var status downloader.Status
	if err := json.Unmarshal(w.Body.Bytes(), &status); err != nil {
		t.Fatal(err)
	}
	if status.State != downloader.StateRunning {
		t.Errorf("state = %q", status.State)
	}
}

func TestHandleDownloadStatusIdle(t *testing.T) {
	cmd := &Server{Databases: []string{"/tmp/library.db"}, downloadManager: &stubDownloadService{}}
	defer cmd.Close()

	w := doDownloadRequest(t, cmd, http.MethodGet, "/api/download", "")
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200: %s", w.Code, w.Body.String())
	}
	var status downloader.Status
	if err := json.Unmarshal(w.Body.Bytes(), &status); err != nil {
		t.Fatal(err)
	}
	if status.State != downloader.StateIdle {
		t.Errorf("state = %q", status.State)
	}
}

func TestHandleDownloadCancel(t *testing.T) {
	stub := &stubDownloadService{status: &downloader.Status{State: downloader.StateRunning}}
	cmd := &Server{Databases: []string{"/tmp/library.db"}, downloadManager: stub}
	defer cmd.Close()

	w := doDownloadRequest(t, cmd, http.MethodPost, "/api/download/cancel", "")
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200: %s", w.Code, w.Body.String())
	}
	if !stub.canceled {
		t.Error("Cancel was not called")
	}
}

func TestHandleDatabasesIncludesDownloadInfo(t *testing.T) {
	cmd := &Server{
		Databases: []string{"/tmp/library.db"},
		downloadManager: &stubDownloadService{cfg: downloader.Config{
			Kind:        downloader.KindLB,
			DownloadDir: "/tmp/downloads",
		}},
	}
	defer cmd.Close()

	w := doDownloadRequest(t, cmd, http.MethodGet, "/api/databases", "")
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200: %s", w.Code, w.Body.String())
	}
	var info struct {
		DownloadAvailable bool   `json:"download_available"`
		DownloadCommand   string `json:"download_command"`
		DownloadDir       string `json:"download_dir"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &info); err != nil {
		t.Fatal(err)
	}
	if !info.DownloadAvailable || info.DownloadCommand != "lb" || info.DownloadDir != "/tmp/downloads" {
		t.Errorf("info = %#v", info)
	}
}
