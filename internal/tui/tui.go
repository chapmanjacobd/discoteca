package tui

import (
	"context"
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/chapmanjacobd/discoteca/internal/models"
	"github.com/chapmanjacobd/discoteca/internal/query"
	"github.com/chapmanjacobd/discoteca/internal/utils"
)

type Pane int

const (
	PaneSidebar Pane = iota
	PaneMediaList
	PaneSearch
)

type (
	searchMsg []models.MediaWithDB
	errMsg    error
)

type itemDelegate struct{}

func (d itemDelegate) Height() int                               { return 2 }
func (d itemDelegate) Spacing() int                              { return 1 }
func (d itemDelegate) Update(_ tea.Msg, _ *list.Model) tea.Cmd { return nil }
func (d itemDelegate) Render(w io.Writer, m list.Model, index int, listItem list.Item) {
	i, ok := listItem.(item)
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

	fmt.Fprintf(w, "%s\n%s", title, desc)
}

type item struct {
	media models.MediaWithDB
}

func (i item) Title() string {
	icon := "❓"
	if i.media.MediaType != nil {
		switch *i.media.MediaType {
		case "audio":
			icon = "🎵"
		case "video":
			icon = "🎬"
		case "text":
			icon = "📄"
		case "image":
			icon = "🖼️"
		}
	}

	title := i.media.Path
	if i.media.Title != nil && *i.media.Title != "" {
		title = *i.media.Title
	}
	return icon + " " + title
}

func (i item) Description() string {
	dur := "-"
	if i.media.Duration != nil {
		dur = utils.FormatDuration(int(*i.media.Duration))
	}
	size := "-"
	if i.media.Size != nil {
		size = utils.FormatSize(*i.media.Size)
	}

	tags := ""
	if i.media.Categories != nil && *i.media.Categories != "" {
		tags = " • " + *i.media.Categories
	}

	return fmt.Sprintf("%s • %s • %s%s", dur, size, i.media.DB, tags)
}

func (i item) FilterValue() string {
	val := i.media.Path
	if i.media.Title != nil {
		val += " " + *i.media.Title
	}
	return val
}

type sidebarItem struct {
	title  string
	filter func(models.MediaWithDB) bool
}

func (i sidebarItem) Title() string       { return i.title }
func (i sidebarItem) Description() string { return "" }
func (i sidebarItem) FilterValue() string { return i.title }

type sidebarDelegate struct{}

func (d sidebarDelegate) Height() int                                  { return 1 }
func (d sidebarDelegate) Spacing() int                                 { return 0 }
func (d sidebarDelegate) Update(_ tea.Msg, _ *list.Model) tea.Cmd      { return nil }
func (d sidebarDelegate) Render(w io.Writer, m list.Model, index int, listItem list.Item) {
	i, ok := listItem.(sidebarItem)
	if !ok {
		return
	}

	str := i.title
	if index == m.Index() {
		str = StyleSelected.Render(str)
	} else {
		str = StyleNormal.Render(str)
	}

	fmt.Fprint(w, str)
}

type Model struct {
	sidebar     list.Model
	mediaList   list.Model
	searchInput textinput.Model
	allMedia    []models.MediaWithDB
	databases   []string
	flags       models.GlobalFlags
	choice      *models.MediaWithDB
	showDetails bool
	showHelp    bool
	quitting    bool
	activePane  Pane
	width       int
	height      int
}

func NewModel(media []models.MediaWithDB, databases []string, flags models.GlobalFlags, customCats []string) *Model {
	// Prepare media list
	items := make([]list.Item, len(media))
	for i, m := range media {
		items[i] = item{media: m}
	}

	ml := list.New(items, itemDelegate{}, 0, 0)
	ml.Title = "Media"
	ml.SetShowStatusBar(true)
	ml.SetFilteringEnabled(false) // We use our own live search
	ml.Styles.Title = StyleTitle

	ti := textinput.New()
	ti.Placeholder = "Search..."
	ti.Prompt = "🔍 "
	ti.CharLimit = 156
	ti.Width = 30

	// Prepare sidebar items
	sidebarItems := []list.Item{
		sidebarItem{title: "🏠 All Media", filter: func(_ models.MediaWithDB) bool { return true }},
		sidebarItem{
			title:  "🕒 History",
			filter: func(m models.MediaWithDB) bool { return m.TimeLastPlayed != nil && *m.TimeLastPlayed > 0 },
		},
		sidebarItem{
			title:  "🎵 Audio",
			filter: func(m models.MediaWithDB) bool { return m.MediaType != nil && *m.MediaType == "audio" },
		},
		sidebarItem{
			title:  "🎬 Video",
			filter: func(m models.MediaWithDB) bool { return m.MediaType != nil && *m.MediaType == "video" },
		},
		sidebarItem{
			title:  "🖼️ Image",
			filter: func(m models.MediaWithDB) bool { return m.MediaType != nil && *m.MediaType == "image" },
		},
		sidebarItem{
			title:  "📄 Text",
			filter: func(m models.MediaWithDB) bool { return m.MediaType != nil && *m.MediaType == "text" },
		},
		sidebarItem{
			title:  "⭐ 5 Stars",
			filter: func(m models.MediaWithDB) bool { return m.Score != nil && *m.Score >= 5 },
		},
		sidebarItem{
			title:  "⭐ 4+ Stars",
			filter: func(m models.MediaWithDB) bool { return m.Score != nil && *m.Score >= 4 },
		},
		sidebarItem{
			title:  "⭐ 3+ Stars",
			filter: func(m models.MediaWithDB) bool { return m.Score != nil && *m.Score >= 3 },
		},
		sidebarItem{
			title:  "🗑️ Trash",
			filter: func(m models.MediaWithDB) bool { return m.TimeDeleted != nil && *m.TimeDeleted > 0 },
		},
	}

	isCustom := make(map[string]bool)
	for _, c := range customCats {
		isCustom[c] = true
	}

	// Extract Categories
	categories := make(map[string]bool)
	for _, m := range media {
		if m.Categories != nil && *m.Categories != "" {
			cats := strings.SplitSeq(*m.Categories, ";")
			for c := range cats {
				if c != "" {
					categories[c] = true
				}
			}
		}
	}
	// Add custom categories that might have 0 count
	for _, c := range customCats {
		categories[c] = true
	}
	sortedCats := make([]string, 0, len(categories))
	for c := range categories {
		sortedCats = append(sortedCats, c)
	}
	sort.Strings(sortedCats)
	for _, c := range sortedCats {
		cat := c // capture
		sidebarItems = append(sidebarItems, sidebarItem{
			title: "🏷️ " + cat,
			filter: func(m models.MediaWithDB) bool {
				return m.Categories != nil && strings.Contains(*m.Categories, ";"+cat+";")
			},
		})
	}

	// Extract Genres
	genres := make(map[string]bool)
	for _, m := range media {
		if m.Genre != nil && *m.Genre != "" {
			genres[*m.Genre] = true
		}
	}
	sortedGenres := make([]string, 0, len(genres))
	for g := range genres {
		sortedGenres = append(sortedGenres, g)
	}
	sort.Strings(sortedGenres)
	for _, g := range sortedGenres {
		genre := g // capture
		sidebarItems = append(sidebarItems, sidebarItem{
			title: "🎭 " + genre,
			filter: func(m models.MediaWithDB) bool {
				return m.Genre != nil && *m.Genre == genre
			},
		})
	}

	sb := list.New(sidebarItems, sidebarDelegate{}, 0, 0)
	sb.Title = "Categories"
	sb.SetShowStatusBar(false)
	sb.SetFilteringEnabled(false)
	sb.SetShowPagination(false)
	sb.SetShowHelp(false)
	sb.Styles.Title = StyleTitle

	m := &Model{
		sidebar:     sb,
		mediaList:   ml,
		searchInput: ti,
		allMedia:    media,
		databases:   databases,
		flags:       flags,
		activePane:  PaneMediaList,
	}

	return m
}

func (m *Model) Init() tea.Cmd {
	return textinput.Blink
}

func (m *Model) performSearch(queryStr string) tea.Cmd {
	return func() tea.Msg {
		flags := m.flags
		if queryStr == "" {
			flags.Search = []string{}
		} else {
			flags.Search = []string{queryStr}
		}

		media, err := query.MediaQuery(context.Background(), m.databases, flags)
		if err != nil {
			return errMsg(err)
		}
		query.SortMedia(media, flags)
		return searchMsg(media)
	}
}

func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case searchMsg:
		m.allMedia = msg
		items := make([]list.Item, len(msg))
		for i, med := range msg {
			items[i] = item{media: med}
		}
		m.mediaList.SetItems(items)
		return m, nil

	case tea.KeyMsg:
		if m.showDetails || m.showHelp {
			if msg.String() == "d" || msg.String() == "esc" || msg.String() == "q" || msg.String() == "?" {
				m.showDetails = false
				m.showHelp = false
				return m, nil
			}
			return m, nil
		}

		if m.activePane == PaneSearch {
			switch msg.String() {
			case "enter", "esc":
				m.activePane = PaneMediaList
				m.searchInput.Blur()
				return m, nil
			}

			var tiCmd tea.Cmd
			m.searchInput, tiCmd = m.searchInput.Update(msg)
			return m, tea.Batch(tiCmd, m.performSearch(m.searchInput.Value()))
		}

		// Global keys
		switch msg.String() {
		case "ctrl+c":
			m.quitting = true
			return m, tea.Quit
		case "q":
			m.quitting = true
			return m, tea.Quit
		case "?":
			m.showHelp = true
			return m, nil
		case "/":
			m.activePane = PaneSearch
			return m, m.searchInput.Focus()
		case "tab":
			switch m.activePane {
			case PaneSidebar:
				m.activePane = PaneMediaList
			case PaneMediaList:
				m.activePane = PaneSearch
				return m, m.searchInput.Focus()
			default:
				m.activePane = PaneSidebar
				m.searchInput.Blur()
			}
			return m, nil
		case "left":
			if m.activePane == PaneMediaList && !m.mediaList.SettingFilter() {
				m.activePane = PaneSidebar
				return m, nil
			}
		case "right":
			if m.activePane == PaneSidebar {
				m.activePane = PaneMediaList
				return m, nil
			}
		case "d":
			if !m.mediaList.SettingFilter() {
				m.showDetails = true
				return m, nil
			}
		case "enter":
			switch m.activePane {
			case PaneMediaList:
				i, ok := m.mediaList.SelectedItem().(item)
				if ok {
					m.choice = &i.media
					return m, tea.Quit
				}
			case PaneSidebar:
				m.updateMediaList()
				m.activePane = PaneMediaList
				return m, nil
			}
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.recalculateSizes()
	}

	if m.activePane == PaneSidebar {
		m.sidebar, cmd = m.sidebar.Update(msg)
		m.updateMediaList()
	} else {
		m.mediaList, cmd = m.mediaList.Update(msg)
	}

	return m, cmd
}

func (m *Model) updateMediaList() {
	sbItem, ok := m.sidebar.SelectedItem().(sidebarItem)
	if !ok {
		return
	}

	var filteredItems []list.Item
	for _, med := range m.allMedia {
		if sbItem.filter(med) {
			filteredItems = append(filteredItems, item{media: med})
		}
	}
	m.mediaList.SetItems(filteredItems)
}

func (m *Model) recalculateSizes() {
	sidebarWidth := min(max(m.width/4, 20), 40)

	mediaListWidth := m.width - sidebarWidth - 4 // borders/padding

	_, v := StyleDoc.GetFrameSize()
	unusedHeight := v + 2 // title and footer

	m.sidebar.SetSize(sidebarWidth, m.height-unusedHeight)
	m.mediaList.SetSize(mediaListWidth, m.height-unusedHeight)
}

func (m *Model) View() string {
	if m.quitting {
		return ""
	}
	if m.showDetails {
		return StyleDoc.Render(m.renderDetails())
	}
	if m.showHelp {
		return StyleDoc.Render(m.renderHelp())
	}

	sidebarView := m.sidebar.View()
	mediaListView := m.mediaList.View()

	switch m.activePane {
	case PaneSidebar:
		sidebarView = StyleActivePane.Render(sidebarView)
		mediaListView = StyleInactivePane.Render(mediaListView)
	case PaneMediaList:
		sidebarView = StyleInactivePane.Render(sidebarView)
		mediaListView = StyleActivePane.Render(mediaListView)
	default:
		sidebarView = StyleInactivePane.Render(sidebarView)
		mediaListView = StyleInactivePane.Render(mediaListView)
	}

	header := "🪩  " + StyleLogoPrefix.Render("Disco") + StyleLogoSuffix.Render("theque")
	searchBar := "  " + m.searchInput.View()

	fullHeader := lipgloss.JoinHorizontal(lipgloss.Center, header, searchBar) + "\n"

	mainContent := lipgloss.JoinHorizontal(lipgloss.Top, sidebarView, mediaListView)

	footer := StyleMuted.Render("\nTab: Switch Pane • /: Search • Enter: Select • d: Details • ?: Help • q: Quit")

	return StyleDoc.Render(fullHeader + mainContent + footer)
}

func (m *Model) renderDetails() string {
	i, ok := m.mediaList.SelectedItem().(item)
	if !ok {
		return "No item selected"
	}
	media := i.media.Media

	var sb strings.Builder
	sb.WriteString(StyleHeader.Render("Media Details") + "\n\n")

	addField := func(label, value string) {
		if value != "" && value != "0" && value != "<nil>" {
			sb.WriteString(lipgloss.NewStyle().Foreground(ColorAccent).Bold(true).Render(label+": ") + value + "\n")
		}
	}

	addField("Path", media.Path)
	if media.Title != nil {
		addField("Title", *media.Title)
	}
	if media.MediaType != nil {
		addField("Type", *media.MediaType)
	}
	if media.Duration != nil {
		addField("Duration", utils.FormatDuration(int(*media.Duration)))
	}
	if media.Size != nil {
		addField("Size", utils.FormatSize(*media.Size))
	}
	if media.VideoCodecs != nil {
		addField("Video Codec", *media.VideoCodecs)
	}
	if media.AudioCodecs != nil {
		addField("Audio Codec", *media.AudioCodecs)
	}
	if media.Width != nil && media.Height != nil {
		addField("Resolution", fmt.Sprintf("%dx%d", *media.Width, *media.Height))
	}
	if media.Album != nil {
		addField("Album", *media.Album)
	}
	if media.Artist != nil {
		addField("Artist", *media.Artist)
	}
	if media.Genre != nil {
		addField("Genre", *media.Genre)
	}
	if media.Categories != nil {
		addField("Categories", *media.Categories)
	}
	if media.Score != nil {
		addField("Score", fmt.Sprintf("%.1f", *media.Score))
	}

	sb.WriteString("\n" + StyleMuted.Render("Press 'd', 'esc', or 'q' to return"))

	return sb.String()
}

func (m *Model) renderHelp() string {
	var sb strings.Builder
	sb.WriteString(StyleHeader.Render("Keyboard Shortcuts") + "\n\n")

	addShortcut := func(key, desc string) {
		sb.WriteString(lipgloss.NewStyle().Foreground(ColorAccent).Width(12).Render(key))
		sb.WriteString(StyleNormal.Render(desc) + "\n")
	}

	sb.WriteString(lipgloss.NewStyle().Foreground(ColorLogo1).Bold(true).Render("Navigation") + "\n")
	addShortcut("Tab", "Switch between sidebar, list, and search")
	addShortcut("↑/↓", "Move selection")
	addShortcut("←/→", "Quick switch between sidebar and list")
	addShortcut("Enter", "Play selected media / Apply sidebar filter")

	sb.WriteString("\n" + lipgloss.NewStyle().Foreground(ColorLogo1).Bold(true).Render("Search & Filter") + "\n")
	addShortcut("/", "Focus search bar")
	addShortcut("Esc", "Exit search / Clear focus")

	sb.WriteString("\n" + lipgloss.NewStyle().Foreground(ColorLogo1).Bold(true).Render("General") + "\n")
	addShortcut("d", "Show media details")
	addShortcut("?", "Toggle this help menu")
	addShortcut("q", "Quit")
	addShortcut("Ctrl+c", "Force quit")

	sb.WriteString("\n" + StyleMuted.Render("Press '?', 'esc', or 'q' to return"))

	return sb.String()
}

func (m *Model) GetChoice() *models.MediaWithDB {
	return m.choice
}
