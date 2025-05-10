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

type CompletionItemI interface {
	utilComponents.SimpleListItem
	GetValue() string
	DisplayValue() string
}

func (ci *CompletionItem) Render(selected bool, width int) string {
	t := theme.CurrentTheme()
	baseStyle := styles.BaseStyle()

	itemStyle := baseStyle

	if selected {
		itemStyle = itemStyle.
			Width(width).
			Background(t.Background()).
			Foreground(t.Primary()).
			Bold(true)
	}

	title := itemStyle.Render(
		ci.GetValue(),
	)

	return title
}

func (ci *CompletionItem) DisplayValue() string {
	return ci.Title
}

func (ci *CompletionItem) GetValue() string {
	return ci.Value
}

func NewCompletionItem(completionItem CompletionItem) CompletionItemI {
	return &completionItem
}

type CompletionProvider interface {
	GetId() string
	GetEntry() CompletionItemI
	GetChildEntries(query string) ([]CompletionItemI, error)
}

type CompletionSelectedMsg struct {
	SearchString    string
	CompletionValue string
}

type CompletionDialogCompleteItemMsg struct {
	Value string
}

type CompletionDialogCloseMsg struct{}

type CompletionDialogInterruptUpdateMsg struct {
	InterrupCmd tea.Cmd
}

type CompletionDialog interface {
	tea.Model
	layout.Bindings
	SetWidth(width int)
}

type completionDialogCmp struct {
	query                string
	completionProvider   CompletionProvider
	width                int
	height               int
	pseudoSearchTextArea textarea.Model
	listView             utilComponents.SimpleList[CompletionItemI]
}

type completionDialogKeyMap struct {
	Enter     key.Binding
	Complete  key.Binding
	Cancel    key.Binding
	Backspace key.Binding
	Escape    key.Binding
	J         key.Binding
	K         key.Binding
	Start     key.Binding
}

var completionDialogKeys = completionDialogKeyMap{
	Start: key.NewBinding(
		key.WithKeys("@"),
	),
	Backspace: key.NewBinding(
		key.WithKeys("backspace"),
	),
	Complete: key.NewBinding(
		key.WithKeys("tab"),
	),
	Cancel: key.NewBinding(
		key.WithKeys(" ", "esc"),
	),
	// Enter: key.NewBinding(
	// 	key.WithKeys("enter"),
	// 	key.WithHelp("enter", "select item"),
	// ),
}

func (c *completionDialogCmp) Init() tea.Cmd {
	return nil
}

func (c *completionDialogCmp) complete(item CompletionItemI) tea.Cmd {
	value := c.pseudoSearchTextArea.Value()

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
	// c.listView.Reset()
	c.pseudoSearchTextArea.Reset()
	c.pseudoSearchTextArea.Blur()

	return util.CmdHandler(CompletionDialogCloseMsg{})
}

func (c *completionDialogCmp) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if c.pseudoSearchTextArea.Focused() {
			var cmd tea.Cmd
			c.pseudoSearchTextArea, cmd = c.pseudoSearchTextArea.Update(msg)
			cmds = append(cmds, cmd)

			var query string
			query = c.pseudoSearchTextArea.Value()
			if query != "" {
				query = query[1:]
			}

			if query != c.query {
				logging.Info("Query", query)
				items, err := c.completionProvider.GetChildEntries(query)
				if err != nil {
					logging.Error("Failed to get child entries", err)
				}

				c.listView.SetItems(items)
				c.query = query
			}

			u, cmd := c.listView.Update(msg)
			c.listView = u.(utilComponents.SimpleList[CompletionItemI])

			cmds = append(cmds, cmd)

			switch {
			// case key.Matches(msg, completionDialogKeys.Enter):
			// 	logging.Info("InterrupCmd1")
			// 	item, i := c.listView.GetSelectedItem()
			// 	if i == -1 {
			// 		logging.Info("InterrupCmd2", "i", i)
			// 		return c, nil
			// 	}
			//
			// 	cmd := c.complete(item)
			//
			// 	logging.Info("InterrupCmd")
			// 	return c, util.CmdHandler(CompletionDialogInterruptUpdateMsg{
			// 		InterrupCmd: cmd,
			// 	})
			case key.Matches(msg, completionDialogKeys.Complete):
				item, i := c.listView.GetSelectedItem()
				if i == -1 {
					return c, nil
				}

				cmd := c.complete(item)

				return c, cmd
			case key.Matches(msg, completionDialogKeys.Cancel) ||
				(key.Matches(msg, completionDialogKeys.Backspace) && len(c.pseudoSearchTextArea.Value()) <= 0):
				return c, c.close()
			}

			return c, tea.Batch(cmds...)
		}
		switch {
		case key.Matches(msg, completionDialogKeys.Start):
			c.pseudoSearchTextArea.SetValue(msg.String())
			return c, c.pseudoSearchTextArea.Focus()
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

func (c *completionDialogCmp) BindingKeys() []key.Binding {
	return layout.KeyMapToSlice(completionDialogKeys)
}

func NewCompletionDialogCmp(completionProvider CompletionProvider) CompletionDialog {
	ti := textarea.New()

	items, err := completionProvider.GetChildEntries("")
	if err != nil {
		logging.Error("Failed to get child entries", err)
	}

	li := utilComponents.NewSimpleList(items)

	return &completionDialogCmp{
		query:                "",
		completionProvider:   completionProvider,
		pseudoSearchTextArea: ti,
		listView:             li,
	}
}
