package tui

import (
	"fmt"

	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/wheelibin/hugh/internal/lights"
)

const backgroundColor = "#011922"
const headerBackgroundColor = "#1e7ba0"

type lightUpdateMessage struct {
	lights []*lights.HughLight
}

var baseStyle = lipgloss.NewStyle().
	BorderStyle(lipgloss.NormalBorder()).
	BorderForeground(lipgloss.Color("240"))

type HughTUI struct {
	teaProgram *tea.Program
}

func NewHughTUI() HughTUI {
	m := NewModel()
	p := tea.NewProgram(m, tea.WithAltScreen())
	p.Run()

	return HughTUI{p}
}

func (t HughTUI) RefreshLights(lights *[]*lights.HughLight) {
	if lights != nil {
		t.teaProgram.Send(lightUpdateMessage{lights: *lights})
	}
}

type Model struct {
	table table.Model
	test  string
}

func NewModel() *Model {

	columns := []table.Column{
		{Title: "Light", Width: 10},
		{Title: "Reachable", Width: 4},
		{Title: "On", Width: 4},
		{Title: "Brightness", Width: 4},
		{Title: "Temperature", Width: 4},
	}

	rows := []table.Row{
		{"Office Lamp", "false", "true", "100", "4800"},
		{"Office Lamp", "false", "true", "100", "4800"},
		{"Office Lamp", "false", "true", "100", "4800"},
		{"Office Lamp", "false", "true", "100", "4800"},
		{"Office Lamp", "false", "true", "100", "4800"},
		{"Office Lamp", "false", "true", "100", "4800"},
		{"Office Lamp", "false", "true", "100", "4800"},
		{"Office Lamp", "false", "true", "100", "4800"},
	}

	t := table.New(
		table.WithColumns(columns),
		table.WithRows(rows),
		table.WithFocused(true),
		table.WithHeight(7),
	)

	s := table.DefaultStyles()
	s.Header = s.Header.
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("240")).
		BorderBottom(true).
		Bold(false)
	s.Selected = s.Selected.
		Foreground(lipgloss.Color("229")).
		Background(lipgloss.Color("57")).
		Bold(false)
	t.SetStyles(s)

	return &Model{t, ""}
}

func (m Model) Init() tea.Cmd {
	// return tea.EnterAltScreen
	return nil
}

func (m Model) Update(message tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := message.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "esc", "ctrl+c":
			return m, tea.Quit
		}

	case lightUpdateMessage:
		fmt.Println(">>>>>>>>>>>>>>>>>>>")
		rows := make([]table.Row, 0)
		for _, l := range msg.lights {
			rows = append(rows, []string{l.Name, "", fmt.Sprint(l.On), "", ""})
		}
		m.table.SetRows(rows)
		m.table.UpdateViewport()
		m.test = msg.lights[0].Name
		return m, nil

	default:
		return m, nil
	}

	return m, nil
}

func (m Model) View() string {
	return m.test

	// return baseStyle.Render(m.table.View()) + "\n"
}
