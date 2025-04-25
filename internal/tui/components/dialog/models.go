package dialog

import (
	"sort"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/opencode-ai/opencode/internal/llm/models"
	"github.com/opencode-ai/opencode/internal/tui/layout"
	"github.com/opencode-ai/opencode/internal/tui/styles"
	"github.com/opencode-ai/opencode/internal/tui/util"
)

// ModelSelectedMsg is sent when a model is selected
type ModelSelectedMsg struct {
	Model models.Model
}

// CloseModelDialogMsg is sent when a model is selected
type CloseModelDialogMsg struct{}

// ModelDialog interface for the model selection dialog
type ModelDialog interface {
	tea.Model
	layout.Bindings
	SetModels(models []models.Model, selectedModelId models.ModelID)
}

type modelDialogCmp struct {
	models      []models.Model
	selectedIdx int
	width       int
	height      int
}

type modelKeyMap struct {
	Up     key.Binding
	Down   key.Binding
	Enter  key.Binding
	Escape key.Binding
	J      key.Binding
	K      key.Binding
}

var modelKeys = modelKeyMap{
	Up: key.NewBinding(
		key.WithKeys("up"),
		key.WithHelp("↑", "previous model"),
	),
	Down: key.NewBinding(
		key.WithKeys("down"),
		key.WithHelp("↓", "next model"),
	),
	Enter: key.NewBinding(
		key.WithKeys("enter"),
		key.WithHelp("enter", "select model"),
	),
	Escape: key.NewBinding(
		key.WithKeys("esc"),
		key.WithHelp("esc", "close"),
	),
	J: key.NewBinding(
		key.WithKeys("j"),
		key.WithHelp("j", "next model"),
	),
	K: key.NewBinding(
		key.WithKeys("k"),
		key.WithHelp("k", "previous model"),
	),
}

func (m *modelDialogCmp) Init() tea.Cmd {
	return nil
}

func (m *modelDialogCmp) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, modelKeys.Up) || key.Matches(msg, modelKeys.K):
			if m.selectedIdx > 0 {
				m.selectedIdx--
			}
			return m, nil
		case key.Matches(msg, modelKeys.Down) || key.Matches(msg, modelKeys.J):
			if m.selectedIdx < len(m.models)-1 {
				m.selectedIdx++
			}
			return m, nil
		case key.Matches(msg, modelKeys.Enter):
			if len(m.models) > 0 {
				return m, util.CmdHandler(ModelSelectedMsg{
					Model: m.models[m.selectedIdx],
				})
			}
		case key.Matches(msg, modelKeys.Escape):
			return m, util.CmdHandler(CloseModelDialogMsg{})
		}
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
	}
	return m, nil
}

func (m *modelDialogCmp) View() string {
	if len(m.models) == 0 {
		return styles.BaseStyle.Padding(1, 2).
			Border(lipgloss.RoundedBorder()).
			BorderBackground(styles.Background).
			BorderForeground(styles.ForgroundDim).
			Width(40).
			Render("No models available for this provider")
	}

	maxWidth := 40

	numModels := len(m.models)
	modelItems := make([]string, 0, numModels)
	startIdx := 0

	for i := startIdx; i < numModels; i++ {
		model := m.models[i]
		itemStyle := styles.BaseStyle.Width(maxWidth)

		if i == m.selectedIdx {
			itemStyle = itemStyle.
				Background(styles.PrimaryColor).
				Foreground(styles.Background).
				Bold(true)
		}

		modelItems = append(modelItems, itemStyle.Render(model.Name))
	}

	title := styles.BaseStyle.
		Foreground(styles.PrimaryColor).
		Bold(true).
		Width(maxWidth).
		Padding(0, 1).
		Render("Select Model")

	content := lipgloss.JoinVertical(
		lipgloss.Left,
		title,
		styles.BaseStyle.Width(maxWidth).Render(""),
		styles.BaseStyle.Width(maxWidth).Render(lipgloss.JoinVertical(lipgloss.Left, modelItems...)),
		styles.BaseStyle.Width(maxWidth).Render(""),
	)

	return styles.BaseStyle.Padding(1, 2).
		Border(lipgloss.RoundedBorder()).
		BorderBackground(styles.Background).
		BorderForeground(styles.ForgroundDim).
		Width(lipgloss.Width(content) + 4).
		Render(content)
}

func (m *modelDialogCmp) BindingKeys() []key.Binding {
	return layout.KeyMapToSlice(modelKeys)
}

func (m *modelDialogCmp) SetModels(llmModels []models.Model, selectedModelId models.ModelID) {
	if llmModels == nil {
		m.models = []models.Model{}
		return
	}

	sortedModels := make([]models.Model, len(llmModels))
	copy(sortedModels, llmModels)

	// Sort models in reverse alphabetical order
	sort.Slice(sortedModels, func(i, j int) bool {
		return sortedModels[i].Name > sortedModels[j].Name
	})

	for i, model := range sortedModels {
		if model.ID == selectedModelId {
			m.selectedIdx = i
		}
	}

	m.models = sortedModels
}

func NewModelDialogCmp() ModelDialog {
	return &modelDialogCmp{
		models:      []models.Model{},
		selectedIdx: 0,
	}
}
