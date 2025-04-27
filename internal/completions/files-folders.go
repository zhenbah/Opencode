package completions

import (
	"bytes"
	"fmt"
	"os/exec"
	"path/filepath"
	"sort"

	"github.com/opencode-ai/opencode/internal/tui/components/dialog"
)

type filesAndFoldersContextGroup struct {
	prefix string
}

func (cg *filesAndFoldersContextGroup) GetId() string {
	return cg.prefix
}

func (cg *filesAndFoldersContextGroup) GetEntry() dialog.CompletionItemI {
	return dialog.NewCompletionItem(dialog.CompletionItem{
		Title: "Files & Folders",
		Value: "files",
	})
}

func getFilesRg() ([]string, error) {
	searchRoot := "."

	rgBin, err := exec.LookPath("rg")
	if err != nil {
		return nil, fmt.Errorf("ripgrep not found in $PATH: %w", err)
	}

	args := []string{
		"--files",
		"-L",
		"--null",
	}

	cmd := exec.Command(rgBin, args...)
	cmd.Dir = "."

	out, err := cmd.CombinedOutput()
	if err != nil {
		if ee, ok := err.(*exec.ExitError); ok && ee.ExitCode() == 1 {
			return nil, nil
		}
		return nil, fmt.Errorf("ripgrep: %w\n%s", err, out)
	}

	var matches []string
	for _, p := range bytes.Split(out, []byte{0}) {
		if len(p) == 0 {
			continue
		}
		abs := filepath.Join(searchRoot, string(p))
		matches = append(matches, abs)
	}

	sort.SliceStable(matches, func(i, j int) bool {
		return len(matches[i]) < len(matches[j])
	})

	return matches, nil
}

func (cg *filesAndFoldersContextGroup) GetChildEntries() ([]dialog.CompletionItemI, error) {
	matches, err := getFilesRg()
	if err != nil {
		return nil, err
	}

	items := make([]dialog.CompletionItemI, 0)

	for _, file := range matches {
		item := dialog.NewCompletionItem(dialog.CompletionItem{
			Title: file,
			Value: file,
		})
		items = append(items, item)
	}

	return items, nil
}

func NewFileAndFolderContextGroup() dialog.CompletionProvider {
	return &filesAndFoldersContextGroup{prefix: "file"}
}
