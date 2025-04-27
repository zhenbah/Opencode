package dialog

import (
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textarea"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/opencode-ai/opencode/internal/logging"
	utilComponents "github.com/opencode-ai/opencode/internal/tui/components/util"
	"github.com/opencode-ai/opencode/internal/tui/layout"
	"github.com/opencode-ai/opencode/internal/tui/styles"
	"github.com/opencode-ai/opencode/internal/tui/theme"
	"github.com/opencode-ai/opencode/internal/tui/util"
)

type CompletionItem struct {
	title string
	Title string
	Value string
}

func (ci *CompletionItem) DisplayValue() string {
	return ci.Title
}

func (ci *CompletionItem) GetValue() string {
	return ci.Value
}

type CompletionItemI interface {
	GetValue() string
	DisplayValue() string
}

func NewCompletionItem(completionItem CompletionItem) CompletionItemI {
	return &completionItem
}

type CompletionProvider interface {
	GetId() string
	GetEntry() CompletionItemI
	GetChildEntries() ([]CompletionItemI, error)
}

type CompletionSelectedMsg struct {
	SearchString    string
	CompletionValue string
}

type CompletionDialogCompleteItemMsg struct {
	Value string
}

type CompletionDialogCloseMsg struct{}

type CompletionDialog interface {
	tea.Model
	layout.Bindings
	SetWidth(width int)
}

type completionDialogCmp struct {
	completionProvider CompletionProvider
	completionItems    []CompletionItemI
	selectedIdx        int
	width              int
	height             int
	counter            int
	searchTextArea     textarea.Model
	listView           utilComponents.SimpleList
}

type completionDialogKeyMap struct {
	Up        key.Binding
	Down      key.Binding
	Enter     key.Binding
	Tab       key.Binding
	Space     key.Binding
	Backspace key.Binding
	Escape    key.Binding
	J         key.Binding
	K         key.Binding
	At        key.Binding
}

var completionDialogKeys = completionDialogKeyMap{
	At: key.NewBinding(
		key.WithKeys("@"),
	),
	Backspace: key.NewBinding(
		key.WithKeys("backspace"),
	),
	Tab: key.NewBinding(
		key.WithKeys("tab"),
	),
	Space: key.NewBinding(
		key.WithKeys(" "),
	),
	Up: key.NewBinding(
		key.WithKeys("up"),
		key.WithHelp("↑", "previous item"),
	),
	Down: key.NewBinding(
		key.WithKeys("down"),
		key.WithHelp("↓", "next item"),
	),
	Enter: key.NewBinding(
		key.WithKeys("enter"),
		key.WithHelp("enter", "select item"),
	),
	Escape: key.NewBinding(
		key.WithKeys("esc"),
		key.WithHelp("esc", "close"),
	),
	J: key.NewBinding(
		key.WithKeys("j"),
		key.WithHelp("j", "next item"),
	),
	K: key.NewBinding(
		key.WithKeys("k"),
		key.WithHelp("k", "previous item"),
	),
}

func (c *completionDialogCmp) Init() tea.Cmd {
	return nil
}

func (c *completionDialogCmp) complete(item CompletionItemI) tea.Cmd {
	value := c.searchTextArea.Value()

	if value == "" {
		return nil
	}

	return tea.Batch(
		util.CmdHandler(CompletionSelectedMsg{
			SearchString:    value,
			CompletionValue: item.GetValue(),
		}),
		c.close(),
	)
}

func (c *completionDialogCmp) close() tea.Cmd {
	c.listView.Reset()
	c.searchTextArea.Reset()
	c.searchTextArea.Blur()

	return util.CmdHandler(CompletionDialogCloseMsg{})
}

func (c *completionDialogCmp) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if c.searchTextArea.Focused() {
			var cmd tea.Cmd
			c.searchTextArea, cmd = c.searchTextArea.Update(msg)
			cmds = append(cmds, cmd)

			var query string
			query = c.searchTextArea.Value()
			if query != "" {
				query = query[1:]
			}

			logging.Info("Query", query)
			c.listView.Filter(query)

			u, cmd := c.listView.Update(msg)
			c.listView = u.(utilComponents.SimpleList)

			cmds = append(cmds, cmd)

			switch {
			case key.Matches(msg, completionDialogKeys.Tab):
				item, i := c.listView.GetSelectedItem()
				if i == -1 {
					return c, nil
				}
				var matchedItem CompletionItemI

				for _, citem := range c.completionItems {
					if item.GetValue() == citem.GetValue() {
						matchedItem = citem
					}
				}

				cmd := c.complete(matchedItem)

				return c, cmd
			case key.Matches(msg, completionDialogKeys.Escape) || key.Matches(msg, completionDialogKeys.Space) ||
				(key.Matches(msg, completionDialogKeys.Backspace) && len(c.searchTextArea.Value()) <= 0):
				return c, c.close()
			}

			return c, tea.Batch(cmds...)
		}
		switch {
		case key.Matches(msg, completionDialogKeys.At):
			c.searchTextArea.SetValue("@")
			return c, c.searchTextArea.Focus()
		}
	case tea.WindowSizeMsg:
		c.width = msg.Width
		c.height = msg.Height
	}

	return c, tea.Batch(cmds...)
}

func (c *completionDialogCmp) View() string {
	t := theme.CurrentTheme()
	baseStyle := styles.BaseStyle()

	return baseStyle.Padding(0, 0).
		Border(lipgloss.NormalBorder()).
		BorderBottom(false).
		BorderRight(false).
		BorderLeft(false).
		BorderBackground(t.Background()).
		BorderForeground(t.TextMuted()).
		Width(c.width).
		Render(c.listView.View())
}

func (c *completionDialogCmp) SetWidth(width int) {
	c.width = width
}

func mapperFunc(i CompletionItemI) utilComponents.SimpleListItem {
	return utilComponents.NewListItem(
		// i.DisplayValue(),
		// "",
		i.GetValue(),
	)
}

func (c *completionDialogCmp) BindingKeys() []key.Binding {
	return layout.KeyMapToSlice(completionDialogKeys)
}

func NewCompletionDialogCmp(completionProvider CompletionProvider) CompletionDialog {
	ti := textarea.New()
	items, err := completionProvider.GetChildEntries()
	li := utilComponents.NewSimpleList(utilComponents.MapSlice(items, mapperFunc))
	if err != nil {
		logging.Error("Failed to get child entries", err)
	}

	return &completionDialogCmp{
		completionProvider: completionProvider,
		completionItems:    items,
		selectedIdx:        0,
		counter:            0,
		searchTextArea:     ti,
		listView:           li,
	}
}
