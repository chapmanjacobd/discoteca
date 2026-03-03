package utils

import (
	"bufio"
	"encoding/json"
	"net"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/chapmanjacobd/discotheque/internal/models"
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
					var cmd MpvCommand
					json.Unmarshal(scanner.Bytes(), &cmd)

					var resp MpvResponse
					if len(cmd.Command) > 0 && cmd.Command[0] == "get_property" {
						resp = MpvResponse{Data: 50, Error: "success"}
					} else if len(cmd.Command) > 0 && cmd.Command[0] == "error" {
						resp = MpvResponse{Error: "invalid property"}
					} else {
						resp = MpvResponse{Data: "pong", Error: "success"}
					}
					jsonData, _ := json.Marshal(resp)
					c.Write(append(jsonData, '\n'))
				}
			}(conn)
		}
	}()

	resp, err := MpvCall(socketPath, "ping")
	if err != nil {
		t.Fatalf("MpvCall failed: %v", err)
	}
	if resp.Data != "pong" {
		t.Errorf("Expected pong, got %v", resp.Data)
	}

	val, err := MpvGetProperty(socketPath, "volume")
	if err != nil {
		t.Fatalf("MpvGetProperty failed: %v", err)
	}
	if val.(float64) != 50 {
		t.Errorf("Expected 50, got %v", val)
	}

	err = MpvSetProperty(socketPath, "volume", 60)
	if err != nil {
		t.Fatalf("MpvSetProperty failed: %v", err)
	}

	err = MpvPause(socketPath, true)
	if err != nil {
		t.Fatalf("MpvPause failed: %v", err)
	}

	err = MpvSeek(socketPath, 10, "relative")
	if err != nil {
		t.Fatalf("MpvSeek failed: %v", err)
	}

	err = MpvLoadFile(socketPath, "file.mp4", "replace")
	if err != nil {
		t.Fatalf("MpvLoadFile failed: %v", err)
	}

	_, err = MpvCall(socketPath, "error")
	if err == nil {
		t.Error("Expected error for 'error' command, got nil")
	}
}

func TestMpvWatchLaterValue(t *testing.T) {
	f, _ := os.CreateTemp("", "watch-later")
	defer os.Remove(f.Name())
	f.WriteString("key1=val1\nkey2=val2\n")
	f.Close()

	val, err := MpvWatchLaterValue(f.Name(), "key2")
	if err != nil {
		t.Fatal(err)
	}
	if val != "val2" {
		t.Errorf("Expected val2, got %s", val)
	}

	val, _ = MpvWatchLaterValue(f.Name(), "missing")
	if val != "" {
		t.Errorf("Expected empty string for missing key, got %s", val)
	}
}

func TestPathToMpvWatchLaterMD5(t *testing.T) {
	path := "/home/xk/github/xk/lb/tests/data/test.mp4"
	expected := "E1E0D0E3F0D2CB748303FDA43224B7E7"

	actual := PathToMpvWatchLaterMD5(path)
	if actual != expected {
		t.Errorf("PathToMpvWatchLaterMD5(%s) = %s; want %s", path, actual, expected)
	}
}

func TestGetPlayhead(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "mpv_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	flags := models.GlobalFlags{
		PlaybackFlags: models.PlaybackFlags{
			WatchLaterDir: tmpDir,
		},
	}
	path := "/home/runner/work/library/library/tests/data/test.mp4"
	md5Hash := PathToMpvWatchLaterMD5(path)
	metadataPath := filepath.Join(tmpDir, md5Hash)

	// Use MPV time
	startTime := time.Now().Add(-2 * time.Second)
	os.WriteFile(metadataPath, []byte("start=5.000000\n"), 0o644)
	if ph := GetPlayhead(flags, path, startTime, 0, 0); ph != 5 {
		t.Errorf("GetPlayhead (mpv time) = %d; want 5", ph)
	}

	// Check invalid MPV time (beyond duration)
	os.WriteFile(metadataPath, []byte("start=13.000000\n"), 0o644)
	// GetPlayhead currently returns mpvPlayhead if found.
	// The Python code:
	//   if mpv_playhead: return mpv_playhead
	// Wait, I should re-read Python's get_playhead logic.

	/*
	   for playhead in [mpv_playhead or 0, python_playhead]:
	       if playhead > 0 and (media_duration is None or media_duration >= playhead):
	           return playhead
	*/

	// So if mpv_playhead is 13 and media_duration is 12, it skips 13 and tries python_playhead (2).

	if ph := GetPlayhead(flags, path, startTime, 0, 12); ph != 2 {
		t.Errorf("GetPlayhead (invalid mpv time) = %d; want 2", ph)
	}

	// Use session time only if MPV does not exist
	os.Remove(metadataPath)
	if ph := GetPlayhead(flags, path, startTime, 0, 0); ph != 2 {
		t.Errorf("GetPlayhead (session time) = %d; want 2", ph)
	}

	// Append existing time
	startTime = time.Now().Add(-3 * time.Second)
	if ph := GetPlayhead(flags, path, startTime, 4, 12); ph != 7 {
		t.Errorf("GetPlayhead (existing time) = %d; want 7", ph)
	}
}

func TestMpvArgsToMap(t *testing.T) {
	args := []string{"--volume=50,mute=yes", "--speed=1.5"}
	expected := map[string]string{
		"volume": "50",
		"mute":   "yes",
		"speed":  "1.5",
	}

	actual := MpvArgsToMap(args)
	if len(actual) != len(expected) {
		t.Errorf("MpvArgsToMap len = %d; want %d", len(actual), len(expected))
	}
	for k, v := range expected {
		if actual[k] != v {
			t.Errorf("MpvArgsToMap[%s] = %s; want %s", k, actual[k], v)
		}
	}
}
