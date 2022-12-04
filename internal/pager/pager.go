package pager

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/reflow/wordwrap"
)

var (
	titleStyle lipgloss.Style
	infoStyle  lipgloss.Style
)

func init() {
	b := lipgloss.RoundedBorder()
	b.Right = "\u251C"
	titleStyle = lipgloss.NewStyle().BorderStyle(b).Padding(0, 1)

	b = lipgloss.RoundedBorder()
	b.Left = "\u2524"
	infoStyle = titleStyle.Copy().BorderStyle(b)
}

type Pager struct {
	model pagerModel
}

func New(name, content string) *Pager {
	return &Pager{
		model: pagerModel{
			name:    name,
			content: content,
		},
	}
}

func (p *Pager) Run() error {
	prog := tea.NewProgram(
		p.model,
		tea.WithMouseCellMotion(),
	)

	_, err := prog.Run()
	return err
}

type pagerModel struct {
	name     string
	content  string
	ready    bool
	viewport viewport.Model
}

func (pm pagerModel) Init() tea.Cmd {
	return tea.ClearScreen
}

func (pm pagerModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var (
		cmd  tea.Cmd
		cmds []tea.Cmd
	)

	switch msg := msg.(type) {
	case tea.KeyMsg:
		k := msg.String()
		if k == "ctrl+c" || k == "q" || k == "esc" {
			return pm, tea.Quit
		}
	case tea.WindowSizeMsg:
		headerHeight := lipgloss.Height(pm.headerView())
		footerHeight := lipgloss.Height(pm.footerView())
		verticalMarginHeight := headerHeight + footerHeight

		if !pm.ready {
			pm.viewport = viewport.New(msg.Width, msg.Height-verticalMarginHeight)
			pm.viewport.HighPerformanceRendering = true
			pm.viewport.YPosition = headerHeight + 1
			pm.viewport.SetContent(wordwrap.String(pm.content, msg.Width))
			pm.ready = true
		} else {
			pm.viewport.Width = msg.Width
			pm.viewport.Height = msg.Height - verticalMarginHeight
		}

		cmds = append(cmds, viewport.Sync(pm.viewport))
	}

	// Handle keyboard and mouse events in the viewport
	pm.viewport, cmd = pm.viewport.Update(msg)
	cmds = append(cmds, cmd)

	return pm, tea.Batch(cmds...)
}

func (pm pagerModel) View() string {
	if !pm.ready {
		return "\n  Initializing..."
	}
	return fmt.Sprintf("%s\n%s\n%s", pm.headerView(), pm.viewport.View(), pm.footerView())
}

func (pm pagerModel) headerView() string {
	title := titleStyle.Render(pm.name)
	line := strings.Repeat("─", max(0, pm.viewport.Width-lipgloss.Width(title)))
	return lipgloss.JoinHorizontal(lipgloss.Center, title, line)
}

func (pm pagerModel) footerView() string {
	info := infoStyle.Render(fmt.Sprintf("%3.f%%", pm.viewport.ScrollPercent()*100))
	line := strings.Repeat("─", max(0, pm.viewport.Width-lipgloss.Width(info)))
	return lipgloss.JoinHorizontal(lipgloss.Center, line, info)
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
