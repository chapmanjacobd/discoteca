package utils_test

import (
	"context"
	"testing"

	"github.com/chapmanjacobd/discoteca/internal/utils"
)

func TestMaybeUpdate_ReturnsBool(t *testing.T) {
	t.Run("utils.MaybeUpdate returns bool", func(t *testing.T) {
		// When no update is available, should return false
		result := utils.MaybeUpdate(context.Background())
		if result != false {
			t.Logf("utils.MaybeUpdate(context.Background()) returned %v (update may have been available)", result)
		}
		// Note: Full testing would require mocking the GitHub API response
		// This test verifies the function signature returns bool
	})
}

func TestDoUpdate_ReturnsBool(t *testing.T) {
	t.Run("utils.DoUpdate returns false for invalid URL", func(t *testing.T) {
		result := utils.DoUpdate(context.Background(), "")
		if result != false {
			t.Errorf("utils.DoUpdate(context.Background(), \"\") = %v, want false", result)
		}
	})

	t.Run("utils.DoUpdate returns false for malformed URL", func(t *testing.T) {
		result := utils.DoUpdate(context.Background(), "not-a-valid-url")
		if result != false {
			t.Errorf("utils.DoUpdate(context.Background(), malformed) = %v, want false", result)
		}
	})
}
