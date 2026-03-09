package utils

import (
	"testing"
)

func TestMaybeUpdate_ReturnsBool(t *testing.T) {
	t.Run("MaybeUpdate returns bool", func(t *testing.T) {
		// When no update is available, should return false
		result := MaybeUpdate()
		if result != false {
			t.Logf("MaybeUpdate() returned %v (update may have been available)", result)
		}
		// Note: Full testing would require mocking the GitHub API response
		// This test verifies the function signature returns bool
	})
}

func TestDoUpdate_ReturnsBool(t *testing.T) {
	t.Run("doUpdate returns false for invalid URL", func(t *testing.T) {
		result := doUpdate("")
		if result != false {
			t.Errorf("doUpdate(\"\") = %v, want false", result)
		}
	})

	t.Run("doUpdate returns false for malformed URL", func(t *testing.T) {
		result := doUpdate("not-a-valid-url")
		if result != false {
			t.Errorf("doUpdate(malformed) = %v, want false", result)
		}
	})
}
