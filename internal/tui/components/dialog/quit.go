package dialog

import (
	"strings"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/opencode-ai/opencode/internal/tui/layout"
	"github.com/opencode-ai/opencode/internal/tui/styles"
	"github.com/opencode-ai/opencode/internal/tui/util"
	"github.com/opencode-ai/opencode/internal/tui/config"
)

const question = "Are you sure you want to quit?"

type CloseQuitMsg struct{}

type QuitDialog interface {
	tea.Model
	layout.Bindings
}

type quitDialogCmp struct {
	selectedNo bool
}

type helpMapping struct {
	LeftRight  key.Binding
	EnterSpace key.Binding
	Yes        key.Binding
	No         key.Binding
	Tab        key.Binding
}

func NewHelpMapping(hotkeys config.HotkeyConfig) helpMapping {
	return helpMapping{
		LeftRight: config.GetKeyBinding(
			hotkeys.Left+","+hotkeys.Right,
			"←/→",
			"switch options",
		),
		EnterSpace: config.GetKeyBinding(
			hotkeys.Enter+",space",
			"enter/space",
			"confirm",
		),
		Yes: config.GetKeyBinding(
			"y,Y",
			"y/Y",
			"yes",
		),
		No: config.GetKeyBinding(
			"n,N",
			"n/N",
			"no",
		),
		Tab: config.GetKeyBinding(
			hotkeys.Tab,
			hotkeys.Tab,
			"switch options",
		),
	}
}

func (q *quitDialogCmp) Init() tea.Cmd {
	return nil
}

func (q *quitDialogCmp) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, helpKeys.LeftRight) || key.Matches(msg, helpKeys.Tab):
			q.selectedNo = !q.selectedNo
			return q, nil
		case key.Matches(msg, helpKeys.EnterSpace):
			if !q.selectedNo {
				return q, tea.Quit
			}
			return q, util.CmdHandler(CloseQuitMsg{})
		case key.Matches(msg, helpKeys.Yes):
			return q, tea.Quit
		case key.Matches(msg, helpKeys.No):
			return q, util.CmdHandler(CloseQuitMsg{})
		}
	}
	return q, nil
}

func (q *quitDialogCmp) View() string {
	yesStyle := styles.BaseStyle
	noStyle := styles.BaseStyle
	spacerStyle := styles.BaseStyle.Background(styles.Background)

	if q.selectedNo {
		noStyle = noStyle.Background(styles.PrimaryColor).Foreground(styles.Background)
		yesStyle = yesStyle.Background(styles.Background).Foreground(styles.PrimaryColor)
	} else {
		yesStyle = yesStyle.Background(styles.PrimaryColor).Foreground(styles.Background)
		noStyle = noStyle.Background(styles.Background).Foreground(styles.PrimaryColor)
	}

	yesButton := yesStyle.Padding(0, 1).Render("Yes")
	noButton := noStyle.Padding(0, 1).Render("No")

	buttons := lipgloss.JoinHorizontal(lipgloss.Left, yesButton, spacerStyle.Render("  "), noButton)

	width := lipgloss.Width(question)
	remainingWidth := width - lipgloss.Width(buttons)
	if remainingWidth > 0 {
		buttons = spacerStyle.Render(strings.Repeat(" ", remainingWidth)) + buttons
	}

	content := styles.BaseStyle.Render(
		lipgloss.JoinVertical(
			lipgloss.Center,
			question,
			"",
			buttons,
		),
	)

	return styles.BaseStyle.Padding(1, 2).
		Border(lipgloss.RoundedBorder()).
		BorderBackground(styles.Background).
		BorderForeground(styles.ForgroundDim).
		Width(lipgloss.Width(content) + 4).
		Render(content)
}

func (q *quitDialogCmp) BindingKeys() []key.Binding {
	return layout.KeyMapToSlice(helpKeys)
}

func NewQuitCmp() QuitDialog {
	return &quitDialogCmp{
		selectedNo: true,
	}
}
