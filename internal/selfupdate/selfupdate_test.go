package selfupdate_test

import (
	"context"
	"testing"

	"github.com/chapmanjacobd/discoteca/internal/selfupdate"
)

func TestMaybeUpdate_ReturnsBool(t *testing.T) {
	t.Run("selfupdate.MaybeUpdate returns bool", func(t *testing.T) {
		// When no update is available, should return false
		result := selfupdate.MaybeUpdate(context.Background())
		if result {
			t.Logf("selfupdate.MaybeUpdate(context.Background()) returned %v (update may have been available)", result)
		}
		// Note: Full testing would require mocking the GitHub API response
		// This test verifies the function signature returns bool
	})
}

func TestDoUpdate_ReturnsBool(t *testing.T) {
	t.Run("selfupdate.DoUpdate returns false for invalid URL", func(t *testing.T) {
		result := selfupdate.DoUpdate(context.Background(), "")
		if result {
			t.Errorf("selfupdate.DoUpdate(context.Background(), \"\") = %v, want false", result)
		}
	})

	t.Run("selfupdate.DoUpdate returns false for malformed URL", func(t *testing.T) {
		result := selfupdate.DoUpdate(context.Background(), "not-a-valid-url")
		if result {
			t.Errorf("selfupdate.DoUpdate(context.Background(), malformed) = %v, want false", result)
		}
	})
}
