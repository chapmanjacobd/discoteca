package main

import (
	"testing"

	"github.com/chapmanjacobd/discoteca/internal/commands"
)

func TestCommandCatalogMatchesCLI(t *testing.T) {
	parser := newTestParser(t)
	nodes := commandNodes(parser)

	catalogNames := map[string]bool{}
	for _, category := range commands.CommandCatalog() {
		for _, spec := range category.Commands {
			catalogNames[spec.Name] = true
			if nodes[spec.Name] == nil {
				t.Errorf("catalog command %q has no CLI node", spec.Name)
			}
		}
	}

	for _, node := range parser.Model.Children {
		if node.Hidden {
			continue
		}
		if !catalogNames[node.Name] {
			t.Errorf("CLI command %q is missing from the catalog", node.Name)
		}
	}
}
