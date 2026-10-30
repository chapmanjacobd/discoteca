package media

import (
	"bufio"
	"context"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/chapmanjacobd/discoteca/internal/config"
	"github.com/chapmanjacobd/discoteca/internal/logging"
	"github.com/chapmanjacobd/discoteca/internal/models"
	"github.com/chapmanjacobd/discoteca/internal/utils"
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
	return config.GetMpvWatchSocket()
}

// CastCommand builds and executes a catt command with optional device specification
func CastCommand(ctx context.Context, castDevice string, args ...string) error {
	cmdArgs := []string{"catt"}
	if castDevice != "" {
		cmdArgs = append(cmdArgs, "-d", castDevice)
	}
	cmdArgs = append(cmdArgs, args...)
	cmd := exec.CommandContext(ctx, cmdArgs[0], cmdArgs[1:]...)
	return cmd.Run()
}

// MpvCall sends a command to mpv via IPC socket and returns the response
func MpvCall(ctx context.Context, socketPath string, args ...any) (*MpvResponse, error) {
	dialCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	conn, err := (&net.Dialer{}).DialContext(dialCtx, "unix", socketPath)
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

// PathToMpvWatchLaterMD5 returns the MD5 hash of the absolute path, which mpv uses for filenames
func PathToMpvWatchLaterMD5(path string) string {
	abs := path
	isRootRelative := strings.HasPrefix(path, "/") || strings.HasPrefix(path, "\\")
	if !isRootRelative && !filepath.IsAbs(path) {
		if a, err := filepath.Abs(path); err == nil {
			abs = a
		}
	}
	// mpv uses forward slashes even on Windows for its MD5 hash
	slashPath := strings.ReplaceAll(abs, "\\", "/")
	hash := md5.Sum([]byte(slashPath))
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
func GetPlayhead(flags models.GlobalFlags, path string, startTime time.Time, existingPlayhead, mediaDuration int) int {
	endTime := time.Now()
	sessionDuration := int(endTime.Sub(startTime).Seconds())
	pythonPlayhead := sessionDuration
	if existingPlayhead > 0 {
		pythonPlayhead += existingPlayhead
	}

	watchLaterDir := flags.WatchLaterDir
	if watchLaterDir == "" {
		watchLaterDir = config.GetMpvWatchLaterDir()
	}

	md5Hash := PathToMpvWatchLaterMD5(path)
	metadataPath := filepath.Join(watchLaterDir, md5Hash)

	mpvPlayhead := 0
	val, err := MpvWatchLaterValue(metadataPath, "start")
	if err == nil && val != "" {
		// val is likely a float string like "5.000000"
		if f := utils.SafeFloat(val); f != nil {
			mpvPlayhead = int(*f)
		}
	}

	logging.Log.Debug("playhead check", "mpv", mpvPlayhead, "session", pythonPlayhead, "path", path)

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
