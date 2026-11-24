package commands

import (
	"sort"
	"strings"
)

// CommandSpec maps a Kong command to its canonical nested path.
//
// Path is the preferred user-facing spelling (for example "filesystem usage").
// Name is the flat Kong command name that is still accepted for compatibility.
// An empty Path means the flat Name is already the canonical spelling.
type CommandSpec struct {
	Name string
	Path string
}

// CommandCategory groups commands by workflow for help and README output.
type CommandCategory struct {
	Name     string
	Commands []CommandSpec
}

// NestedCommand is a canonical nested path split into its command parts.
type NestedCommand struct {
	Parts   []string
	Command string
}

var commandCatalog = []CommandCategory{
	{
		Name: "Media collection",
		Commands: []CommandSpec{
			{Name: "add", Path: "database add"},
			{Name: "calibre-import", Path: "database import calibre"},
			{Name: "mpv-watchlater", Path: "database import mpv-watchlater"},
			{Name: "check", Path: "database check"},
		},
	},
	{
		Name: "Playback",
		Commands: []CommandSpec{
			{Name: "watch", Path: "play video"},
			{Name: "listen", Path: "play audio"},
			{Name: "open"},
			{Name: "browse"},
			{Name: "now", Path: "playback now"},
			{Name: "next", Path: "playback next"},
			{Name: "pause", Path: "playback pause"},
			{Name: "seek", Path: "playback seek"},
			{Name: "stop", Path: "playback stop"},
		},
	},
	{
		Name: "Search and reporting",
		Commands: []CommandSpec{
			{Name: "print"},
			{Name: "search"},
			{Name: "search-captions", Path: "captions"},
			{Name: "playlists", Path: "database playlists"},
			{Name: "stats", Path: "database stats"},
			{Name: "history", Path: "database history"},
			{Name: "history-add", Path: "database history add"},
		},
	},
	{
		Name: "Database management",
		Commands: []CommandSpec{
			{Name: "search-db", Path: "database search"},
			{Name: "dedupe", Path: "database dedupe"},
			{Name: "merge-dbs", Path: "database merge"},
			{Name: "optimize", Path: "database optimize"},
			{Name: "repair", Path: "database repair"},
		},
	},
	{
		Name: "File and folder analysis",
		Commands: []CommandSpec{
			{Name: "media-check", Path: "file check"},
			{Name: "sample-hash", Path: "file hash"},
			{Name: "similar-files", Path: "file similar"},
			{Name: "similar-folders", Path: "folder similar"},
			{Name: "big-dirs", Path: "folder large"},
			{Name: "categorize", Path: "folder categorize"},
		},
	},
	{
		Name: "Filesystem inspection",
		Commands: []CommandSpec{
			{Name: "files-info", Path: "filesystem list"},
			{Name: "disk-usage", Path: "filesystem usage"},
		},
	},
	{
		Name: "Text processing",
		Commands: []CommandSpec{
			{Name: "regex-sort", Path: "text sort regex"},
			{Name: "cluster-sort", Path: "text sort cluster"},
		},
	},
	{
		Name: "Web",
		Commands: []CommandSpec{
			{Name: "serve"},
		},
	},
	{
		Name: "Help and maintenance",
		Commands: []CommandSpec{
			{Name: "readme"},
			{Name: "explode"},
			{Name: "update"},
			{Name: "version"},
		},
	},
}

var nestedGroupAliases = map[string]string{
	"db": "database",
	"fs": "filesystem",
}

var nestedPartAliases = map[string]string{
	"ls": "list",
}

// CommandCatalog returns all commands grouped by workflow.
func CommandCatalog() []CommandCategory {
	return commandCatalog
}

// CommandDisplayPath returns the canonical user-facing spelling for a command.
func CommandDisplayPath(spec CommandSpec) string {
	if spec.Path != "" {
		return spec.Path
	}
	return spec.Name
}

// CommandCatalogInGroup returns the commands below a nested path prefix.
func CommandCatalogInGroup(prefix string) []CommandSpec {
	prefixParts := strings.Fields(prefix)
	if len(prefixParts) == 0 {
		return nil
	}

	var specs []CommandSpec
	for _, category := range commandCatalog {
		for _, spec := range category.Commands {
			parts := strings.Fields(CommandDisplayPath(spec))
			if len(parts) <= len(prefixParts) {
				continue
			}
			matches := true
			for i, part := range prefixParts {
				if parts[i] != part {
					matches = false
					break
				}
			}
			if matches {
				specs = append(specs, spec)
			}
		}
	}
	sort.Slice(specs, func(i, j int) bool {
		return CommandDisplayPath(specs[i]) < CommandDisplayPath(specs[j])
	})
	return specs
}

// NestedHelpPrefix returns the longest canonical nested path prefix found in
// args. It is used to show help for intermediate groups such as
// "database import" or "text sort".
func NestedHelpPrefix(args []string) []string {
	if len(args) == 0 {
		return nil
	}

	group, ok := CanonicalNestedGroup(args[0])
	if !ok {
		return nil
	}

	best := []string{group}
	for _, nested := range NestedCommands() {
		if nested.Parts[0] != group {
			continue
		}
		limit := min(len(args), len(nested.Parts))
		for length := limit; length >= 2; length-- {
			matched := true
			for i := 1; i < length; i++ {
				if CanonicalNestedPart(args[i]) != nested.Parts[i] {
					matched = false
					break
				}
			}
			if matched && length > len(best) {
				best = append([]string{}, nested.Parts[:length]...)
				break
			}
		}
	}
	return best
}

// NestedCommands returns every command that has a canonical nested path.
func NestedCommands() []NestedCommand {
	var nested []NestedCommand
	for _, category := range commandCatalog {
		for _, spec := range category.Commands {
			if spec.Path == "" {
				continue
			}
			nested = append(nested, NestedCommand{
				Parts:   strings.Fields(spec.Path),
				Command: spec.Name,
			})
		}
	}
	return nested
}

// CanonicalNestedGroup resolves a group name or alias to its canonical group.
func CanonicalNestedGroup(name string) (string, bool) {
	groups := make(map[string]struct{})
	for _, category := range commandCatalog {
		for _, spec := range category.Commands {
			if spec.Path == "" {
				continue
			}
			parts := strings.Fields(spec.Path)
			if len(parts) > 0 {
				groups[parts[0]] = struct{}{}
			}
		}
	}

	if _, ok := groups[name]; ok {
		return name, true
	}
	if target, ok := nestedGroupAliases[name]; ok {
		if _, exists := groups[target]; exists {
			return target, true
		}
	}
	for group := range groups {
		if trimmed, found := strings.CutSuffix(group, "s"); found {
			if name == trimmed {
				return group, true
			}
		} else if name == group+"s" {
			return group, true
		}
	}
	return "", false
}

// NestedGroupAliases returns the aliases for a canonical nested group.
func NestedGroupAliases(group string) []string {
	var aliases []string
	seen := map[string]bool{}
	for alias, target := range nestedGroupAliases {
		if target == group {
			aliases = append(aliases, alias)
			seen[alias] = true
		}
	}
	if alias, found := strings.CutSuffix(group, "s"); found {
		if !seen[alias] {
			aliases = append(aliases, alias)
		}
	} else {
		alias := group + "s"
		if !seen[alias] {
			aliases = append(aliases, alias)
		}
	}
	sort.Strings(aliases)
	return aliases
}

// CanonicalNestedPart resolves aliases for components inside a nested path.
func CanonicalNestedPart(part string) string {
	if canonical, ok := nestedPartAliases[part]; ok {
		return canonical
	}
	return part
}
