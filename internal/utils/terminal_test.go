package utils_test

import (
	"testing"

	"github.com/chapmanjacobd/discoteca/internal/utils"
)

func TestTruncateMiddle(t *testing.T) {
	if utils.TruncateMiddle("hello world", 10) != "hell…orld" {
		t.Errorf("got %s", utils.TruncateMiddle("hello world", 10))
	}
	if utils.TruncateMiddle("hello", 10) != "hello" {
		t.Errorf("got %s", utils.TruncateMiddle("hello", 10))
	}
}

func TestGetTerminalWidth(t *testing.T) {
	w := utils.GetTerminalWidth()
	if w <= 0 {
		t.Errorf("expected positive width")
	}
}

func TestCommandExists(t *testing.T) {
	if !utils.CommandExists("go") {
		t.Errorf("expected go to exist")
	}
	if utils.CommandExists("non-existent-command-12345") {
		t.Errorf("expected non-existent command not to exist")
	}
}
