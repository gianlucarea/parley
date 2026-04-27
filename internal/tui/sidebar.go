package tui

import (
	"fmt"
	"io"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/gianlucarea/parley/internal/collection"
)

// collectionItem wraps a collection.Request for the bubbles list.
type collectionItem struct {
	req     collection.Request
	colName string
}

func (i collectionItem) FilterValue() string { return i.req.Name }

// sectionItem is a non-selectable collection header row.
type sectionItem struct{ name string }

func (s sectionItem) FilterValue() string { return "" }

// itemDelegate renders both section headers and request entries.
type itemDelegate struct{}

func (d itemDelegate) Height() int                             { return 1 }
func (d itemDelegate) Spacing() int                            { return 0 }
func (d itemDelegate) Update(_ tea.Msg, _ *list.Model) tea.Cmd { return nil }
func (d itemDelegate) Render(w io.Writer, m list.Model, index int, listItem list.Item) {
	switch i := listItem.(type) {
	case sectionItem:
		fmt.Fprint(w, sectionStyle.Render("  "+i.name))
	case collectionItem:
		str := fmt.Sprintf("%s %s", methodStyle(i.req.Method), i.req.Name)
		if index == m.Index() {
			fmt.Fprint(w, selectedItemStyle.Render(str))
		} else {
			fmt.Fprint(w, itemStyle.Render(str))
		}
	}
}

var (
	sectionStyle      = lipgloss.NewStyle().Foreground(lipgloss.Color("#AAAAAA")).Bold(true)
	itemStyle         = lipgloss.NewStyle().PaddingLeft(4)
	selectedItemStyle = lipgloss.NewStyle().PaddingLeft(3).
				Foreground(lipgloss.Color("#7D56F4")).
				Bold(true).
				SetString("> ")
	filterActiveStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#7D56F4"))
	filterDimStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("#444444"))
)

func methodStyle(m string) string {
	colors := map[string]string{
		"GET":    "#04B575",
		"POST":   "#F4A535",
		"PUT":    "#4DA6FF",
		"PATCH":  "#A855F7",
		"DELETE": "#F45B5B",
	}
	c, ok := colors[m]
	if !ok {
		c = "#888888"
	}
	return lipgloss.NewStyle().Foreground(lipgloss.Color(c)).Bold(true).Render(m)
}

// Sidebar holds the collections list pane model.
type Sidebar struct {
	list        list.Model
	collections []collection.Collection
	filter      textinput.Model
	filtering   bool
	listW       int
	listH       int
}

func NewSidebar(collections []collection.Collection, width, height int) Sidebar {
	ti := textinput.New()
	ti.Placeholder = "filter…"
	ti.Prompt = "/ "
	ti.PromptStyle = filterActiveStyle
	ti.TextStyle = filterActiveStyle

	s := Sidebar{
		collections: collections,
		filter:      ti,
		listW:       width,
		listH:       height - 1, // always reserve 1 row for filter bar
	}
	s.list = s.buildList()
	s.skipSectionItems(1)
	return s
}

// buildList constructs a fresh list.Model from the current filter query.
func (s *Sidebar) buildList() list.Model {
	items := s.filteredItems()
	l := list.New(items, itemDelegate{}, s.listW, s.listH)
	l.Title = "Collections"
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(false) // we manage filtering ourselves
	l.KeyMap.Quit.SetEnabled(false)
	l.Styles.Title = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#7D56F4")).
		Bold(true).
		Padding(0, 1)
	return l
}

// filteredItems returns list items matching the current filter query.
// Section headers are included whenever at least one child matches.
func (s *Sidebar) filteredItems() []list.Item {
	query := strings.ToLower(strings.TrimSpace(s.filter.Value()))
	var items []list.Item
	for _, col := range s.collections {
		if len(col.Requests) == 0 {
			continue
		}
		var matching []list.Item
		for _, req := range col.Requests {
			nameMatch := strings.Contains(strings.ToLower(req.Name), query)
			colMatch := strings.Contains(strings.ToLower(col.Name), query)
			if query == "" || nameMatch || colMatch {
				matching = append(matching, collectionItem{req: req, colName: col.Name})
			}
		}
		if len(matching) > 0 {
			items = append(items, sectionItem{name: col.Name})
			items = append(items, matching...)
		}
	}
	return items
}

func (s Sidebar) Update(msg tea.Msg) (Sidebar, tea.Cmd) {
	if s.filtering {
		if key, ok := msg.(tea.KeyMsg); ok {
			switch key.String() {
			case "esc":
				s.filter.Reset()
				s.filtering = false
				s.filter.Blur()
				s.list = s.buildList()
				s.skipSectionItems(1)
				return s, nil
			case "enter":
				s.filtering = false
				s.filter.Blur()
				return s, nil
			case "up", "k":
				s.moveCursor(-1)
				return s, nil
			case "down", "j":
				s.moveCursor(1)
				return s, nil
			}
		}
		var cmd tea.Cmd
		s.filter, cmd = s.filter.Update(msg)
		// Rebuild list and preserve cursor position when query changes.
		prevIdx := s.list.Index()
		s.list = s.buildList()
		if prevIdx > 0 && prevIdx < len(s.list.Items()) {
			s.list.Select(prevIdx)
		}
		s.skipSectionItems(1)
		return s, cmd
	}

	// Normal navigation mode.
	if key, ok := msg.(tea.KeyMsg); ok {
		switch key.String() {
		case "/":
			s.filtering = true
			return s, s.filter.Focus()
		case "up", "k":
			s.moveCursor(-1)
			return s, nil
		case "down", "j":
			s.moveCursor(1)
			return s, nil
		}
	}

	var cmd tea.Cmd
	s.list, cmd = s.list.Update(msg)
	return s, cmd
}

func (s Sidebar) View() string {
	var filterBar string
	switch {
	case s.filtering:
		filterBar = s.filter.View()
	case s.filter.Value() != "":
		filterBar = filterActiveStyle.Render("/ " + s.filter.Value())
	default:
		filterBar = filterDimStyle.Render("/ to filter")
	}
	return lipgloss.JoinVertical(lipgloss.Left, s.list.View(), filterBar)
}

func (s Sidebar) SelectedRequest() (collection.Request, bool) {
	item, ok := s.list.SelectedItem().(collectionItem)
	if !ok {
		return collection.Request{}, false
	}
	return item.req, true
}

func (s *Sidebar) SetSize(w, h int) {
	s.listW = w
	s.listH = h - 1
	s.list.SetSize(w, s.listH)
}

// moveCursor advances the selection by dir (+1 or -1), skipping sectionItems.
func (s *Sidebar) moveCursor(dir int) {
	items := s.list.Items()
	next := s.list.Index() + dir
	for next >= 0 && next < len(items) {
		if _, isSection := items[next].(sectionItem); !isSection {
			s.list.Select(next)
			return
		}
		next += dir
	}
}

// skipSectionItems positions the cursor on the first non-header item.
func (s *Sidebar) skipSectionItems(dir int) {
	items := s.list.Items()
	if len(items) == 0 {
		return
	}
	idx := s.list.Index()
	for idx >= 0 && idx < len(items) {
		if _, isSection := items[idx].(sectionItem); !isSection {
			break
		}
		idx += dir
	}
	if idx >= 0 && idx < len(items) {
		s.list.Select(idx)
	}
}
