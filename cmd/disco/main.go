package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/alecthomas/kong"
	_ "github.com/mattn/go-sqlite3"

	"github.com/chapmanjacobd/discoteca/internal/logging"
	"github.com/chapmanjacobd/discoteca/internal/selfupdate"
)

func main() {
	os.Exit(run())
}

func run() int {
	// Setup profiling (only when built with -tags pyroscope)
	cleanup := setupProfiling()
	defer cleanup()

	// Initialize version information
	selfupdate.InitVersionInfo()

	selfupdate.AutoUpdate()
	cli := &CLI{}
	parser, err := kong.New(cli,
		kong.Name("disco"),
		kong.Description("discoteca management tool"),
		kong.UsageOnError(),
		kong.BindTo(context.Background(), (*context.Context)(nil)),
	)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to initialize CLI parser: %v\n", err)
		return 1
	}

	args := os.Args[1:]
	flatNames := collectFlatNames(parser)

	// Multitool dispatch: if binary name matches a command or nested group,
	// use it as the first argument.
	binaryName := filepath.Base(os.Args[0])
	// Strip disco- prefix if present (e.g. disco-add -> add)
	binaryName = strings.TrimPrefix(binaryName, "disco-")

	if binaryName != "disco" && binaryName != "main" && !strings.HasPrefix(binaryName, "go_build_") {
		if flatNames[binaryName] || isNestedGroupName(binaryName) {
			args = append([]string{binaryName}, args...)
		}
	}

	resolved := resolveNestedArgs(args, flatNames)
	if resolved.Matched {
		args = resolved.Args
	} else if resolved.Group != "" {
		group := nestedHelpPrefix(args)
		if group == "" {
			group = resolved.Group
		}
		switch {
		case hasHelpFlag(args[1:]):
			printGroupHelp(os.Stdout, parser, group)
			return 0
		case len(args) == 1:
			printGroupHelp(os.Stderr, parser, group)
			return 1
		default:
			fmt.Fprintf(os.Stderr, "Unknown command: disco %s\n\n", joinCommandPath(args))
			printGroupHelp(os.Stderr, parser, group)
			return 1
		}
	}

	if len(args) == 0 {
		printCommandOverview(os.Stderr, parser)
		return 1
	}
	if len(args) == 1 && hasHelpFlag(args) {
		printCommandOverview(os.Stdout, parser)
		return 0
	}

	ctx, err := parser.Parse(args)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Parse failed: %v\n", err)
		return 1
	}

	// Configure default logger (Warn level by default)
	logging.SetupLogging(0)

	err = ctx.Run()
	if err != nil {
		logging.Log.Error("Command failed", "error", err)
		return 1
	}

	return 0
}
