package utils_test

import (
	"bufio"
	"encoding/json"
	"net"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/chapmanjacobd/discoteca/internal/models"
	"github.com/chapmanjacobd/discoteca/internal/utils"
)

func TestMpvCall(t *testing.T) {
	socketPath := filepath.Join(t.TempDir(), "mpv-test.sock")
	ln, err := net.Listen("unix", socketPath)
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()

	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				return
			}
			go func(c net.Conn) {
				defer c.Close()
				scanner := bufio.NewScanner(c)
				for scanner.Scan() {
					var cmd utils.MpvCommand
					json.Unmarshal(scanner.Bytes(), &cmd)

					var resp utils.MpvResponse
					if len(cmd.Command) > 0 && cmd.Command[0] == "get_property" {
						resp = utils.MpvResponse{Data: 50, Error: "success"}
					} else if len(cmd.Command) > 0 && cmd.Command[0] == "error" {
						resp = utils.MpvResponse{Error: "invalid property"}
					} else {
						resp = utils.MpvResponse{Data: "pong", Error: "success"}
					}
					jsonData, _ := json.Marshal(resp)
					c.Write(append(jsonData, '\n'))
				}
			}(conn)
		}
	}()

	resp, err := utils.MpvCall(socketPath, "ping")
	if err != nil {
		t.Fatalf("utils.MpvCall failed: %v", err)
	}
	if resp.Data != "pong" {
		t.Errorf("Expected pong, got %v", resp.Data)
	}

	val, err := utils.MpvGetProperty(socketPath, "volume")
	if err != nil {
		t.Fatalf("utils.MpvGetProperty failed: %v", err)
	}
	if val.(float64) != 50 {
		t.Errorf("Expected 50, got %v", val)
	}

	err = utils.MpvSetProperty(socketPath, "volume", 60)
	if err != nil {
		t.Fatalf("utils.MpvSetProperty failed: %v", err)
	}

	err = utils.MpvPause(socketPath, true)
	if err != nil {
		t.Fatalf("utils.MpvPause failed: %v", err)
	}

	err = utils.MpvSeek(socketPath, 10, "relative")
	if err != nil {
		t.Fatalf("utils.MpvSeek failed: %v", err)
	}

	err = utils.MpvLoadFile(socketPath, "file.mp4", "replace")
	if err != nil {
		t.Fatalf("utils.MpvLoadFile failed: %v", err)
	}

	_, err = utils.MpvCall(socketPath, "error")
	if err == nil {
		t.Error("Expected error for 'error' command, got nil")
	}
}

func TestMpvWatchLaterValue(t *testing.T) {
	f, _ := os.CreateTemp(t.TempDir(), "watch-later")
	defer os.Remove(f.Name())
	f.WriteString("key1=val1\nkey2=val2\n")
	f.Close()

	val, err := utils.MpvWatchLaterValue(f.Name(), "key2")
	if err != nil {
		t.Fatal(err)
	}
	if val != "val2" {
		t.Errorf("Expected val2, got %s", val)
	}

	val, _ = utils.MpvWatchLaterValue(f.Name(), "missing")
	if val != "" {
		t.Errorf("Expected empty string for missing key, got %s", val)
	}
}

func TestPathToMpvWatchLaterMD5(t *testing.T) {
	// We want to test that it produces the SAME hash as mpv would for this path.
	// mpv uses forward slashes for the hash even on Windows.
	path := "/home/xk/github/xk/lb/tests/data/test.mp4"
	// The function internals should NOT use filepath.Abs if we provide an absolute-looking path starting with "/"
	got := utils.PathToMpvWatchLaterMD5(path)
	want := "E1E0D0E3F0D2CB748303FDA43224B7E7"
	if got != want {
		t.Errorf("utils.PathToMpvWatchLaterMD5(%s) = %s; want %s", path, got, want)
	}
}

func TestGetPlayhead(t *testing.T) {
	tmpDir := t.TempDir()

	flags := models.GlobalFlags{
		PlaybackFlags: models.PlaybackFlags{
			WatchLaterDir: tmpDir,
		},
	}
	path := "/home/runner/work/library/library/tests/data/test.mp4"
	md5Hash := utils.PathToMpvWatchLaterMD5(path)
	metadataPath := filepath.Join(tmpDir, md5Hash)

	// Use MPV time
	startTime := time.Now().Add(-2 * time.Second)
	os.WriteFile(metadataPath, []byte("start=5.000000\n"), 0o644)
	if ph := utils.GetPlayhead(flags, path, startTime, 0, 0); ph != 5 {
		t.Errorf("utils.GetPlayhead (mpv time) = %d; want 5", ph)
	}

	// Check invalid MPV time (beyond duration)
	os.WriteFile(metadataPath, []byte("start=13.000000\n"), 0o644)
	// utils.GetPlayhead currently returns mpvPlayhead if found.
	// The Python code:
	//   if mpv_playhead: return mpv_playhead
	// Wait, I should re-read Python's get_playhead logic.

	/*
	   for playhead in [mpv_playhead or 0, python_playhead]:
	       if playhead > 0 and (media_duration is None or media_duration >= playhead):
	           return playhead
	*/

	// So if mpv_playhead is 13 and media_duration is 12, it skips 13 and tries python_playhead (2).

	if ph := utils.GetPlayhead(flags, path, startTime, 0, 12); ph != 2 {
		t.Errorf("utils.GetPlayhead (invalid mpv time) = %d; want 2", ph)
	}

	// Use session time only if MPV does not exist
	os.Remove(metadataPath)
	if ph := utils.GetPlayhead(flags, path, startTime, 0, 0); ph != 2 {
		t.Errorf("utils.GetPlayhead (session time) = %d; want 2", ph)
	}

	// Append existing time
	startTime = time.Now().Add(-3 * time.Second)
	if ph := utils.GetPlayhead(flags, path, startTime, 4, 12); ph != 7 {
		t.Errorf("utils.GetPlayhead (existing time) = %d; want 7", ph)
	}
}

func TestMpvArgsToMap(t *testing.T) {
	args := []string{"--volume=50,mute=yes", "--speed=1.5"}
	expected := map[string]string{
		"volume": "50",
		"mute":   "yes",
		"speed":  "1.5",
	}

	actual := utils.MpvArgsToMap(args)
	if len(actual) != len(expected) {
		t.Errorf("utils.MpvArgsToMap len = %d; want %d", len(actual), len(expected))
	}
	for k, v := range expected {
		if actual[k] != v {
			t.Errorf("utils.MpvArgsToMap[%s] = %s; want %s", k, actual[k], v)
		}
	}
}
