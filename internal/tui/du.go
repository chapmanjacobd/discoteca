package tui

import (
	"fmt"
	"io"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/chapmanjacobd/discoteca/internal/models"
	"github.com/chapmanjacobd/discoteca/internal/query"
	"github.com/chapmanjacobd/discoteca/internal/utils"
)

var (
	sizeBarStyle = StyleMuted
	barFullStyle = lipgloss.NewStyle().Foreground(ColorAccent)
)

// duTreeNode represents a node in the disk usage tree
type duTreeNode struct {
	Path          string
	Name          string
	Count         int
	TotalSize     int64
	TotalDuration int64
	Children      map[string]*duTreeNode
	IsDir         bool
}

// buildDUTree builds a tree structure from media list for fast navigation
func buildDUTree(media []models.MediaWithDB) *duTreeNode {
	root := &duTreeNode{
		Path:     "",
		Name:     "root",
		Children: make(map[string]*duTreeNode),
		IsDir:    true,
	}

	for _, m := range media {
		path := m.Path
		parts := strings.FieldsFunc(path, func(r rune) bool {
			return r == '/' || r == '\\'
		})
		isAbs := len(path) > 0 && (path[0] == '/' || path[0] == '\\')

		// Update root stats
		if m.Size != nil {
			root.TotalSize += *m.Size
		}
		if m.Duration != nil {
			root.TotalDuration += *m.Duration
		}
		root.Count++

		// Build tree path
		current := root
		var currentPath string

		for i, part := range parts {
			// Build path up to this component
			if isAbs {
				if i == 0 {
					currentPath = "/" + part
				} else {
					currentPath = currentPath + "/" + part
				}
			} else {
				if i == 0 {
					currentPath = part
				} else {
					currentPath = currentPath + "/" + part
				}
			}

			if _, ok := current.Children[part]; !ok {
				current.Children[part] = &duTreeNode{
					Path:     currentPath,
					Name:     part,
					Children: make(map[string]*duTreeNode),
					IsDir:    true,
				}
			}
			current = current.Children[part]

			// Add file stats to this node
			if m.Size != nil {
				current.TotalSize += *m.Size
			}
			if m.Duration != nil {
				current.TotalDuration += *m.Duration
			}
			current.Count++
		}
	}

	return root
}

// getNodesAtDepth returns nodes at the specified depth from the tree
func getNodesAtDepth(node *duTreeNode, targetDepth, currentDepth int, pathPrefix string) []DUItem {
	var results []DUItem

	if currentDepth == targetDepth {
		// Return children of this node
		for _, child := range node.Children {
			results = append(results, DUItem{
				stats: models.FolderStats{
					Path:          child.Path,
					Count:         child.Count,
					TotalSize:     child.TotalSize,
					TotalDuration: child.TotalDuration,
				},
				isDir: true,
			})
		}
		return results
	}

	// Find the child matching the path prefix
	if pathPrefix != "" {
		parts := strings.FieldsFunc(pathPrefix, func(r rune) bool {
			return r == '/' || r == '\\'
		})
		if len(parts) > currentDepth {
			nextPart := parts[currentDepth]
			if child, ok := node.Children[nextPart]; ok {
				return getNodesAtDepth(child, targetDepth, currentDepth+1, pathPrefix)
			}
		}
		return results
	}

	// No path prefix - traverse all children
	for _, child := range node.Children {
		results = append(results, getNodesAtDepth(child, targetDepth, currentDepth+1, "")...)
	}

	return results
}

type DUItem struct {
	stats models.FolderStats
	isDir bool
}

func (i DUItem) Title() string {
	if i.isDir {
		return "📁 " + filepath.Base(i.stats.Path)
	}
	return "📄 " + filepath.Base(i.stats.Path)
}

func (i DUItem) Description() string {
	return fmt.Sprintf("%s • %d files • %s",
		utils.FormatSize(i.stats.TotalSize),
		i.stats.Count,
		utils.FormatDuration(int(i.stats.TotalDuration)))
}

func (i DUItem) FilterValue() string {
	return i.stats.Path
}

type DUModel struct {
	list        list.Model
	tree        *duTreeNode
	currentPath string
	history     []string
	quitting    bool
	flags       models.GlobalFlags
}

func NewDUModel(media []models.MediaWithDB, flags models.GlobalFlags) DUModel {
	m := DUModel{
		flags: flags,
		// Build tree once at startup for O(1) navigation
		tree: buildDUTree(media),
	}
	m.updateList()
	return m
}

func (m *DUModel) updateList() {
	// Determine target depth (children of current path)
	targetDepth := 1
	if m.currentPath != "" {
		cleanPath := filepath.Clean(m.currentPath)
		targetDepth = strings.Count(cleanPath, "/") + strings.Count(cleanPath, "\\") + 1
	}

	// Get nodes from tree at target depth (O(1) lookup instead of O(n) filtering)
	var items []list.Item
	var maxSize int64

	if m.tree != nil {
		nodes := getNodesAtDepth(m.tree, targetDepth, 0, m.currentPath)
		items = make([]list.Item, len(nodes))
		for i, node := range nodes {
			items[i] = node
			if node.stats.TotalSize > maxSize {
				maxSize = node.stats.TotalSize
			}
		}
	}

	// Sort using the standard sort function
	stats := make([]models.FolderStats, len(items))
	for i, item := range items {
		stats[i] = item.(DUItem).stats
	}
	query.SortFolders(stats, m.flags.SortBy, m.flags.Reverse)

	// Rebuild items with sorted stats
	for i, s := range stats {
		items[i] = DUItem{stats: s, isDir: true}
		if s.TotalSize > maxSize {
			maxSize = s.TotalSize
		}
	}

	l := list.New(items, duDelegate{maxSize: maxSize}, 0, 0)
	l.Title = "🪩  " + StyleLogoPrefix.Render(
		"Disco",
	) + StyleLogoSuffix.Render(
		"theque",
	) + " Disk Usage: " + m.currentPath
	if m.currentPath == "" {
		l.Title = "🪩  " + StyleLogoPrefix.Render("Disco") + StyleLogoSuffix.Render("theque") + " Disk Usage: Root"
	}
	l.SetShowStatusBar(true)
	l.SetFilteringEnabled(true)
	l.Styles.Title = StyleTitle
	m.list = l
}

type duDelegate struct {
	maxSize int64
}

func (d duDelegate) Height() int                             { return 2 }
func (d duDelegate) Spacing() int                            { return 1 }
func (d duDelegate) Update(_ tea.Msg, _ *list.Model) tea.Cmd { return nil }
func (d duDelegate) Render(w io.Writer, m list.Model, index int, listItem list.Item) {
	i, ok := listItem.(DUItem)
	if !ok {
		return
	}

	title := i.Title()
	desc := i.Description()

	if index == m.Index() {
		title = StyleSelected.Render(title)
		desc = StyleMuted.Render("  " + desc)
	} else {
		title = StyleNormal.Render(title)
		desc = StyleMuted.Render("  " + desc)
	}

	barWidth := 20
	filled := 0
	if d.maxSize > 0 {
		filled = int(float64(i.stats.TotalSize) / float64(d.maxSize) * float64(barWidth))
	}
	bar := "[" + barFullStyle.Render(
		strings.Repeat("#", filled),
	) + sizeBarStyle.Render(
		strings.Repeat("-", barWidth-filled),
	) + "]"

	fmt.Fprintf(w, "%s %s\n%s", title, bar, desc)
}

func (m DUModel) Init() tea.Cmd {
	return nil
}

func (m DUModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if m.list.FilterState() == list.Filtering {
			break
		}

		switch msg.String() {
		case "ctrl+c", "q":
			m.quitting = true
			return m, tea.Quit
		case "enter", "right":
			i, ok := m.list.SelectedItem().(DUItem)
			if ok && i.isDir {
				m.history = append(m.history, m.currentPath)
				m.currentPath = i.stats.Path
				m.updateList()
				return m, nil
			}
		case "backspace", "left":
			if len(m.history) > 0 {
				m.currentPath = m.history[len(m.history)-1]
				m.history = m.history[:len(m.history)-1]
				m.updateList()
				return m, nil
			}
		}
	case tea.WindowSizeMsg:
		h, v := StyleDoc.GetFrameSize()
		m.list.SetSize(msg.Width-h, msg.Height-v)
	}

	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func (m DUModel) View() string {
	if m.quitting {
		return ""
	}
	return StyleDoc.Render(m.list.View())
}
