package tui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/gianlucarea/parley/internal/collection"
)

// Editor displays the selected request's fields in a read-only summary view.
// In v1 it shows method, URL, headers, body, and auth — editable fields are
// out of scope for this phase.
type Editor struct {
	req    collection.Request
	width  int
	height int
}

func NewEditor(width, height int) Editor {
	return Editor{width: width, height: height}
}

func (e *Editor) SetRequest(req collection.Request) {
	e.req = req
}

func (e *Editor) SetSize(w, h int) {
	e.width = w
	e.height = h
}

func (e Editor) Update(msg tea.Msg) (Editor, tea.Cmd) {
	return e, nil
}

var (
	labelStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#888888")).
			Width(10)
	valueStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#EEEEEE"))
	panelStyle = lipgloss.NewStyle().Padding(1, 2)
)

func (e Editor) View() string {
	if e.req.Name == "" {
		return panelStyle.Render(
			lipgloss.NewStyle().Foreground(lipgloss.Color("#555555")).
				Render("No request selected. Use the sidebar to pick one."),
		)
	}

	var b strings.Builder
	row := func(label, value string) {
		b.WriteString(labelStyle.Render(label))
		b.WriteString("  ")
		b.WriteString(valueStyle.Render(value))
		b.WriteString("\n")
	}

	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#7D56F4"))
	b.WriteString(titleStyle.Render(e.req.Name))
	b.WriteString("\n\n")

	row("Method", methodStyle(e.req.Method))
	row("URL", e.req.URL)

	if len(e.req.Headers) > 0 {
		b.WriteString("\n" + labelStyle.Render("Headers") + "\n")
		for k, v := range e.req.Headers {
			b.WriteString("  " + valueStyle.Render(k+": "+v) + "\n")
		}
	}

	if e.req.Body != "" {
		b.WriteString("\n" + labelStyle.Render("Body") + "\n")
		b.WriteString("  " + valueStyle.Render(e.req.Body) + "\n")
	}

	if e.req.Auth.Type != "" {
		b.WriteString("\n" + labelStyle.Render("Auth") + "\n")
		switch strings.ToLower(e.req.Auth.Type) {
		case "bearer":
			b.WriteString("  " + valueStyle.Render("Bearer "+e.req.Auth.Token) + "\n")
		case "basic":
			b.WriteString("  " + valueStyle.Render("Basic "+e.req.Auth.Username+":***") + "\n")
		}
	}

	return panelStyle.Width(e.width).Render(b.String())
}
