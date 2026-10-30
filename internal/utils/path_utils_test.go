package utils_test

import (
	"testing"

	"github.com/chapmanjacobd/discoteca/internal/utils"
)

func TestRandomString(t *testing.T) {
	s := utils.RandomString(10)
	if len(s) != 10 {
		t.Errorf("utils.RandomString(10) len = %d, want 10", len(s))
	}
}
