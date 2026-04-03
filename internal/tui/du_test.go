package tui

import (
	"path/filepath"
	"testing"

	"github.com/chapmanjacobd/discoteca/internal/models"
	tea "github.com/charmbracelet/bubbletea"
)

func TestDUModel(t *testing.T) {
	v1Size := int64(1000)
	v2Size := int64(2000)
	media := []models.MediaWithDB{
		{Media: models.Media{Path: filepath.FromSlash("/home/user/vids/v1.mp4"), Size: &v1Size}},
		{Media: models.Media{Path: filepath.FromSlash("/home/user/vids/v2.mp4"), Size: &v2Size}},
		{Media: models.Media{Path: filepath.FromSlash("/home/user/music/m1.mp3"), Size: &v2Size}},
	}

	m := NewDUModel(media, models.GlobalFlags{})
	if m.list.Title == "" {
		t.Error("Expected list title to be set")
	}

	// Test navigation
	// Initial state: root, showing /home
	// Check that we have items at root level
	items := m.list.Items()
	if len(items) == 0 {
		t.Fatal("Expected items at root level, got none")
	}

	// Mock window size
	m2, _ := m.Update(tea.WindowSizeMsg{Width: 100, Height: 40})
	m = m2.(DUModel)

	// Verify we can navigate - select first item and press Enter
	if len(items) > 0 {
		// Select first item
		m2, _ = m.Update(tea.KeyMsg{Type: tea.KeyDown})
		m = m2.(DUModel)

		// Try to enter (if it's a directory)
		m2, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
		m = m2.(DUModel)
	}

	// Mock Backspace to go back
	m2, _ = m.Update(tea.KeyMsg{Type: tea.KeyBackspace})
	m = m2.(DUModel)

	// Mock Quit
	m2, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")})
	m = m2.(DUModel)
	if !m.quitting {
		t.Error("Expected quitting to be true after 'q'")
	}

	// View should be empty when quitting
	if m.View() != "" {
		t.Error("Expected empty view when quitting")
	}
}

func TestDUItem(t *testing.T) {
	stats := models.FolderStats{
		Path:          filepath.FromSlash("/test/path"),
		Count:         5,
		TotalSize:     5000,
		TotalDuration: 120,
	}
	item := duItem{stats: stats, isDir: true}
	if item.Title() != "📁 path" {
		t.Errorf("Unexpected title: %s", item.Title())
	}
	if item.FilterValue() != filepath.FromSlash("/test/path") {
		t.Errorf("Unexpected filter value: %s", item.FilterValue())
	}
	desc := item.Description()
	if desc == "" {
		t.Error("Description should not be empty")
	}
}
