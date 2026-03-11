package main

import (
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/alecthomas/kong"
	"github.com/chapmanjacobd/discoteca/internal/models"
	"github.com/chapmanjacobd/discoteca/internal/utils"
	_ "github.com/mattn/go-sqlite3"
)

func main() {
	utils.AutoUpdate()
	cli := &CLI{}
	parser, err := kong.New(cli,
		kong.Name("disco"),
		kong.Description("discoteca management tool"),
		kong.UsageOnError(),
	)
	if err != nil {
		panic(err)
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
		parser.FatalIfErrorf(err)
	}

	// Configure default logger
	// Subcommands will call models.SetupLogging to update models.LogLevel if needed
	logger := slog.New(&utils.PlainHandler{
		Level: models.LogLevel,
		Out:   os.Stderr,
	})
	slog.SetDefault(logger)

	err = ctx.Run()
	if err != nil {
		slog.Error("Command failed", "error", err)
		os.Exit(1)
	}
}
