package commands

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/chapmanjacobd/discoteca/internal/testutils"
)

func setupTestServeCmd(dbPath string) *ServeCmd {
	cmd := &ServeCmd{
		Databases: []string{dbPath},
	}
	cmd.APIToken = "test-token"
	return cmd
}

func TestServeCmd_HandlePlay_FileNotFound(t *testing.T) {
	fixture := testutils.Setup(t)
	defer fixture.Cleanup()

	// 1. Add a file to DB
	f1 := fixture.CreateDummyFile("media1.mp4")
	addCmd := &AddCmd{
		Args: []string{fixture.DBPath, f1},
	}
	addCmd.AfterApply()
	addCmd.Run(nil)

	// 2. Delete the file from filesystem
	os.Remove(f1)

	// 3. Setup ServeCmd
	cmd := setupTestServeCmd(fixture.DBPath)

	// 4. Create request
	reqBody, _ := json.Marshal(map[string]string{"path": f1})
	req := httptest.NewRequest(http.MethodPost, "/api/play", bytes.NewBuffer(reqBody))
	req.Header.Set("X-Disco-Token", cmd.APIToken)
	w := httptest.NewRecorder()

	// 5. Call handlePlay
	cmd.handlePlay(w, req)

	// 6. Verify response
	resp := w.Result()
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("Expected status 404, got %d", resp.StatusCode)
	}

	// 7. Verify DB update
	dbConn := fixture.GetDB()
	defer dbConn.Close()
	var timeDeleted sql.NullInt64
	err := dbConn.QueryRow("SELECT time_deleted FROM media WHERE path = ?", f1).Scan(&timeDeleted)
	if err != nil {
		t.Fatalf("Query failed: %v", err)
	}
	if !timeDeleted.Valid || timeDeleted.Int64 == 0 {
		t.Errorf("Expected file to be marked as deleted in DB")
	}
}

func TestServeCmd_HandleHLSSegment_FileNotFound(t *testing.T) {
	fixture := testutils.Setup(t)
	defer fixture.Cleanup()

	// 1. Add a file to DB
	f1 := fixture.CreateDummyFile("media1.mp4")
	addCmd := &AddCmd{
		Args: []string{fixture.DBPath, f1},
	}
	addCmd.AfterApply()
	addCmd.Run(nil)

	// 2. Delete the file from filesystem
	os.Remove(f1)

	// 3. Setup ServeCmd
	cmd := setupTestServeCmd(fixture.DBPath)

	// 4. Create request
	req := httptest.NewRequest(http.MethodGet, "/api/hls/segment?path="+f1+"&index=0", nil)
	req.Header.Set("X-Disco-Token", cmd.APIToken)
	w := httptest.NewRecorder()

	// 5. Call handleHLSSegment
	cmd.handleHLSSegment(w, req)

	// 6. Verify response
	resp := w.Result()
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("Expected status 404, got %d", resp.StatusCode)
	}

	// 7. Verify DB update
	dbConn := fixture.GetDB()
	defer dbConn.Close()
	var timeDeleted sql.NullInt64
	err := dbConn.QueryRow("SELECT time_deleted FROM media WHERE path = ?", f1).Scan(&timeDeleted)
	if err != nil {
		t.Fatalf("Query failed: %v", err)
	}
	if !timeDeleted.Valid || timeDeleted.Int64 == 0 {
		t.Errorf("Expected file to be marked as deleted in DB")
	}
}

func TestServeCmd_HandleSubtitles_FileNotFound(t *testing.T) {
	fixture := testutils.Setup(t)
	defer fixture.Cleanup()

	// 1. Add a file to DB
	f1 := fixture.CreateDummyFile("media1.mp4")
	addCmd := &AddCmd{
		Args: []string{fixture.DBPath, f1},
	}
	addCmd.AfterApply()
	addCmd.Run(nil)

	// 2. Delete the file from filesystem
	os.Remove(f1)

	// 3. Setup ServeCmd
	cmd := setupTestServeCmd(fixture.DBPath)

	// 4. Create request
	req := httptest.NewRequest(http.MethodGet, "/api/subtitles?path="+f1+"&index=0", nil)
	req.Header.Set("X-Disco-Token", cmd.APIToken)
	w := httptest.NewRecorder()

	// 5. Call handleSubtitles
	cmd.handleSubtitles(w, req)

	// 6. Verify response
	resp := w.Result()
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("Expected status 404, got %d", resp.StatusCode)
	}

	// 7. Verify DB update
	dbConn := fixture.GetDB()
	defer dbConn.Close()
	var timeDeleted sql.NullInt64
	err := dbConn.QueryRow("SELECT time_deleted FROM media WHERE path = ?", f1).Scan(&timeDeleted)
	if err != nil {
		t.Fatalf("Query failed: %v", err)
	}
	if !timeDeleted.Valid || timeDeleted.Int64 == 0 {
		t.Errorf("Expected file to be marked as deleted in DB")
	}
}
