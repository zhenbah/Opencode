package utilComponents

import (
	"strings"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/opencode-ai/opencode/internal/tui/layout"
	"github.com/opencode-ai/opencode/internal/tui/styles"
	"github.com/opencode-ai/opencode/internal/tui/theme"
)

type listItem struct {
	// title       string
	// description string
	value string
}

func (li *listItem) GetValue() string {
	return li.value
}

type SimpleListItem interface {
	GetValue() string
}

func NewListItem(
	// title string,
	// description string,
	value string,
) SimpleListItem {
	return &listItem{
		// title:       title,
		// description: description,
		value: value,
	}
}

type SimpleList interface {
	tea.Model
	layout.Bindings
	GetSelectedItem() (item SimpleListItem, idx int)
	SetItems(items []SimpleListItem)
	Filter(query string)
	Reset()
}

type simpleListCmp struct {
	items         []SimpleListItem
	filtereditems []SimpleListItem
	query         string
	selectedIdx   int
	width         int
	height        int
}

type simpleListKeyMap struct {
	Up    key.Binding
	Down  key.Binding
	Enter key.Binding
}

var simpleListKeys = simpleListKeyMap{
	Up: key.NewBinding(
		key.WithKeys("up"),
		key.WithHelp("↑", "previous list item"),
	),
	Down: key.NewBinding(
		key.WithKeys("down"),
		key.WithHelp("↓", "next list item"),
	),
	Enter: key.NewBinding(
		key.WithKeys("enter"),
		key.WithHelp("enter", "select item"),
	),
}

func (c *simpleListCmp) Init() tea.Cmd {
	return nil
}

func (c *simpleListCmp) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, simpleListKeys.Up):
			if c.selectedIdx > 0 {
				c.selectedIdx--
			}
			return c, nil
		case key.Matches(msg, simpleListKeys.Down):
			if c.selectedIdx < len(c.filtereditems)-1 {
				c.selectedIdx++
			}
			return c, nil
		}
	}

	return c, nil
}

func (c *simpleListCmp) BindingKeys() []key.Binding {
	return layout.KeyMapToSlice(simpleListKeys)
}

func (c *simpleListCmp) GetSelectedItem() (SimpleListItem, int) {
	if len(c.filtereditems) > 0 {
		return c.filtereditems[c.selectedIdx], c.selectedIdx
	}

	return nil, -1
}

func (c *simpleListCmp) SetItems(items []SimpleListItem) {
	c.selectedIdx = 0
	c.items = items
}

func (c *simpleListCmp) Filter(query string) {
	if query == c.query {
		return
	}
	c.query = query

	filteredItems := make([]SimpleListItem, 0)

	for _, item := range c.items {
		if strings.Contains(item.GetValue(), query) {
			filteredItems = append(filteredItems, item)
		}
	}

	c.selectedIdx = 0
	c.filtereditems = filteredItems
}

func (c *simpleListCmp) Reset() {
	c.selectedIdx = 0
	c.filtereditems = c.items
}

func MapSlice[In any](input []In, mapper func(In) SimpleListItem) []SimpleListItem {
	if input == nil {
		return nil
	}

	output := make([]SimpleListItem, len(input))

	for i, element := range input {
		output[i] = mapper(element)
	}

	return output
}

func (c *simpleListCmp) View() string {
	t := theme.CurrentTheme()
	baseStyle := styles.BaseStyle()

	items := c.filtereditems
	maxWidth := 80
	maxVisibleCommands := 7
	startIdx := 0

	if len(items) <= 0 {
		return "No content available"
	}

	if len(items) > maxVisibleCommands {
		// Center the selected item when possible
		halfVisible := maxVisibleCommands / 2
		if c.selectedIdx >= halfVisible && c.selectedIdx < len(items)-halfVisible {
			startIdx = c.selectedIdx - halfVisible
		} else if c.selectedIdx >= len(items)-halfVisible {
			startIdx = len(items) - maxVisibleCommands
		}
	}

	endIdx := min(startIdx+maxVisibleCommands, len(items))

	// return styles.BaseStyle.Padding(0, 0).
	// 	Render(fmt.Sprintf("lenItems %d, s: %d, e: %d", len(c.items), startIdx+maxVisibleCommands, endIdx))

	listItems := make([]string, 0, maxVisibleCommands)

	for i := startIdx; i < endIdx; i++ {
		item := items[i]
		itemStyle := baseStyle.Width(maxWidth)
		// descStyle := styles.BaseStyle.Width(maxWidth).Foreground(styles.ForgroundDim)

		if i == c.selectedIdx {
			itemStyle = itemStyle.
				Background(t.Background()).
				Foreground(t.Primary()).
				Bold(true)
			// descStyle = descStyle.
			// 	Background(styles.PrimaryColor).
			// 	Foreground(styles.Background)
		}

		title := itemStyle.Render(
			item.GetValue(),
		)
		listItems = append(listItems, title)
	}

	return lipgloss.JoinVertical(
		lipgloss.Left,
		baseStyle.
			Width(maxWidth).
			Padding(0, 1).
			Render(
				lipgloss.
					JoinVertical(lipgloss.Left, listItems...),
			),
	)
}

func NewSimpleList(items []SimpleListItem) SimpleList {
	return &simpleListCmp{
		items:         items,
		filtereditems: items,
		selectedIdx:   0,
		query:         "",
		// selectedStyle:     lipgloss.NewStyle().Foreground(lipgloss.Color("205")).Bold(true),
		// normalStyle:       lipgloss.NewStyle(),
	}
}
