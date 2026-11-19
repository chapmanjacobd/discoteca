package commands_test

import (
	"reflect"
	"testing"

	"github.com/chapmanjacobd/discoteca/internal/commands"
)

func TestCommandCatalogIsComplete(t *testing.T) {
	names := map[string]bool{}
	for _, category := range commands.CommandCatalog() {
		if category.Name == "" {
			t.Error("catalog category has an empty name")
		}
		if len(category.Commands) == 0 {
			t.Errorf("catalog category %q has no commands", category.Name)
		}
		for _, spec := range category.Commands {
			if spec.Name == "" {
				t.Errorf("catalog category %q has a command with an empty name", category.Name)
			}
			if names[spec.Name] {
				t.Errorf("command %q appears more than once in the catalog", spec.Name)
			}
			names[spec.Name] = true
		}
	}
	if len(names) == 0 {
		t.Fatal("catalog is empty")
	}
}

func TestNestedCommandsHaveKnownGroups(t *testing.T) {
	groups := map[string]bool{}
	for _, nested := range commands.NestedCommands() {
		if len(nested.Parts) == 0 {
			t.Errorf("command %q has an empty nested path", nested.Command)
			continue
		}
		groups[nested.Parts[0]] = true
	}
	for group := range groups {
		if _, ok := commands.CanonicalNestedGroup(group); !ok {
			t.Errorf("nested path uses unknown group %q", group)
		}
		if len(commands.CommandCatalogInGroup(group)) == 0 && !hasGroupCommand(group) {
			t.Errorf("group %q has no commands", group)
		}
	}
}

func hasGroupCommand(group string) bool {
	for _, nested := range commands.NestedCommands() {
		if len(nested.Parts) == 1 && nested.Parts[0] == group {
			return true
		}
	}
	return false
}

func TestNestedHelpPrefix(t *testing.T) {
	tests := []struct {
		args []string
		want []string
	}{
		{args: []string{"database"}, want: []string{"database"}},
		{args: []string{"database", "import"}, want: []string{"database", "import"}},
		{args: []string{"database", "nope"}, want: []string{"database"}},
		{args: []string{"text", "sort", "nope"}, want: []string{"text", "sort"}},
		{args: []string{"db", "import", "nope"}, want: []string{"database", "import"}},
	}

	for _, tt := range tests {
		got := commands.NestedHelpPrefix(tt.args)
		if !reflect.DeepEqual(got, tt.want) {
			t.Errorf("NestedHelpPrefix(%v) = %v, want %v", tt.args, got, tt.want)
		}
	}
}

func TestCanonicalNestedGroupAliases(t *testing.T) {
	tests := map[string]string{
		"database":  "database",
		"db":        "database",
		"databases": "database",
		"files":     "file",
		"fs":        "filesystem",
		"caption":   "captions",
		"plays":     "play",
	}

	for input, want := range tests {
		got, ok := commands.CanonicalNestedGroup(input)
		if !ok {
			t.Errorf("commands.CanonicalNestedGroup(%q) did not resolve", input)
			continue
		}
		if got != want {
			t.Errorf("commands.CanonicalNestedGroup(%q) = %q, want %q", input, got, want)
		}
	}

	if _, ok := commands.CanonicalNestedGroup("not-a-group"); ok {
		t.Error("CanonicalNestedGroup resolved an unknown group")
	}
}
