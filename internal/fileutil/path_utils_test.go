package fileutil_test

import (
	"testing"

	"github.com/chapmanjacobd/discoteca/internal/fileutil"
)

func TestRandomString(t *testing.T) {
	s := fileutil.RandomString(10)
	if len(s) != 10 {
		t.Errorf("fileutil.RandomString(10) len = %d, want 10", len(s))
	}
}
