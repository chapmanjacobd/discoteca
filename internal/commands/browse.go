package commands

import (
	"context"
	"errors"
	"os/exec"
	"runtime"
	"strings"

	"github.com/chapmanjacobd/discoteca/internal/models"
	"github.com/chapmanjacobd/discoteca/internal/query"
	"github.com/chapmanjacobd/discoteca/internal/utils"
)

type OpenCmd struct {
	models.CoreFlags        `embed:""`
	models.PathFilterFlags  `embed:""`
	models.FilterFlags      `embed:""`
	models.MediaFilterFlags `embed:""`
	models.SortFlags        `embed:""`
	models.PostActionFlags  `embed:""`

	Databases []string `help:"SQLite database files" required:"" arg:"" type:"existingfile"`
}

func (c *OpenCmd) Run(ctx context.Context) error {
	models.SetupLogging(c.Verbose)
	flags := models.GlobalFlags{
		CoreFlags:        c.CoreFlags,
		PathFilterFlags:  c.PathFilterFlags,
		FilterFlags:      c.FilterFlags,
		MediaFilterFlags: c.MediaFilterFlags,
		SortFlags:        c.SortFlags,
		PostActionFlags:  c.PostActionFlags,
	}
	media, err := query.MediaQuery(ctx, c.Databases, flags)
	if err != nil {
		return err
	}

	media = query.FilterMedia(media, flags)

	for _, m := range media {
		if !utils.FileExists(m.Path) {
			continue
		}

		var cmd *exec.Cmd
		switch runtime.GOOS {
		case "linux":
			cmd = exec.CommandContext(ctx, "xdg-open", m.Path)
		case "darwin":
			cmd = exec.CommandContext(ctx, "open", m.Path)
		case "windows":
			cmd = exec.CommandContext(ctx, "cmd", "/c", "start", m.Path)
		}

		if err := cmd.Start(); err != nil {
			return err
		}
	}

	return ExecutePostAction(flags, media)
}

type BrowseCmd struct {
	models.CoreFlags        `embed:""`
	models.PathFilterFlags  `embed:""`
	models.FilterFlags      `embed:""`
	models.MediaFilterFlags `embed:""`
	models.SortFlags        `embed:""`

	Databases []string `help:"SQLite database files" required:"" arg:"" type:"existingfile"`
	Browser   string   `help:"Browser to use"`
}

func (c *BrowseCmd) Run(ctx context.Context) error {
	models.SetupLogging(c.Verbose)
	flags := models.GlobalFlags{
		CoreFlags:        c.CoreFlags,
		PathFilterFlags:  c.PathFilterFlags,
		FilterFlags:      c.FilterFlags,
		MediaFilterFlags: c.MediaFilterFlags,
		SortFlags:        c.SortFlags,
	}
	media, err := query.MediaQuery(ctx, c.Databases, flags)
	if err != nil {
		return err
	}

	media = query.FilterMedia(media, flags)

	browser := c.Browser
	if browser == "" {
		browser = utils.GetDefaultBrowser()
	}

	var urls []string
	for _, m := range media {
		if strings.HasPrefix(m.Path, "http") {
			urls = append(urls, m.Path)
		}
	}

	if len(urls) == 0 {
		return errors.New("no URLs found")
	}

	var cmd *exec.Cmd
	if runtime.GOOS == "windows" && browser == "cmd" {
		// On Windows, use 'cmd /c start' to open URLs
		args := append([]string{"/c", "start"}, urls...)
		cmd = exec.CommandContext(ctx, "cmd", args...)
	} else {
		args := append([]string{browser}, urls...)
		cmd = exec.CommandContext(ctx, args[0], args[1:]...)
	}
	return cmd.Start()
}
