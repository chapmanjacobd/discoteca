package commands

import (
	"bufio"
	"encoding/json"
	"net"
	"path/filepath"
	"strings"
	"testing"

	"github.com/chapmanjacobd/discoteca/internal/models"
	"github.com/chapmanjacobd/discoteca/internal/utils"
)

func startMockMpvServer(t *testing.T) string {
	socketPath := filepath.Join(t.TempDir(), "mpv.sock")
	ln, err := net.Listen("unix", socketPath)
	if err != nil {
		t.Fatal(err)
	}

	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				return
			}
			go func(c net.Conn) {
				defer c.Close()
				scanner := bufio.NewScanner(c)
				if scanner.Scan() {
					line := scanner.Text()
					resp := utils.MpvResponse{}
					if strings.Contains(line, "time-pos") {
						resp.Data = 10.5
					} else if strings.Contains(line, "duration") {
						resp.Data = 100.0
					} else if strings.Contains(line, "path") {
						resp.Data = "/path/to/media.mp4"
					} else {
						resp.Data = "ok"
					}

					jsonData, _ := json.Marshal(resp)
					c.Write(append(jsonData, '\n'))
				}
			}(conn)
		}
	}()

	return socketPath
}

func TestPlaybackControlCmds(t *testing.T) {
	socketPath := startMockMpvServer(t)

	baseFlags := models.ControlFlags{
		MpvSocket: socketPath,
	}

	t.Run("NowCmd", func(t *testing.T) {
		cmd := &NowCmd{
			MpvControlBase: MpvControlBase{ControlFlags: baseFlags},
		}
		if err := cmd.Run(nil); err != nil {
			t.Errorf("NowCmd failed: %v", err)
		}
	})

	t.Run("PauseCmd", func(t *testing.T) {
		cmd := &PauseCmd{
			MpvControlBase: MpvControlBase{ControlFlags: baseFlags},
		}
		if err := cmd.Run(nil); err != nil {
			t.Errorf("PauseCmd failed: %v", err)
		}
	})

	t.Run("NextCmd", func(t *testing.T) {
		cmd := &NextCmd{
			MpvControlBase: MpvControlBase{ControlFlags: baseFlags},
		}
		if err := cmd.Run(nil); err != nil {
			t.Errorf("NextCmd failed: %v", err)
		}
	})

	t.Run("StopCmd", func(t *testing.T) {
		cmd := &StopCmd{
			MpvControlBase: MpvControlBase{ControlFlags: baseFlags},
		}
		if err := cmd.Run(nil); err != nil {
			t.Errorf("StopCmd failed: %v", err)
		}
	})

	t.Run("SeekCmd", func(t *testing.T) {
		cmd := &SeekCmd{
			MpvControlBase: MpvControlBase{ControlFlags: baseFlags},
			Time:           "+10",
		}
		if err := cmd.Run(nil); err != nil {
			t.Errorf("SeekCmd failed: %v", err)
		}

		cmd.Time = "00:01:00"
		if err := cmd.Run(nil); err != nil {
			t.Errorf("SeekCmd failed: %v", err)
		}
	})
}
