package commands_test

import (
	"bufio"
	"context"
	"encoding/json"
	"net"
	"path/filepath"
	"strings"
	"testing"

	"github.com/chapmanjacobd/discoteca/internal/commands"
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

	t.Run("commands.NowCmd", func(t *testing.T) {
		cmd := &commands.NowCmd{
			MpvControlBase: commands.MpvControlBase{ControlFlags: baseFlags},
		}
		if err := cmd.Run(context.Background()); err != nil {
			t.Errorf("commands.NowCmd failed: %v", err)
		}
	})

	t.Run("commands.PauseCmd", func(t *testing.T) {
		cmd := &commands.PauseCmd{
			MpvControlBase: commands.MpvControlBase{ControlFlags: baseFlags},
		}
		if err := cmd.Run(context.Background()); err != nil {
			t.Errorf("commands.PauseCmd failed: %v", err)
		}
	})

	t.Run("commands.NextCmd", func(t *testing.T) {
		cmd := &commands.NextCmd{
			MpvControlBase: commands.MpvControlBase{ControlFlags: baseFlags},
		}
		if err := cmd.Run(context.Background()); err != nil {
			t.Errorf("commands.NextCmd failed: %v", err)
		}
	})

	t.Run("commands.StopCmd", func(t *testing.T) {
		cmd := &commands.StopCmd{
			MpvControlBase: commands.MpvControlBase{ControlFlags: baseFlags},
		}
		if err := cmd.Run(context.Background()); err != nil {
			t.Errorf("commands.StopCmd failed: %v", err)
		}
	})

	t.Run("commands.SeekCmd", func(t *testing.T) {
		cmd := &commands.SeekCmd{
			MpvControlBase: commands.MpvControlBase{ControlFlags: baseFlags},
			Time:           "+10",
		}
		if err := cmd.Run(context.Background()); err != nil {
			t.Errorf("commands.SeekCmd failed: %v", err)
		}

		cmd.Time = "00:01:00"
		if err := cmd.Run(context.Background()); err != nil {
			t.Errorf("commands.SeekCmd failed: %v", err)
		}
	})
}
