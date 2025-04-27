package chat

import (
	"errors"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/filepicker"
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/opencode-ai/opencode/internal/logging"
	"github.com/opencode-ai/opencode/internal/message"
	"github.com/opencode-ai/opencode/internal/preview"
	"github.com/opencode-ai/opencode/internal/tui/styles"
	"github.com/opencode-ai/opencode/internal/tui/util"
)

var enterKey = key.NewBinding(
	key.WithKeys("enter"),
)
var down = key.NewBinding(
	key.WithKeys("j"),
)

var up = key.NewBinding(
	key.WithKeys("k"),
)
var forward = key.NewBinding(
	key.WithKeys("l"),
)
var enter = key.NewBinding(
	key.WithKeys("enter"),
)
var backward = key.NewBinding(
	key.WithKeys("h"),
)

var openFilepiceker = key.NewBinding(
	key.WithKeys("ctrl+f"),
)

type filepickerCmp struct {
	width       int
	height      int
	filepicker  filepicker.Model
	cursor      int
	err         error
	cursorChain stack
	imageString string
}
type stack []int

func (s stack) Push(v int) stack {
	return append(s, v)
}

func (s stack) Pop() (stack, int) {
	l := len(s)
	return s[:l-1], s[l-1]
}

type clearErrorMsg struct{}

type AttachmentAddedMsg struct {
	Attachment message.Attachment
}

func (f *filepickerCmp) Init() tea.Cmd {
	return f.filepicker.Init()
}

func (f *filepickerCmp) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		f.width = 80
		f.height = 10
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, down):
			f.cursor++
			f.getCurrentFileBelowCursor()
		case key.Matches(msg, up):
			f.cursor--
		case key.Matches(msg, enter):
			if didSelect, _ := f.filepicker.DidSelectFile(msg); !didSelect {
				f.cursorChain.Push(f.cursor)
				f.cursor = 0
			}
		case key.Matches(msg, forward):
			f.cursor = 0
		case key.Matches(msg, backward):
			if len(f.cursorChain) != 0 {
				f.cursorChain, f.cursor = f.cursorChain.Pop()
			}
		case key.Matches(msg, openFilepiceker):
			f.cursor = 0
		}
	}
	var cmd tea.Cmd
	f.filepicker, cmd = f.filepicker.Update(msg)

	if didSelect, path := f.filepicker.DidSelectFile(msg); didSelect {
		content, err := os.ReadFile(path)
		if err != nil {
			f.err = errors.New("unable to read the selected file")
			return f, tea.Batch(cmd, clearErrorAfter(2*time.Second))
		}

		mimeBufferSize := min(512, len(content))
		mimeType := http.DetectContentType(content[:mimeBufferSize])
		fileName := filepath.Base(path)
		attachment := message.Attachment{FilePath: path, FileName: fileName, MimeType: mimeType, Content: content}
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

	if f.imageString != "" {

		return lipgloss.JoinVertical(lipgloss.Left, styles.BaseStyle.Padding(1).
			Border(lipgloss.RoundedBorder()).
			BorderForeground(styles.ForgroundDim).
			Width(f.width).
			BorderBackground(styles.Background).
			Render(s.String()), f.imageString)
	}

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
	return &filepickerCmp{filepicker: fp, cursorChain: make(stack, 0)}
}

func (f *filepickerCmp) getCurrentFileBelowCursor() {
	dirs := readDir(f.filepicker.CurrentDirectory, f.filepicker.ShowHidden)
	dir := dirs[f.cursor]
	if !dir.IsDir() {
		f.imageString = preview.PreviewImage(f.filepicker.CurrentDirectory+"/"+dir.Name(), 200, 100)
	}
}

func readDir(path string, showHidden bool) []os.DirEntry {
	dirEntries, err := os.ReadDir(path)
	if err != nil {
		logging.ErrorPersist("Error while selecting files", err)
	}

	sort.Slice(dirEntries, func(i, j int) bool {
		if dirEntries[i].IsDir() == dirEntries[j].IsDir() {
			return dirEntries[i].Name() < dirEntries[j].Name()
		}
		return dirEntries[i].IsDir()
	})

	if showHidden {
		return dirEntries
	}
	var sanitizedDirEntries []os.DirEntry
	for _, dirEntry := range dirEntries {
		isHidden, _ := IsHidden(dirEntry.Name())
		if isHidden {
			continue
		}
		sanitizedDirEntries = append(sanitizedDirEntries, dirEntry)
	}
	return sanitizedDirEntries
}

func IsHidden(file string) (bool, error) {
	return strings.HasPrefix(file, "."), nil
}
