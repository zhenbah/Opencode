package chat

import (
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/viewport"
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
var returnKey = key.NewBinding(
	key.WithKeys("esc"),
	key.WithHelp("esc", "close"),
)

type filepickerCmp struct {
	basePath     string
	width        int
	height       int
	cursor       int
	err          error
	cursorChain  stack
	viewport     viewport.Model
	dirs         []os.DirEntry
	cwd          []string
	selectedFile string
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
	return nil
}

func (f *filepickerCmp) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		f.width = 60
		f.height = 20
		f.viewport.Width = 80
		f.viewport.Height = 23
		f.cursor = 0
		f.getCurrentFileBelowCursor()
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, down):
			if f.cursor < len(f.dirs)-1 {
				f.cursor++
				f.getCurrentFileBelowCursor()
			}
		case key.Matches(msg, up):
			if f.cursor > 0 {
				f.cursor--
				f.getCurrentFileBelowCursor()
			}
		case key.Matches(msg, enter):
			if f.dirs[f.cursor].IsDir() {
				f.cursorChain.Push(f.cursor)
				f.cwd = append(f.cwd, f.dirs[f.cursor].Name())
				f.cursor = 0
			} else {
				return f.addAttachmentToMessage()
			}
		case key.Matches(msg, returnKey):
			f.cursorChain = make(stack, 0)
			f.cursor = 0
		case key.Matches(msg, forward):
			if f.dirs[f.cursor].IsDir() {
				f.cwd = append(f.cwd, f.dirs[f.cursor].Name())
				f.cursorChain = f.cursorChain.Push(f.cursor)
				f.dirs = f.getCWDFiles()
				f.cursor = 0
			}
		case key.Matches(msg, backward):
			if len(f.cursorChain) != 0 {
				f.cursorChain, f.cursor = f.cursorChain.Pop()
				f.cwd = f.cwd[:len(f.cwd)-1]
				f.dirs = readDir(f.basePath+strings.Join(f.cwd, "/"), false)
				if f.dirs[f.cursor].IsDir() {
					f.dirs = f.getCWDFiles()
				}
			}
		case key.Matches(msg, openFilepiceker):
			f.dirs = f.getCWDFiles()
			f.cursor = 0
			f.getCurrentFileBelowCursor()
		}
	}
	return f, nil
}

func (f *filepickerCmp) addAttachmentToMessage() (tea.Model, tea.Cmd) {

	var cmd tea.Cmd

	if isExtSupported(f.dirs[f.cursor].Name()) {
		f.selectedFile = f.dirs[f.cursor].Name()
		selectedFilePath := f.basePath + strings.Join(f.cwd, "/") + "/" + f.selectedFile
		isvalid, err := preview.ValidateFileSize(selectedFilePath, int64(5*1024*1024))
		if err != nil {
			logging.ErrorPersist("unable to read the image")
			return f, nil
		}
		if !isvalid {
			logging.ErrorPersist("file too large, max 5MB")
			return f, nil
		}

		content, err := os.ReadFile(f.basePath + strings.Join(f.cwd, "/") + "/" + f.selectedFile)
		if err != nil {
			f.selectedFile = ""
			f.err = errors.New("unable to read the selected file")
			return f, tea.Batch(cmd, clearErrorAfter(2*time.Second))
		}

		mimeBufferSize := min(512, len(content))
		mimeType := http.DetectContentType(content[:mimeBufferSize])
		fileName := f.selectedFile
		attachment := message.Attachment{FilePath: selectedFilePath, FileName: fileName, MimeType: mimeType, Content: content}
		f.selectedFile = ""
		return f, util.CmdHandler(AttachmentAddedMsg{attachment})
	}
	if !strings.HasSuffix(f.selectedFile, ".png") &&
		!strings.HasSuffix(f.selectedFile, ".jpg") &&
		!strings.HasSuffix(f.selectedFile, ".jpeg") &&
		!strings.HasSuffix(f.selectedFile, ".webp") {
		f.err = errors.New(f.selectedFile + " is not valid.")
		f.selectedFile = ""
		return f, tea.Batch(cmd, clearErrorAfter(2*time.Second))

	}
	return f, nil
}

func clearErrorAfter(t time.Duration) tea.Cmd {
	return tea.Tick(t, func(_ time.Time) tea.Msg {
		return clearErrorMsg{}
	})
}

func (f *filepickerCmp) View() string {
	const maxVisibleDirs = 20
	const maxWidth = 80

	adjustedWidth := maxWidth
	for _, file := range f.dirs {
		if len(file.Name()) > adjustedWidth-4 { // Account for padding
			adjustedWidth = len(file.Name()) + 4
		}
	}
	adjustedWidth = max(30, min(adjustedWidth, f.width-15))

	files := make([]string, 0, maxVisibleDirs)
	startIdx := 0

	if len(f.dirs) > maxVisibleDirs {
		halfVisible := maxVisibleDirs / 2
		if f.cursor >= halfVisible && f.cursor < len(f.dirs)-halfVisible {
			startIdx = f.cursor - halfVisible
		} else if f.cursor >= len(f.dirs)-halfVisible {
			startIdx = len(f.dirs) - maxVisibleDirs
		}
	}

	endIdx := min(startIdx+maxVisibleDirs, len(f.dirs))

	for i := startIdx; i < endIdx; i++ {
		file := f.dirs[i]
		itemStyle := styles.BaseStyle.Width(adjustedWidth)

		if i == f.cursor {
			itemStyle = itemStyle.
				Background(styles.PrimaryColor).
				Foreground(styles.Background).
				Bold(true)
		}
		filename := file.Name()

		if len(filename) > adjustedWidth-4 {
			filename = filename[:adjustedWidth-7] + "..."
		}
		files = append(files, itemStyle.Padding(0, 1).Render(filename))
	}

	// Pad to always show exactly 21 lines
	for len(files) < maxVisibleDirs {
		files = append(files, styles.BaseStyle.Width(adjustedWidth).Render(""))
	}

	title := styles.BaseStyle.
		Foreground(styles.PrimaryColor).
		Bold(true).
		Width(adjustedWidth).
		Padding(0, 1).
		Render("Pick a file")

	viewportstyle := lipgloss.NewStyle().
		Width(f.viewport.Width-2).
		Background(styles.Background).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(styles.ForgroundDim).
		Padding(1, 1, 1, 1).
		Render(f.viewport.View())

	content := lipgloss.JoinVertical(
		lipgloss.Left,
		title,
		styles.BaseStyle.Width(adjustedWidth).Render(""),
		styles.BaseStyle.Width(adjustedWidth).Render(lipgloss.JoinVertical(lipgloss.Left, files...)),
		styles.BaseStyle.Width(adjustedWidth).Render(""),
	)

	return lipgloss.JoinHorizontal(lipgloss.Center, styles.BaseStyle.Padding(1, 2).
		Border(lipgloss.RoundedBorder()).
		BorderBackground(styles.Background).
		BorderForeground(styles.ForgroundDim).
		Width(lipgloss.Width(content)+4).
		Render(content), viewportstyle)
}

type FilepickerCmp interface {
	tea.Model
}

func NewFilepickerCmp() FilepickerCmp {
	homepath, err := os.UserHomeDir()
	if err != nil {
		logging.Error("error loading use files")
		return nil
	}

	dirs := readDir(homepath, false)
	viewport := viewport.New(0, 0)
	return &filepickerCmp{basePath: homepath + "/", dirs: dirs, cursorChain: make(stack, 0), viewport: viewport}
}

func (f *filepickerCmp) getCWDFiles() []os.DirEntry {
	dirs := readDir(f.basePath+strings.Join(f.cwd, "/"), false)
	return dirs
}
func (f *filepickerCmp) getCurrentFileBelowCursor() {
	if len(f.dirs) == 0 || f.cursor < 0 || f.cursor >= len(f.dirs) {
		logging.Error(fmt.Sprintf("Invalid cursor position. Dirs length: %d, Cursor: %d", len(f.dirs), f.cursor))
		f.viewport.SetContent("Preview unavailable")
		return
	}

	dir := f.dirs[f.cursor]
	filename := dir.Name()
	if !dir.IsDir() && isExtSupported(filename) {
		fullPath := f.basePath + strings.Join(f.cwd, "/") + "/" + dir.Name()

		go func() {
			imageString, err := preview.ImagePreview(f.viewport.Width-4, fullPath)
			if err != nil {
				logging.ErrorPersist(err.Error())
				f.viewport.SetContent("Preview unavailable")
				return
			}

			f.viewport.SetContent(imageString)
		}()
	} else {
		f.viewport.SetContent("Preview unavailable")
	}
}

func readDir(path string, showHidden bool) []os.DirEntry {
	logging.Info(fmt.Sprintf("Reading directory: %s", path))

	entriesChan := make(chan []os.DirEntry, 1)
	errChan := make(chan error, 1)

	go func() {
		dirEntries, err := os.ReadDir(path)
		if err != nil {
			errChan <- err
			return
		}
		entriesChan <- dirEntries
	}()

	select {
	case dirEntries := <-entriesChan:
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
			if !isHidden {
				sanitizedDirEntries = append(sanitizedDirEntries, dirEntry)
			}
		}

		return sanitizedDirEntries

	case err := <-errChan:
		logging.ErrorPersist(fmt.Sprintf("Error reading directory %s", path), err)
		return []os.DirEntry{}

	case <-time.After(5 * time.Second):
		logging.ErrorPersist(fmt.Sprintf("Timeout reading directory %s", path), nil)
		return []os.DirEntry{}
	}
}

func IsHidden(file string) (bool, error) {
	return strings.HasPrefix(file, "."), nil
}

func isExtSupported(path string) bool {

	ext := strings.ToLower(filepath.Ext(path))
	return (ext == ".jpg" || ext == ".jpeg" || ext == ".webp" || ext == ".png")
}
