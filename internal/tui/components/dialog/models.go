package dialog

import (
	"fmt"
	"sort"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/opencode-ai/opencode/internal/llm/models"
	"github.com/opencode-ai/opencode/internal/tui/layout"
	"github.com/opencode-ai/opencode/internal/tui/styles"
	"github.com/opencode-ai/opencode/internal/tui/util"
)

const (
	numVisibleModels = 5
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
	models       []models.Model
	selectedIdx  int
	width        int
	height       int
	scrollOffset int
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
			// Move selection up or wrap to bottom
			if m.selectedIdx > 0 {
				m.selectedIdx--
			} else {
				m.selectedIdx = len(m.models) - 1
				m.scrollOffset = max(0, len(m.models)-numVisibleModels)
			}
			// Keep selection visible
			if m.selectedIdx < m.scrollOffset {
				m.scrollOffset = m.selectedIdx
			}
		case key.Matches(msg, modelKeys.Down) || key.Matches(msg, modelKeys.J):
			// Move selection down or wrap to top
			if m.selectedIdx < len(m.models)-1 {
				m.selectedIdx++
			} else {
				m.selectedIdx = 0
				m.scrollOffset = 0
			}
			// Keep selection visible
			if m.selectedIdx >= m.scrollOffset+numVisibleModels {
				m.scrollOffset = m.selectedIdx - (numVisibleModels - 1)
			}
		case key.Matches(msg, modelKeys.Enter):
			util.ReportInfo(fmt.Sprintf("selected model: %s", m.models[m.selectedIdx].Name))
			return m, util.CmdHandler(ModelSelectedMsg{Model: m.models[m.selectedIdx]})
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
	maxWidth := 40

	if len(m.models) == 0 {
		return styles.BaseStyle.Padding(1, 2).
			Border(lipgloss.RoundedBorder()).
			BorderBackground(styles.Background).
			BorderForeground(styles.ForgroundDim).
			Width(maxWidth).
			Render("No models available")
	}

	// Calculate visible range
	endIdx := min(m.scrollOffset+numVisibleModels, len(m.models))
	modelItems := make([]string, 0, endIdx-m.scrollOffset)

	// Render visible models
	for i := m.scrollOffset; i < endIdx; i++ {
		itemStyle := styles.BaseStyle.Width(maxWidth)
		if i == m.selectedIdx {
			itemStyle = itemStyle.Background(styles.PrimaryColor).
				Foreground(styles.Background).Bold(true)
		}
		modelItems = append(modelItems, itemStyle.Render(m.models[i].Name))
	}

	// Add scroll indicators if needed
	var scrollIndicator string
	if len(m.models) > numVisibleModels {
		if m.scrollOffset > 0 {
			scrollIndicator += "↑ "
		}
		if endIdx < len(m.models) {
			scrollIndicator += "↓"
		}
		if scrollIndicator != "" {
			scrollIndicator = styles.BaseStyle.
				Foreground(styles.PrimaryColor).
				Width(maxWidth).
				Align(lipgloss.Right).
				Bold(true).
				Render(scrollIndicator)
		}
	}

	title := styles.BaseStyle.
		Foreground(styles.PrimaryColor).
		Bold(true).
		Width(maxWidth).
		Padding(0, 0, 1).
		Render("Select Model")

	content := lipgloss.JoinVertical(
		lipgloss.Left,
		title,
		styles.BaseStyle.Width(maxWidth).Render(lipgloss.JoinVertical(lipgloss.Left, modelItems...)),
		scrollIndicator,
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
	if llmModels == nil || len(llmModels) == 0 {
		m.models = []models.Model{}
		m.scrollOffset = 0
		return
	}

	// Sort models in reverse alphabetical order
	sortedModels := make([]models.Model, len(llmModels))
	copy(sortedModels, llmModels)
	sort.Slice(sortedModels, func(i, j int) bool {
		return sortedModels[i].Name > sortedModels[j].Name
	})

	m.selectedIdx = 0
	for i, model := range sortedModels {
		if model.ID == selectedModelId {
			m.selectedIdx = i
			break
		}
	}

	// Set scroll position to keep selected model visible
	m.scrollOffset = 0
	if m.selectedIdx >= numVisibleModels {
		m.scrollOffset = m.selectedIdx - (numVisibleModels - 1)
	}

	m.models = sortedModels
}

func NewModelDialogCmp() ModelDialog {
	return &modelDialogCmp{
		models:       []models.Model{},
		selectedIdx:  0,
		scrollOffset: 0,
	}
}
