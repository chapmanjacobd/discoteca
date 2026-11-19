package main

import (
	"fmt"
	"io"
	"strings"
	"text/tabwriter"

	"github.com/alecthomas/kong"

	"github.com/chapmanjacobd/discoteca/internal/commands"
)

func commandNodes(parser *kong.Kong) map[string]*kong.Node {
	nodes := make(map[string]*kong.Node, len(parser.Model.Children))
	for _, node := range parser.Model.Children {
		if node.Hidden {
			continue
		}
		nodes[node.Name] = node
	}
	return nodes
}

// printCommandOverview prints top-level help grouped by workflow.
func printCommandOverview(w io.Writer, parser *kong.Kong) {
	nodes := commandNodes(parser)

	fmt.Fprintln(w, "Usage: disco <command>")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "discoteca management tool")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Flags:")
	fmt.Fprintln(w, "  -h, --help    Show context-sensitive help.")

	for _, category := range commands.CommandCatalog() {
		fmt.Fprintf(w, "\n%s:\n", category.Name)
		tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
		for _, spec := range category.Commands {
			node := nodes[spec.Name]
			if node == nil {
				continue
			}
			fmt.Fprintf(tw, "  %s\t%s\n", commands.CommandDisplayPath(spec), node.Help)
		}
		_ = tw.Flush()
	}

	fmt.Fprintln(w, "\nRun \"disco <command> --help\" for more information on a command.")
}

// printGroupHelp prints the commands below one nested group.
func printGroupHelp(w io.Writer, parser *kong.Kong, group string) {
	nodes := commandNodes(parser)

	fmt.Fprintf(w, "Usage: disco %s <command>\n", group)
	if !strings.Contains(group, " ") {
		if aliases := commands.NestedGroupAliases(group); len(aliases) > 0 {
			fmt.Fprintf(w, "Aliases: %s\n", strings.Join(aliases, ", "))
		}
	}
	fmt.Fprintln(w)

	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	for _, spec := range commands.CommandCatalogInGroup(group) {
		node := nodes[spec.Name]
		if node == nil {
			continue
		}
		path := commands.CommandDisplayPath(spec)
		fmt.Fprintf(tw, "  %s\t%s\n", strings.TrimPrefix(path, group+" "), node.Help)
	}
	_ = tw.Flush()

	fmt.Fprintf(w, "\nRun \"disco %s <command> --help\" for more information on a command.\n", group)
}
