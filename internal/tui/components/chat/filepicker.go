package chat

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/filepicker"
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/opencode-ai/opencode/internal/tui/styles"
	"github.com/opencode-ai/opencode/internal/tui/util"
)

type filepickerCmp struct {
	width      int
	height     int
	keys       []key.Binding
	filepicker filepicker.Model
	quitting   bool
	err        error
}

type Attachment struct {
	FilePath string
	FileName string
	Content  []byte
}

type clearErrorMsg struct{}

type AttachmentAddedMsg struct {
	Attachment Attachment
}

func (f *filepickerCmp) Init() tea.Cmd {
	return f.filepicker.Init()
}

func (f *filepickerCmp) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg.(type) {
	case tea.WindowSizeMsg:
		f.width = 60
		f.height = 10
	}
	var cmd tea.Cmd
	f.filepicker, cmd = f.filepicker.Update(msg)

	if didSelect, path := f.filepicker.DidSelectFile(msg); didSelect {
		content, err := os.ReadFile(path)
		if err != nil {
			f.err = errors.New("unable to read the selected file")
			return f, tea.Batch(cmd, clearErrorAfter(2*time.Second))
		}
		fileName := filepath.Base(path)
		attachment := Attachment{FilePath: path, FileName: fileName, Content: content}
		return f, util.CmdHandler(AttachmentAddedMsg{attachment})
	}

	if didSelect, path := f.filepicker.DidSelectDisabledFile(msg); didSelect {
		f.err = errors.New(path + " is not valid.")
		return f, tea.Batch(cmd, clearErrorAfter(2*time.Second))
	}

	return f, cmd
}

func clearErrorAfter(t time.Duration) tea.Cmd {
	return tea.Tick(t, func(_ time.Time) tea.Msg {
		return clearErrorMsg{}
	})
}

func (f *filepickerCmp) View() string {
	var s strings.Builder

	headerStyle := styles.BaseStyle.
		Bold(true).
		Foreground(styles.PrimaryColor)
	s.WriteString("\n  ")
	if f.err != nil {
		s.WriteString(f.filepicker.Styles.DisabledFile.Render(f.err.Error()))
	}
	s.WriteString(headerStyle.Render("Pick a file:"))
	s.WriteString("\n\n" + f.filepicker.View() + "\n")

	return styles.BaseStyle.Padding(1).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(styles.ForgroundDim).
		Width(f.width).
		BorderBackground(styles.Background).
		Render(s.String())
}

type FilepickerCmp interface {
	tea.Model
}

func NewFilepickerCmp() FilepickerCmp {
	fp := filepicker.New()
	fp.CurrentDirectory, _ = os.UserHomeDir()
	fp.Height = 10
	fp.AutoHeight = false
	r := lipgloss.NewStyle()
	fp.Styles = filepicker.Styles{
		DisabledCursor:   r.Foreground(styles.Grey),
		Symlink:          r.Foreground(lipgloss.Color("36")),
		Directory:        r.Foreground(styles.Forground),
		DisabledFile:     r.Foreground(styles.Grey),
		DisabledSelected: r.Foreground(styles.Grey),
		Permission:       r.Foreground(styles.Lavender),
		Selected:         r.Foreground(styles.PrimaryColor).Bold(true),
	}
	fp.ShowSize = false
	fp.AllowedTypes = []string{".png", ".jpg", ".jpeg", ".webp"}
	return &filepickerCmp{filepicker: fp}
}
