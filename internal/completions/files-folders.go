package completions

import (
	"bytes"
	"fmt"
	"os/exec"
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

func getFilesRg(query string) ([]string, error) {
	searchRoot := "."

	rgBin, err := exec.LookPath("rg")
	if err != nil {
		return nil, fmt.Errorf("ripgrep not found in $PATH: %w", err)
	}

	fzfBin, err := exec.LookPath("fzf")
	if err != nil {
		return nil, fmt.Errorf("fzf not found in $PATH: %w", err)
	}

	rgArgs := []string{
		"--files",
		"-L",
		"--null",
	}
	cmdRg := exec.Command(rgBin, rgArgs...)
	cmdRg.Dir = searchRoot

	rgPipe, err := cmdRg.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to get rg stdout pipe: %w", err)
	}
	defer rgPipe.Close()

	fzfArgs := []string{
		"--filter",
		query,
		"--read0",
		"--print0",
	}
	cmdFzf := exec.Command(fzfBin, fzfArgs...)
	cmdFzf.Dir = searchRoot
	cmdFzf.Stdin = rgPipe
	var fzfOut bytes.Buffer
	var fzfErr bytes.Buffer
	cmdFzf.Stdout = &fzfOut
	cmdFzf.Stderr = &fzfErr

	if err := cmdFzf.Start(); err != nil {
		return nil, fmt.Errorf("failed to start fzf: %w", err)
	}

	errRg := cmdRg.Run()

	errFzf := cmdFzf.Wait()

	if errRg != nil {
		return nil, fmt.Errorf("rg command failed: %w", errRg)
	}

	if errFzf != nil {
		if exitErr, ok := errFzf.(*exec.ExitError); ok && exitErr.ExitCode() == 1 {
			return []string{}, nil
		}
		return nil, fmt.Errorf("fzf command failed: %w\nStderr: %s", errFzf, fzfErr.String())
	}

	outputBytes := fzfOut.Bytes()
	if len(outputBytes) > 0 && outputBytes[len(outputBytes)-1] == 0 {
		outputBytes = outputBytes[:len(outputBytes)-1]
	}

	if len(outputBytes) == 0 {
		return []string{}, nil
	}

	split := bytes.Split(outputBytes, []byte{0})
	matches := make([]string, 0, len(split))

	for _, p := range split {
		if len(p) == 0 {
			continue
		}
		matches = append(matches, string(p))
	}

	sort.SliceStable(matches, func(i, j int) bool {
		if len(matches[i]) != len(matches[j]) {
			return len(matches[i]) < len(matches[j])
		}
		return matches[i] < matches[j]
	})

	return matches, nil
}

func (cg *filesAndFoldersContextGroup) GetChildEntries(query string) ([]dialog.CompletionItemI, error) {
	matches, err := getFilesRg(query)
	if err != nil {
		return nil, err
	}

	items := make([]dialog.CompletionItemI, 0, len(matches))

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
