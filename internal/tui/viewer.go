package tui

import (
	"fmt"
	"strings"

	"github.com/alecthomas/chroma/v2/quick"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	httpexec "github.com/gianlucarea/parley/internal/http"
)

// Viewer renders the HTTP response: status bar + syntax-highlighted body.
type Viewer struct {
	viewport viewport.Model
	resp     *httpexec.Response
	err      error
	loading  bool
}

func NewViewer(width, height int) Viewer {
	vp := viewport.New(width, height-3) // reserve 3 lines for the status bar
	return Viewer{viewport: vp}
}

func (v *Viewer) SetSize(w, h int) {
	v.viewport.Width = w
	v.viewport.Height = h - 3
}

func (v *Viewer) SetLoading(loading bool) {
	v.loading = loading
	if loading {
		v.viewport.SetContent(lipgloss.NewStyle().Foreground(lipgloss.Color("#888888")).Render("Sending request…"))
	}
}

func (v *Viewer) SetResponse(resp httpexec.Response) {
	v.resp = &resp
	v.err = nil
	v.loading = false
	v.viewport.SetContent(highlightBody(resp.Body))
	v.viewport.GotoTop()
}

func (v *Viewer) SetError(err error) {
	v.err = err
	v.loading = false
	v.viewport.SetContent(
		lipgloss.NewStyle().Foreground(lipgloss.Color("#F45B5B")).Render("Error: " + err.Error()),
	)
}

func (v Viewer) Update(msg tea.Msg) (Viewer, tea.Cmd) {
	var cmd tea.Cmd
	v.viewport, cmd = v.viewport.Update(msg)
	return v, cmd
}

var (
	statusOKStyle   = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#04B575"))
	statusErrStyle  = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#F45B5B"))
	statusInfoStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#888888"))
	dividerStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("#333333"))
)

func (v Viewer) View() string {
	var statusBar string
	if v.resp != nil {
		codeSty := statusOKStyle
		if v.resp.StatusCode >= 400 {
			codeSty = statusErrStyle
		}
		elapsed := fmt.Sprintf("%.0fms", float64(v.resp.Elapsed.Milliseconds()))
		statusBar = lipgloss.JoinHorizontal(lipgloss.Left,
			codeSty.Render(v.resp.Status),
			statusInfoStyle.Render("  •  "+elapsed),
		)
	} else if v.err != nil {
		statusBar = statusErrStyle.Render("Request Failed")
	} else if v.loading {
		statusBar = statusInfoStyle.Render("Loading…")
	} else {
		statusBar = statusInfoStyle.Render("Press Enter to run the selected request")
	}

	divider := dividerStyle.Render(strings.Repeat("─", v.viewport.Width))
	return lipgloss.JoinVertical(lipgloss.Left,
		statusBar,
		divider,
		v.viewport.View(),
	)
}

// highlightBody applies syntax highlighting to JSON/HTML/XML bodies.
// Falls back to plain text if detection fails.
func highlightBody(body string) string {
	if body == "" {
		return lipgloss.NewStyle().Foreground(lipgloss.Color("#555555")).Render("(empty body)")
	}
	var buf strings.Builder
	err := quick.Highlight(&buf, body, "json", "terminal16m", "monokai")
	if err != nil {
		// fallback: plain
		return body
	}
	return buf.String()
}
