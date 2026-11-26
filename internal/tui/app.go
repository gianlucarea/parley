package tui

import (
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/gianlucarea/parley/internal/collection"
	"github.com/gianlucarea/parley/internal/env"
	httpexec "github.com/gianlucarea/parley/internal/http"
)

// pane identifies the active focus area.
type pane int

const (
	paneSidebar pane = iota
	paneEditor
	paneViewer
)

// responseMsg is sent back to the app when an HTTP request completes.
type responseMsg struct {
	resp httpexec.Response
	err  error
}

// App is the root Bubbletea model.
type App struct {
	sidebar  Sidebar
	editor   Editor
	viewer   Viewer
	focus    pane
	vars     map[string]string
	envs     env.Set
	envOrder []string // "" (base only) first, then named envs
	envIdx   int
	width    int
	height   int
	showHelp bool
	status   string // transient message shown in the bottom bar
}

// New creates and returns an initialised App using the given environment set,
// starting on envName ("" means base .parley.env only).
func New(collections []collection.Collection, envs env.Set, envName string) App {
	// sensible defaults before window size arrives
	sw, ew, vw, h := 28, 40, 60, 24
	order := append([]string{""}, envs.Names...)
	idx := 0
	for i, n := range order {
		if n == envName {
			idx = i
		}
	}
	a := App{
		sidebar:  NewSidebar(collections, sw, h),
		editor:   NewEditor(ew, h),
		viewer:   NewViewer(vw, h),
		focus:    paneSidebar,
		envs:     envs,
		envOrder: order,
		envIdx:   idx,
	}
	a.vars, _ = envs.Resolve(order[idx])
	return a
}

// currentEnv returns the display name of the active environment.
func (a App) currentEnv() string {
	if n := a.envOrder[a.envIdx]; n != "" {
		return n
	}
	return "default"
}

func (a App) Init() tea.Cmd { return nil }

// --- update -----------------------------------------------------------------

func (a App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case tea.WindowSizeMsg:
		a.width = msg.Width
		a.height = msg.Height
		sw, ew, vw := splitWidths(msg.Width)
		a.sidebar.SetSize(sw, msg.Height-1)
		a.editor.SetSize(ew, msg.Height-1)
		a.viewer.SetSize(vw, msg.Height-1)

	case tea.KeyMsg:
		if a.focus == paneSidebar && a.sidebar.Filtering() && msg.String() != "ctrl+c" {
			break // let the filter input receive every key
		}
		a.status = ""
		switch msg.String() {
		case "e":
			if len(a.envOrder) > 1 {
				a.envIdx = (a.envIdx + 1) % len(a.envOrder)
				a.vars, _ = a.envs.Resolve(a.envOrder[a.envIdx])
			} else {
				a.status = "no named envs: add .parley.<name>.env in the launch directory"
			}
			return a, nil
		case "ctrl+c", "q":
			return a, tea.Quit
		case "?":
			a.showHelp = !a.showHelp
			return a, nil
		case "tab":
			a.focus = (a.focus + 1) % 3
			return a, nil
		case "enter":
			if a.focus == paneSidebar {
				a.focus = paneEditor
				if req, ok := a.sidebar.SelectedRequest(); ok {
					a.editor.SetRequest(req)
				}
			} else if a.focus == paneEditor {
				if req, ok := a.sidebar.SelectedRequest(); ok {
					a.viewer.SetLoading(true)
					vars := a.vars
					return a, func() tea.Msg {
						resp, err := httpexec.Execute(req, vars)
						return responseMsg{resp: resp, err: err}
					}
				}
			}
		}

	case responseMsg:
		if msg.err != nil {
			a.viewer.SetError(msg.err)
		} else {
			a.viewer.SetResponse(msg.resp)
		}
		a.focus = paneViewer
		return a, nil

	case list.FilterMatchesMsg:
		// fall through to sidebar update
	}

	// Route remaining msgs to the focused pane.
	var cmd tea.Cmd
	switch a.focus {
	case paneSidebar:
		a.sidebar, cmd = a.sidebar.Update(msg)
	case paneEditor:
		a.editor, cmd = a.editor.Update(msg)
	case paneViewer:
		a.viewer, cmd = a.viewer.Update(msg)
	}
	return a, cmd
}

// --- view -------------------------------------------------------------------

var (
	activePane   = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color("#7D56F4"))
	inactivePane = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color("#333333"))
	helpStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("#555555"))
	envStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("#7D56F4")).Bold(true)
)

func (a App) View() string {
	if a.showHelp {
		return helpOverlay()
	}

	sw, ew, vw := splitWidths(a.width)
	h := a.height - 3 // borders

	sidebarView := paneBox(a.sidebar.View(), sw, h, a.focus == paneSidebar)
	editorView := paneBox(a.editor.View(), ew, h, a.focus == paneEditor)
	viewerView := paneBox(a.viewer.View(), vw, h, a.focus == paneViewer)

	main := lipgloss.JoinHorizontal(lipgloss.Top, sidebarView, editorView, viewerView)
	hintText := "Tab: switch pane  Enter: select/run  e: switch env  ?: help  q: quit"
	if a.status != "" {
		hintText = a.status
	}
	hint := envStyle.Render("env: "+a.currentEnv()) + "  " + helpStyle.Render(hintText)
	return lipgloss.JoinVertical(lipgloss.Left, main, hint)
}

func paneBox(content string, w, h int, active bool) string {
	sty := inactivePane
	if active {
		sty = activePane
	}
	return sty.Width(w).Height(h).Render(content)
}

func helpOverlay() string {
	help := `
  parley — keyboard shortcuts

  Tab          Switch focus between panes
  Enter        Select request / Run request
  ↑ / ↓        Navigate sidebar
  / (slash)    Filter requests in sidebar
  Esc          Clear filter
  e            Cycle environment (.parley.<name>.env)
  ?            Toggle this help
  q / Ctrl+C   Quit
`
	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#7D56F4")).
		Padding(1, 3).
		Render(help)
}

func splitWidths(total int) (sidebar, editor, viewer int) {
	if total == 0 {
		return 28, 40, 60
	}
	sidebar = total / 5
	remaining := total - sidebar
	editor = remaining / 2
	viewer = remaining - editor
	return
}

// Run starts the Bubbletea program for the given collections and environments.
func Run(collections []collection.Collection, envs env.Set, envName string) error {
	p := tea.NewProgram(New(collections, envs, envName), tea.WithAltScreen())
	_, err := p.Run()
	return err
}
