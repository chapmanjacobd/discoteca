package commands

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/chapmanjacobd/discoteca/internal/testutils"
)

func TestServeCmd_HandleProgress_PlayCount(t *testing.T) {
	fixture := testutils.Setup(t)
	defer fixture.Cleanup()

	f1 := fixture.CreateDummyFile("media1.mp4")
	addCmd := &AddCmd{
		Args: []string{fixture.DBPath, f1},
	}
	addCmd.AfterApply()
	addCmd.Run(context.Background())

	cmd := &ServeCmd{
		Databases: []string{fixture.DBPath},
	}

	// 1. Initial play count should be 0
	dbConn := fixture.GetDB()
	defer dbConn.Close()
	var playCount int
	err := dbConn.QueryRow("SELECT COALESCE(play_count, 0) FROM media WHERE path = ?", f1).Scan(&playCount)
	if err != nil || playCount != 0 {
		t.Fatalf("Expected play_count 0, got %d (err: %v)", playCount, err)
	}

	// 2. Increment play count via /api/progress
	reqBody, _ := json.Marshal(map[string]any{
		"path":      f1,
		"playhead":  100,
		"duration":  200,
		"completed": true,
	})
	req := httptest.NewRequest(http.MethodPost, "/api/progress", bytes.NewBuffer(reqBody))
	w := httptest.NewRecorder()
	cmd.handleProgress(w, req)

	if w.Result().StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Result().StatusCode)
	}

	err = dbConn.QueryRow("SELECT COALESCE(play_count, 0) FROM media WHERE path = ?", f1).Scan(&playCount)
	if err != nil || playCount != 1 {
		t.Fatalf("Expected play_count 1, got %d (err: %v)", playCount, err)
	}

	// 3. Read-only mode should NOT increment play count
	cmd.ReadOnly = true
	req = httptest.NewRequest(http.MethodPost, "/api/progress", bytes.NewBuffer(reqBody))
	w = httptest.NewRecorder()
	cmd.handleProgress(w, req)

	if w.Result().StatusCode != http.StatusForbidden {
		t.Errorf("Expected status 403 in read-only mode, got %d", w.Result().StatusCode)
	}

	err = dbConn.QueryRow("SELECT COALESCE(play_count, 0) FROM media WHERE path = ?", f1).Scan(&playCount)
	if err != nil || playCount != 1 {
		t.Fatalf("Expected play_count still 1, got %d (err: %v)", playCount, err)
	}
}

func TestServeCmd_HandleMarkPlayed(t *testing.T) {
	fixture := testutils.Setup(t)
	defer fixture.Cleanup()

	f1 := fixture.CreateDummyFile("media1.mp4")
	addCmd := &AddCmd{
		Args: []string{fixture.DBPath, f1},
	}
	addCmd.AfterApply()
	addCmd.Run(context.Background())

	cmd := &ServeCmd{
		Databases: []string{fixture.DBPath},
	}

	dbConn := fixture.GetDB()
	defer dbConn.Close()

	// 1. Mark as played
	reqBody, _ := json.Marshal(map[string]string{"path": f1})
	req := httptest.NewRequest(http.MethodPost, "/api/mark-played", bytes.NewBuffer(reqBody))
	w := httptest.NewRecorder()
	cmd.handleMarkPlayed(w, req)

	if w.Result().StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Result().StatusCode)
	}

	var playCount int
	dbConn.QueryRow("SELECT COALESCE(play_count, 0) FROM media WHERE path = ?", f1).Scan(&playCount)
	if playCount != 1 {
		t.Errorf("Expected play_count 1, got %d", playCount)
	}

	// 2. Mark as played again
	req = httptest.NewRequest(http.MethodPost, "/api/mark-played", bytes.NewBuffer(json.RawMessage(reqBody)))
	w = httptest.NewRecorder()
	cmd.handleMarkPlayed(w, req)

	dbConn.QueryRow("SELECT COALESCE(play_count, 0) FROM media WHERE path = ?", f1).Scan(&playCount)
	if playCount != 2 {
		t.Errorf("Expected play_count 2, got %d", playCount)
	}
}

func TestServeCmd_HandleMarkUnplayed(t *testing.T) {
	fixture := testutils.Setup(t)
	defer fixture.Cleanup()

	f1 := fixture.CreateDummyFile("media1.mp4")
	addCmd := &AddCmd{
		Args: []string{fixture.DBPath, f1},
	}
	addCmd.AfterApply()
	addCmd.Run(context.Background())

	cmd := &ServeCmd{
		Databases: []string{fixture.DBPath},
	}

	dbConn := fixture.GetDB()
	defer dbConn.Close()

	// 1. Mark as played first
	reqBody, _ := json.Marshal(map[string]string{"path": f1})
	req := httptest.NewRequest(http.MethodPost, "/api/mark-played", bytes.NewBuffer(reqBody))
	cmd.handleMarkPlayed(httptest.NewRecorder(), req)

	// 2. Mark as unplayed
	req = httptest.NewRequest(http.MethodPost, "/api/mark-unplayed", bytes.NewBuffer(reqBody))
	w := httptest.NewRecorder()
	cmd.handleMarkUnplayed(w, req)

	if w.Result().StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Result().StatusCode)
	}

	var playCount int
	dbConn.QueryRow("SELECT COALESCE(play_count, 0) FROM media WHERE path = ?", f1).Scan(&playCount)
	if playCount != 0 {
		t.Errorf("Expected play_count 0, got %d", playCount)
	}
}
