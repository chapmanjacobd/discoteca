package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/alecthomas/kong"
	_ "github.com/mattn/go-sqlite3"

	"github.com/chapmanjacobd/discoteca/internal/models"
	"github.com/chapmanjacobd/discoteca/internal/utils"
)

func main() {
	os.Exit(run())
}

func run() int {
	// Setup profiling (only when built with -tags pyroscope)
	cleanup := setupProfiling()
	defer cleanup()

	// Initialize version information
	utils.InitVersionInfo()

	utils.AutoUpdate()
	cli := &CLI{}
	parser, err := kong.New(cli,
		kong.Name("disco"),
		kong.Description("discoteca management tool"),
		kong.UsageOnError(),
	)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to initialize CLI parser: %v\n", err)
		return 1
	}

	args := os.Args[1:]
	// Multitool dispatch: if binary name matches a command, use it as the first argument
	binaryName := filepath.Base(os.Args[0])
	// Strip disco- prefix if present (e.g. disco-add -> add)
	binaryName = strings.TrimPrefix(binaryName, "disco-")

	if binaryName != "disco" && binaryName != "main" && !strings.HasPrefix(binaryName, "go_build_") {
		for _, cmd := range parser.Model.Node.Children {
			if cmd.Name == binaryName {
				args = append([]string{binaryName}, args...)
				break
			}
		}
	}

	ctx, err := parser.Parse(args)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Parse failed: %v\n", err)
		return 1
	}

	// Configure default logger (Warn level by default)
	models.SetupLogging(0)

	err = ctx.Run(context.Background())
	if err != nil {
		models.Log.Error("Command failed", "error", err)
		return 1
	}

	return 0
}
