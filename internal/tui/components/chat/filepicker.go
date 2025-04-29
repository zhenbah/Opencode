package chat

import (
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

const (
	maxAttachmentSize = int64(5 * 1024 * 1024)
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
	cwd          *DirNode
	selectedFile string
}

type DirNode struct {
	parent    *DirNode
	child     *DirNode
	directory string
}
type stack []int

func (s stack) Push(v int) stack {
	return append(s, v)
}

func (s stack) Pop() (stack, int) {
	l := len(s)
	return s[:l-1], s[l-1]
}

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
				newWorkingDir := DirNode{parent: f.cwd, directory: f.cwd.directory + "/" + f.dirs[f.cursor].Name()}
				f.cwd.child = &newWorkingDir
				f.cwd = f.cwd.child
				f.cursor = 0
			} else {
				return f.addAttachmentToMessage()
			}
		case key.Matches(msg, returnKey):
			f.cursorChain = make(stack, 0)
			f.cursor = 0
		case key.Matches(msg, forward):
			if f.dirs[f.cursor].IsDir() {
				newWorkingDir := DirNode{parent: f.cwd, directory: f.cwd.directory + "/" + f.dirs[f.cursor].Name()}
				f.cwd.child = &newWorkingDir
				f.cwd = f.cwd.child
				f.cursorChain = f.cursorChain.Push(f.cursor)
				f.dirs = readDir(f.cwd.directory, false)
				f.cursor = 0
			}
		case key.Matches(msg, backward):
			if len(f.cursorChain) != 0 && f.cwd.parent != nil {
				f.cursorChain, f.cursor = f.cursorChain.Pop()

				f.cwd = f.cwd.parent
				f.cwd.child = nil
				f.dirs = readDir(f.cwd.directory, false)
			}
		case key.Matches(msg, openFilepiceker):
			f.dirs = readDir(f.cwd.directory, false)
			f.cursor = 0
			f.getCurrentFileBelowCursor()
		}
	}
	return f, nil
}

func (f *filepickerCmp) addAttachmentToMessage() (tea.Model, tea.Cmd) {
	if isExtSupported(f.dirs[f.cursor].Name()) {
		f.selectedFile = f.dirs[f.cursor].Name()
		selectedFilePath := f.cwd.directory + "/" + f.selectedFile
		isFileLarge, err := preview.ValidateFileSize(selectedFilePath, maxAttachmentSize)
		if err != nil {
			logging.ErrorPersist("unable to read the image")
			return f, nil
		}
		if isFileLarge {
			logging.ErrorPersist("file too large, max 5MB")
			return f, nil
		}

		content, err := os.ReadFile(f.cwd.directory + "/" + f.selectedFile)
		if err != nil {
			logging.ErrorPersist("Unable read selected file")
			return f, nil
		}

		mimeBufferSize := min(512, len(content))
		mimeType := http.DetectContentType(content[:mimeBufferSize])
		fileName := f.selectedFile
		attachment := message.Attachment{FilePath: selectedFilePath, FileName: fileName, MimeType: mimeType, Content: content}
		f.selectedFile = ""
		return f, util.CmdHandler(AttachmentAddedMsg{attachment})
	}
	if !isExtSupported(f.selectedFile) {
		logging.ErrorPersist("Unsupported file")
		return f, nil
	}
	return f, nil
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
		if file.IsDir() {
			filename = "\ue6ad " + filename
		} else if isExtSupported(file.Name()) {
			filename = "\uf03e " + filename
		} else {
			filename = "\uf15b " + filename
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
	baseDir := DirNode{parent: nil, directory: homepath}
	dirs := readDir(homepath, false)
	viewport := viewport.New(0, 0)
	return &filepickerCmp{cwd: &baseDir, dirs: dirs, cursorChain: make(stack, 0), viewport: viewport}
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
		fullPath := f.cwd.directory + "/" + dir.Name()

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
			logging.ErrorPersist(err.Error())
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
