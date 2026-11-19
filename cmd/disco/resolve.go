package main

import (
	"strings"

	"github.com/alecthomas/kong"

	"github.com/chapmanjacobd/discoteca/internal/commands"
)

// nestedResolution is the result of translating a canonical nested command
// path (for example "filesystem usage") into the flat Kong command name.
type nestedResolution struct {
	Args    []string
	Group   string
	Matched bool
}

// collectFlatNames returns every top-level command name and alias accepted by
// Kong. It lets aliases like "fs" keep working when they also happen to be a
// nested group alias.
func collectFlatNames(parser *kong.Kong) map[string]bool {
	names := make(map[string]bool, len(parser.Model.Children))
	for _, node := range parser.Model.Children {
		names[node.Name] = true
		for _, alias := range node.Aliases {
			names[alias] = true
		}
	}
	return names
}

func hasHelpFlag(args []string) bool {
	for _, arg := range args {
		if arg == "-h" || arg == "--help" {
			return true
		}
	}
	return false
}

func isNestedGroupName(name string) bool {
	_, ok := commands.CanonicalNestedGroup(name)
	return ok
}

func nestedHelpPrefix(args []string) string {
	return strings.Join(commands.NestedHelpPrefix(args), " ")
}

// resolveNestedArgs rewrites a nested command path to its flat Kong command.
//
// Longest paths win so "database history add" resolves to history-add rather
// than to history. When no path matches, Group is set so the caller can show
// group help. Flat commands and aliases take precedence over group help so
// existing spellings such as "disco fs" and "disco play" are unchanged.
func resolveNestedArgs(args []string, flatNames map[string]bool) nestedResolution {
	if len(args) == 0 {
		return nestedResolution{Args: args}
	}

	group, isGroup := commands.CanonicalNestedGroup(args[0])
	if !isGroup {
		return nestedResolution{Args: args}
	}

	bestParts := 0
	bestCommand := ""
	for _, nested := range commands.NestedCommands() {
		if len(nested.Parts) < 1 || nested.Parts[0] != group || len(nested.Parts) > len(args) {
			continue
		}

		matched := true
		for i := 1; i < len(nested.Parts); i++ {
			if commands.CanonicalNestedPart(args[i]) != nested.Parts[i] {
				matched = false
				break
			}
		}
		if matched && len(nested.Parts) > bestParts {
			bestParts = len(nested.Parts)
			bestCommand = nested.Command
		}
	}

	if bestParts == 0 {
		if flatNames[args[0]] {
			return nestedResolution{Args: args}
		}
		return nestedResolution{Args: args, Group: group}
	}

	resolved := make([]string, 0, len(args)-bestParts+1)
	resolved = append(resolved, bestCommand)
	resolved = append(resolved, args[bestParts:]...)
	return nestedResolution{Args: resolved, Group: group, Matched: true}
}

// joinCommandPath is used in error messages for unknown nested commands.
func joinCommandPath(args []string) string {
	return strings.Join(args, " ")
}
