package media_test

import (
	"bufio"
	"context"
	"encoding/json"
	"net"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/chapmanjacobd/discoteca/internal/media"
	"github.com/chapmanjacobd/discoteca/internal/models"
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
			conn, err2 := ln.Accept()
			if err2 != nil {
				return
			}
			go func(c net.Conn) {
				defer c.Close()
				scanner := bufio.NewScanner(c)
				for scanner.Scan() {
					var cmd media.MpvCommand
					json.Unmarshal(scanner.Bytes(), &cmd)

					var resp media.MpvResponse
					if len(cmd.Command) > 0 && cmd.Command[0] == "get_property" {
						resp = media.MpvResponse{Data: 50, Error: "success"}
					} else if len(cmd.Command) > 0 && cmd.Command[0] == "error" {
						resp = media.MpvResponse{Error: "invalid property"}
					} else {
						resp = media.MpvResponse{Data: "pong", Error: "success"}
					}
					jsonData, err3 := json.Marshal(resp)
					if err3 != nil {
						t.Logf("json.Marshal failed: %v", err3)
						return
					}
					c.Write(append(jsonData, '\n'))
				}
			}(conn)
		}
	}()

	resp, err := media.MpvCall(context.Background(), socketPath, "ping")
	if err != nil {
		t.Fatalf("media.MpvCall failed: %v", err)
	}
	if resp.Data != "pong" {
		t.Errorf("Expected pong, got %v", resp.Data)
	}

	_, err = media.MpvCall(context.Background(), socketPath, "error")
	if err == nil {
		t.Error("Expected error for 'error' command, got nil")
	}
}

func TestMpvWatchLaterValue(t *testing.T) {
	f, _ := os.CreateTemp(t.TempDir(), "watch-later")
	defer os.Remove(f.Name())
	f.WriteString("key1=val1\nkey2=val2\n")
	f.Close()

	val, err := media.MpvWatchLaterValue(f.Name(), "key2")
	if err != nil {
		t.Fatal(err)
	}
	if val != "val2" {
		t.Errorf("Expected val2, got %s", val)
	}

	val, _ = media.MpvWatchLaterValue(f.Name(), "missing")
	if val != "" {
		t.Errorf("Expected empty string for missing key, got %s", val)
	}
}

func TestPathToMpvWatchLaterMD5(t *testing.T) {
	// We want to test that it produces the SAME hash as mpv would for this path.
	// mpv uses forward slashes for the hash even on Windows.
	path := "/home/xk/github/xk/lb/tests/data/test.mp4"
	// The function internals should NOT use filepath.Abs if we provide an absolute-looking path starting with "/"
	got := media.PathToMpvWatchLaterMD5(path)
	want := "E1E0D0E3F0D2CB748303FDA43224B7E7"
	if got != want {
		t.Errorf("media.PathToMpvWatchLaterMD5(%s) = %s; want %s", path, got, want)
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
	md5Hash := media.PathToMpvWatchLaterMD5(path)
	metadataPath := filepath.Join(tmpDir, md5Hash)

	// Use MPV time
	startTime := time.Now().Add(-2 * time.Second)
	os.WriteFile(metadataPath, []byte("start=5.000000\n"), 0o644)
	if ph := media.GetPlayhead(flags, path, startTime, 0, 0); ph != 5 {
		t.Errorf("media.GetPlayhead (mpv time) = %d; want 5", ph)
	}

	// Check invalid MPV time (beyond duration)
	os.WriteFile(metadataPath, []byte("start=13.000000\n"), 0o644)
	// media.GetPlayhead currently returns mpvPlayhead if found.
	// The Python code:
	//   if mpv_playhead: return mpv_playhead
	// Wait, I should re-read Python's get_playhead logic.

	/*
	   for playhead in [mpv_playhead or 0, python_playhead]:
	       if playhead > 0 and (media_duration is None or media_duration >= playhead):
	           return playhead
	*/

	// So if mpv_playhead is 13 and media_duration is 12, it skips 13 and tries python_playhead (2).

	if ph := media.GetPlayhead(flags, path, startTime, 0, 12); ph != 2 {
		t.Errorf("media.GetPlayhead (invalid mpv time) = %d; want 2", ph)
	}

	// Use session time only if MPV does not exist
	os.Remove(metadataPath)
	if ph := media.GetPlayhead(flags, path, startTime, 0, 0); ph != 2 {
		t.Errorf("media.GetPlayhead (session time) = %d; want 2", ph)
	}

	// Append existing time
	startTime = time.Now().Add(-3 * time.Second)
	if ph := media.GetPlayhead(flags, path, startTime, 4, 12); ph != 7 {
		t.Errorf("media.GetPlayhead (existing time) = %d; want 7", ph)
	}
}
