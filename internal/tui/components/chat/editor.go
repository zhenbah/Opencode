package chat

import (
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textarea"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/opencode-ai/opencode/internal/app"
	"github.com/opencode-ai/opencode/internal/logging"
	"github.com/opencode-ai/opencode/internal/session"
	"github.com/opencode-ai/opencode/internal/tui/layout"
	"github.com/opencode-ai/opencode/internal/tui/styles"
	"github.com/opencode-ai/opencode/internal/tui/util"
	"os"
	"os/exec"
	"slices"
)

type editorCmp struct {
	app         *app.App
	session     session.Session
	textarea    textarea.Model
	attachments []Attachment
	deleteMode  bool
}

type EditorKeyMaps struct {
	Send       key.Binding
	OpenEditor key.Binding
}

type bluredEditorKeyMaps struct {
	Send       key.Binding
	Focus      key.Binding
	OpenEditor key.Binding
}

var editorMaps = EditorKeyMaps{
	Send: key.NewBinding(
		key.WithKeys("enter", "ctrl+s"),
		key.WithHelp("enter", "send message"),
	),
	OpenEditor: key.NewBinding(
		key.WithKeys("ctrl+e"),
		key.WithHelp("ctrl+e", "open editor"),
	),
}

var deleteAttachmentsKey = key.NewBinding(
	key.WithKeys("ctrl+d"),
	key.WithHelp("ctrl+d", "delete attachmets"),
)

var deleteFirst = key.NewBinding(
	key.WithKeys("1"),
	key.WithHelp("1", "delete 1st attachmet"),
)

var deleteSecond = key.NewBinding(
	key.WithKeys("2"),
	key.WithHelp("2", "delete 2nd attachmet"),
)
var deleteThird = key.NewBinding(
	key.WithKeys("3"),
	key.WithHelp("3", "delete 3rd attachmet"),
)

var deleteFourth = key.NewBinding(
	key.WithKeys("4"),
	key.WithHelp("4", "delete 4th attachmet"),
)

var deleteFifth = key.NewBinding(
	key.WithKeys("5"),
	key.WithHelp("5", "delete 5th attachmet"),
)
var deleteAll = key.NewBinding(
	key.WithKeys("d"),
	key.WithHelp("d", "delete all attachmet"),
)

func openEditor() tea.Cmd {
	editor := os.Getenv("EDITOR")
	if editor == "" {
		editor = "nvim"
	}

	tmpfile, err := os.CreateTemp("", "msg_*.md")
	if err != nil {
		return util.ReportError(err)
	}
	tmpfile.Close()
	c := exec.Command(editor, tmpfile.Name()) //nolint:gosec
	c.Stdin = os.Stdin
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	return tea.ExecProcess(c, func(err error) tea.Msg {
		if err != nil {
			return util.ReportError(err)
		}
		content, err := os.ReadFile(tmpfile.Name())
		if err != nil {
			return util.ReportError(err)
		}
		if len(content) == 0 {
			return util.ReportWarn("Message is empty")
		}
		os.Remove(tmpfile.Name())
		return SendMsg{
			Text: string(content),
		}
	})
}

func (m *editorCmp) Init() tea.Cmd {
	return textarea.Blink
}

func (m *editorCmp) send() tea.Cmd {
	if m.app.CoderAgent.IsSessionBusy(m.session.ID) {
		return util.ReportWarn("Agent is working, please wait...")
	}

	value := m.textarea.Value()
	m.textarea.Reset()
	if value == "" {
		return nil
	}
	return tea.Batch(
		util.CmdHandler(SendMsg{
			Text:        value,
			Attachments: m.attachments,
		}),
	)
}

func (m *editorCmp) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	switch msg := msg.(type) {
	case SessionSelectedMsg:
		if msg.ID != m.session.ID {
			m.session = msg
		}
		return m, nil
	case AttachmentAddedMsg:
		if len(m.attachments) >= 5 {
			logging.Error("cannot add more than 5 images")
			return m, nil
		}
		m.attachments = append(m.attachments, msg.Attachment)
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, deleteAttachmentsKey):
			m.deleteMode = true
		case key.Matches(msg, deleteFirst):
			if m.deleteMode {
				if len(m.attachments) >= 1 {
					m.deleteMode = false
					m.attachments = slices.Delete(m.attachments, 0, 1)
					return m, nil
				}
			}
		case key.Matches(msg, deleteSecond):
			if m.deleteMode {
				if len(m.attachments) >= 2 {
					m.attachments = slices.Delete(m.attachments, 1, 2)
					m.deleteMode = false
					return m, nil
				}
			}
		case key.Matches(msg, deleteThird):
			if m.deleteMode {
				if len(m.attachments) >= 3 {
					m.attachments = slices.Delete(m.attachments, 2, 3)
					m.deleteMode = false
					return m, nil
				}
			}
		case key.Matches(msg, deleteFourth):
			if m.deleteMode {
				if len(m.attachments) >= 4 {
					m.attachments = slices.Delete(m.attachments, 3, 4)
					m.deleteMode = false
					return m, nil
				}
			}
		case key.Matches(msg, deleteFifth):
			if m.deleteMode {
				if len(m.attachments) >= 5 {
					m.attachments = m.attachments[:4]
					m.deleteMode = false
					return m, nil
				}
			}
		case key.Matches(msg, deleteAll):
			if m.deleteMode {
				if len(m.attachments) >= 1 {
					m.deleteMode = false
					m.attachments = nil
					return m, nil
				}
			}
		}
		if key.Matches(msg, messageKeys.PageUp) || key.Matches(msg, messageKeys.PageDown) ||
			key.Matches(msg, messageKeys.HalfPageUp) || key.Matches(msg, messageKeys.HalfPageDown) {
			return m, nil
		}
		if key.Matches(msg, editorMaps.OpenEditor) {
			if m.app.CoderAgent.IsSessionBusy(m.session.ID) {
				return m, util.ReportWarn("Agent is working, please wait...")
			}
			return m, openEditor()
		}
		// Handle Enter key
		if m.textarea.Focused() && key.Matches(msg, editorMaps.Send) {
			value := m.textarea.Value()
			if len(value) > 0 && value[len(value)-1] == '\\' {
				// If the last character is a backslash, remove it and add a newline
				m.textarea.SetValue(value[:len(value)-1] + "\n")
				return m, nil
			} else {
				// Otherwise, send the message
				return m, m.send()
			}
		}
	}
	m.textarea, cmd = m.textarea.Update(msg)
	return m, cmd
}

func (m *editorCmp) View() string {
	style := lipgloss.NewStyle().Padding(0, 0, 0, 1).Bold(true)
	var attachments string
	for _, attachment := range m.attachments {
		attachments += "  " + attachment.FileName
	}
	return lipgloss.JoinVertical(lipgloss.Top, attachments, lipgloss.JoinHorizontal(lipgloss.Top, style.Render(">"), m.textarea.View()))
}

func (m *editorCmp) SetSize(width, height int) tea.Cmd {
	m.textarea.SetWidth(width - 3) // account for the prompt and padding right
	m.textarea.SetHeight(height)
	return nil
}

func (m *editorCmp) GetSize() (int, int) {
	return m.textarea.Width(), m.textarea.Height()
}

func (m *editorCmp) BindingKeys() []key.Binding {
	bindings := []key.Binding{}
	bindings = append(bindings, layout.KeyMapToSlice(editorMaps)...)
	return bindings
}

func NewEditorCmp(app *app.App) tea.Model {
	ti := textarea.New()
	ti.Prompt = " "
	ti.ShowLineNumbers = false
	ti.BlurredStyle.Base = ti.BlurredStyle.Base.Background(styles.Background)
	ti.BlurredStyle.CursorLine = ti.BlurredStyle.CursorLine.Background(styles.Background)
	ti.BlurredStyle.Placeholder = ti.BlurredStyle.Placeholder.Background(styles.Background)
	ti.BlurredStyle.Text = ti.BlurredStyle.Text.Background(styles.Background)

	ti.FocusedStyle.Base = ti.FocusedStyle.Base.Background(styles.Background)
	ti.FocusedStyle.CursorLine = ti.FocusedStyle.CursorLine.Background(styles.Background)
	ti.FocusedStyle.Placeholder = ti.FocusedStyle.Placeholder.Background(styles.Background)
	ti.FocusedStyle.Text = ti.BlurredStyle.Text.Background(styles.Background)
	ti.CharLimit = -1
	ti.Focus()
	return &editorCmp{
		app:      app,
		textarea: ti,
	}
}
