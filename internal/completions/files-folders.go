package completions

import (
	"bytes"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/bmatcuk/doublestar/v4"
	"github.com/lithammer/fuzzysearch/fuzzy"
	"github.com/opencode-ai/opencode/internal/logging"
	"github.com/opencode-ai/opencode/internal/tui/components/dialog"
)

type filesAndFoldersContextGroup struct {
	rgPath  string
	fzfPath string
	prefix  string
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

type fileInfo struct {
	path    string
	modTime time.Time
}

type GlobParams struct {
	Pattern string `json:"pattern"`
	Path    string `json:"path"`
}

type GlobResponseMetadata struct {
	NumberOfFiles int  `json:"number_of_files"`
	Truncated     bool `json:"truncated"`
}

func globWithDoublestar() ([]string, error) {
	searchPath := "."
	pattern := "**/*"
	fsys := os.DirFS(searchPath)

	relPattern := strings.TrimPrefix(pattern, "/")

	var matches []fileInfo

	err := doublestar.GlobWalk(fsys, relPattern, func(path string, d fs.DirEntry) error {
		if d.IsDir() {
			return nil
		}
		if skipHidden(path) {
			return nil
		}

		info, err := d.Info()
		if err != nil {
			return nil // Skip files we can't access
		}

		absPath := path // Restore absolute path
		if !strings.HasPrefix(absPath, searchPath) {
			absPath = filepath.Join(searchPath, absPath)
		}

		matches = append(matches, fileInfo{
			path:    absPath,
			modTime: info.ModTime(),
		})

		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("glob walk error: %w", err)
	}

	sort.Slice(matches, func(i, j int) bool {
		return matches[i].modTime.After(matches[j].modTime)
	})

	results := make([]string, len(matches))
	for i, m := range matches {
		results[i] = m.path
	}

	return results, nil
}

func skipHidden(path string) bool {
	// Check for hidden files (starting with a dot)
	base := filepath.Base(path)
	if base != "." && strings.HasPrefix(base, ".") {
		return true
	}

	// List of commonly ignored directories in development projects
	commonIgnoredDirs := map[string]bool{
		"node_modules":     true,
		"vendor":           true,
		"dist":             true,
		"build":            true,
		"target":           true,
		".git":             true,
		".idea":            true,
		".vscode":          true,
		"__pycache__":      true,
		"bin":              true,
		"obj":              true,
		"out":              true,
		"coverage":         true,
		"tmp":              true,
		"temp":             true,
		"logs":             true,
		"generated":        true,
		"bower_components": true,
		"jspm_packages":    true,
	}

	// Check if any path component is in our ignore list
	parts := strings.SplitSeq(path, string(os.PathSeparator))
	for part := range parts {
		if commonIgnoredDirs[part] {
			return true
		}
	}

	return false
}

func (cg *filesAndFoldersContextGroup) rgCmd() *exec.Cmd {
	rgArgs := []string{
		"--files",
		"-L",
		"--null",
	}
	cmdRg := exec.Command(cg.rgPath, rgArgs...)
	cmdRg.Dir = "."
	return cmdRg
}

func (cg *filesAndFoldersContextGroup) fzfCmd(query string) *exec.Cmd {
	fzfArgs := []string{
		"--filter",
		query,
		"--read0",
		"--print0",
	}
	cmdFzf := exec.Command(cg.fzfPath, fzfArgs...)
	cmdFzf.Dir = "."
	return cmdFzf
}

func (cg *filesAndFoldersContextGroup) getFiles(query string) ([]string, error) {
	if cg.rgPath != "" && cg.fzfPath != "" {

		cmdRg := cg.rgCmd()
		cmdFzf := cg.fzfCmd(query)

		rgPipe, err := cmdRg.StdoutPipe()
		if err != nil {
			return nil, fmt.Errorf("failed to get rg stdout pipe: %w", err)
		}
		defer rgPipe.Close()

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
			file := filepath.Join(".", string(p))
			if !skipHidden(file) {
				matches = append(matches, file)
			}
		}

		sort.SliceStable(matches, func(i, j int) bool {
			if len(matches[i]) != len(matches[j]) {
				return len(matches[i]) < len(matches[j])
			}
			return matches[i] < matches[j]
		})

		return matches, nil
	} else if cg.rgPath != "" {
		// With only rg
		logging.Info("only RG")
		cmdRg := cg.rgCmd()
		var rgOut bytes.Buffer
		var rgErr bytes.Buffer
		cmdRg.Stdout = &rgOut
		cmdRg.Stderr = &rgErr

		if err := cmdRg.Run(); err != nil {
			return nil, fmt.Errorf("rg command failed: %w\nStderr: %s", err, rgErr.String())
		}

		outputBytes := rgOut.Bytes()
		if len(outputBytes) > 0 && outputBytes[len(outputBytes)-1] == 0 {
			outputBytes = outputBytes[:len(outputBytes)-1]
		}

		split := bytes.Split(outputBytes, []byte{0})
		allFiles := make([]string, 0, len(split))

		for _, p := range split {
			if len(p) == 0 {
				continue
			}
			path := filepath.Join(".", string(p))
			if !skipHidden(path) {
				allFiles = append(allFiles, path)
			}
		}

		matches := fuzzy.Find(query, allFiles)

		return matches, nil

	} else if cg.fzfPath != "" {
		// When only fzf is available
		files, err := globWithDoublestar()
		if err != nil {
			return nil, fmt.Errorf("failed to list files: %w", err)
		}

		allFiles := make([]string, 0, len(files))
		for _, file := range files {
			if !skipHidden(file) {
				allFiles = append(allFiles, file)
			}
		}

		cmdFzf := cg.fzfCmd(query)
		var fzfIn bytes.Buffer
		for _, file := range allFiles {
			fzfIn.WriteString(file)
			fzfIn.WriteByte(0)
		}

		cmdFzf.Stdin = &fzfIn
		var fzfOut bytes.Buffer
		var fzfErr bytes.Buffer
		cmdFzf.Stdout = &fzfOut
		cmdFzf.Stderr = &fzfErr

		if err := cmdFzf.Run(); err != nil {
			if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ExitCode() == 1 {
				return []string{}, nil
			}
			return nil, fmt.Errorf("fzf command failed: %w\nStderr: %s", err, fzfErr.String())
		}

		outputBytes := fzfOut.Bytes()
		if len(outputBytes) > 0 && outputBytes[len(outputBytes)-1] == 0 {
			outputBytes = outputBytes[:len(outputBytes)-1]
		}

		split := bytes.Split(outputBytes, []byte{0})
		matches := make([]string, 0, len(split))

		for _, p := range split {
			if len(p) == 0 {
				continue
			}
			matches = append(matches, string(p))
		}

		return matches, nil
	} else {
		// When neither fzf nor rg is available
		allFiles, err := globWithDoublestar()
		if err != nil {
			return nil, fmt.Errorf("failed to glob files: %w", err)
		}

		matches := fuzzy.Find(query, allFiles)

		return matches, nil
	}
}

func (cg *filesAndFoldersContextGroup) GetChildEntries(query string) ([]dialog.CompletionItemI, error) {
	matches, err := cg.getFiles(query)
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
	rgBin, err := exec.LookPath("rg")
	if err != nil {
		logging.Error("ripGrep not found in $PATH", err)
	}
	fzfBin, err := exec.LookPath("fzf")
	if err != nil {
		logging.Error("fzf not found in $PATH", err)
	}

	return &filesAndFoldersContextGroup{
		rgPath:  rgBin,
		fzfPath: fzfBin,
		prefix:  "file",
	}
}
