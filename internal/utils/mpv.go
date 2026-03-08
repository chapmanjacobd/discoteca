package utils

import (
	"bufio"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log/slog"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/chapmanjacobd/discotheque/internal/models"
)

type MpvResponse struct {
	Data  any    `json:"data"`
	Error string `json:"error"`
	ID    int    `json:"request_id"`
}

type MpvCommand struct {
	Command []any `json:"command"`
}

// GetMpvSocketPath returns the socket path to use, either from provided value or default
func GetMpvSocketPath(provided string) string {
	if provided != "" {
		return provided
	}
	return GetMpvWatchSocket()
}

// CastCommand builds and executes a catt command with optional device specification
func CastCommand(castDevice string, args ...string) error {
	cmdArgs := []string{"catt"}
	if castDevice != "" {
		cmdArgs = append(cmdArgs, "-d", castDevice)
	}
	cmdArgs = append(cmdArgs, args...)
	cmd := exec.Command(cmdArgs[0], cmdArgs[1:]...)
	return cmd.Run()
}

// MpvCall sends a command to mpv via IPC socket and returns the response
func MpvCall(socketPath string, args ...any) (*MpvResponse, error) {
	conn, err := net.DialTimeout("unix", socketPath, 2*time.Second)
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	cmd := MpvCommand{Command: args}
	jsonData, err := json.Marshal(cmd)
	if err != nil {
		return nil, err
	}

	_, err = conn.Write(append(jsonData, '\n'))
	if err != nil {
		return nil, err
	}

	// Read response
	if err := conn.SetReadDeadline(time.Now().Add(2 * time.Second)); err != nil {
		return nil, err
	}

	scanner := bufio.NewScanner(conn)
	if scanner.Scan() {
		var resp MpvResponse
		if err := json.Unmarshal(scanner.Bytes(), &resp); err != nil {
			return nil, err
		}
		if resp.Error != "" && resp.Error != "success" {
			return &resp, fmt.Errorf("mpv error: %s", resp.Error)
		}
		return &resp, nil
	}

	return nil, scanner.Err()
}

// MpvSetProperty sets a property in mpv
func MpvSetProperty(socketPath string, name string, value any) error {
	_, err := MpvCall(socketPath, "set_property", name, value)
	return err
}

// MpvGetProperty gets a property from mpv
func MpvGetProperty(socketPath string, name string) (any, error) {
	resp, err := MpvCall(socketPath, "get_property", name)
	if err != nil {
		return nil, err
	}
	return resp.Data, nil
}

// MpvPause pauses or unpauses playback
func MpvPause(socketPath string, pause bool) error {
	return MpvSetProperty(socketPath, "pause", pause)
}

// MpvSeek seeks to a specific position
func MpvSeek(socketPath string, value float64, mode string) error {
	// mode can be "relative", "absolute", "absolute-percent", "relative-percent"
	if mode == "" {
		mode = "relative"
	}
	_, err := MpvCall(socketPath, "seek", value, mode)
	return err
}

// MpvLoadFile loads a file into mpv
func MpvLoadFile(socketPath string, path string, mode string) error {
	// mode can be "replace", "append", "append-play"
	if mode == "" {
		mode = "replace"
	}
	_, err := MpvCall(socketPath, "loadfile", path, mode)
	return err
}

// PathToMpvWatchLaterMD5 returns the MD5 hash of the absolute path as used by mpv
func PathToMpvWatchLaterMD5(path string) string {
	abs, err := filepath.Abs(path)
	if err != nil {
		abs = path
	}
	hash := md5.Sum([]byte(abs))
	return strings.ToUpper(hex.EncodeToString(hash[:]))
}

// MpvWatchLaterValue reads a value for a given key from an mpv watch_later file
func MpvWatchLaterValue(path, key string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if after, ok := strings.CutPrefix(line, key+"="); ok {
			return after, nil
		}
	}

	if err := scanner.Err(); err != nil {
		return "", err
	}

	return "", nil
}

// GetPlayhead calculates the playhead position based on session duration, existing playhead and mpv watch_later
func GetPlayhead(flags models.GlobalFlags, path string, startTime time.Time, existingPlayhead int, mediaDuration int) int {
	endTime := time.Now()
	sessionDuration := int(endTime.Sub(startTime).Seconds())
	pythonPlayhead := sessionDuration
	if existingPlayhead > 0 {
		pythonPlayhead += existingPlayhead
	}

	watchLaterDir := flags.WatchLaterDir
	if watchLaterDir == "" {
		watchLaterDir = GetMpvWatchLaterDir()
	}

	md5Hash := PathToMpvWatchLaterMD5(path)
	metadataPath := filepath.Join(watchLaterDir, md5Hash)

	mpvPlayhead := 0
	val, err := MpvWatchLaterValue(metadataPath, "start")
	if err == nil && val != "" {
		// val is likely a float string like "5.000000"
		if f := SafeFloat(val); f != nil {
			mpvPlayhead = int(*f)
		}
	}

	slog.Debug("playhead check", "mpv", mpvPlayhead, "session", pythonPlayhead, "path", path)

	// Prefer mpv playhead if it's within bounds
	if mpvPlayhead > 0 && (mediaDuration <= 0 || mpvPlayhead <= mediaDuration) {
		return mpvPlayhead
	}

	// Fallback to session-based playhead if it's within bounds
	if pythonPlayhead > 0 && (mediaDuration <= 0 || pythonPlayhead <= mediaDuration) {
		return pythonPlayhead
	}

	return 0
}

// MpvArgsToMap parses mpv command line arguments into a map
func MpvArgsToMap(argStrings []string) map[string]string {
	argMap := make(map[string]string)
	for _, s := range argStrings {
		for arg := range strings.SplitSeq(s, ",") {
			parts := strings.SplitN(strings.TrimLeft(arg, "-"), "=", 2)
			if len(parts) == 2 {
				argMap[parts[0]] = parts[1]
			}
		}
	}
	return argMap
}
