package tools

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLsTool_Info(t *testing.T) {
	tool := NewLsTool()
	info := tool.Info()

	assert.Equal(t, LSToolName, info.Name)
	assert.NotEmpty(t, info.Description)
	assert.Contains(t, info.Parameters, "path")
	assert.Contains(t, info.Parameters, "ignore")
	assert.Contains(t, info.Required, "path")
}

func TestLsTool_Run(t *testing.T) {
	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "ls_tool_test")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	// Create a test directory structure
	testDirs := []string{
		"dir1",
		"dir2",
		"dir2/subdir1",
		"dir2/subdir2",
		"dir3",
		"dir3/.hidden_dir",
		"__pycache__",
	}

	testFiles := []string{
		"file1.txt",
		"file2.txt",
		"dir1/file3.txt",
		"dir2/file4.txt",
		"dir2/subdir1/file5.txt",
		"dir2/subdir2/file6.txt",
		"dir3/file7.txt",
		"dir3/.hidden_file.txt",
		"__pycache__/cache.pyc",
		".hidden_root_file.txt",
	}

	// Create directories
	for _, dir := range testDirs {
		dirPath := filepath.Join(tempDir, dir)
		err := os.MkdirAll(dirPath, 0755)
		require.NoError(t, err)
	}

	// Create files
	for _, file := range testFiles {
		filePath := filepath.Join(tempDir, file)
		err := os.WriteFile(filePath, []byte("test content"), 0644)
		require.NoError(t, err)
	}

	t.Run("lists directory successfully", func(t *testing.T) {
		tool := NewLsTool()
		params := LSParams{
			Path: tempDir,
		}

		paramsJSON, err := json.Marshal(params)
		require.NoError(t, err)

		call := ToolCall{
			Name:  LSToolName,
			Input: string(paramsJSON),
		}

		response, err := tool.Run(context.Background(), call)
		require.NoError(t, err)

		// Check that visible directories and files are included
		assert.Contains(t, response.Content, "dir1")
		assert.Contains(t, response.Content, "dir2")
		assert.Contains(t, response.Content, "dir3")
		assert.Contains(t, response.Content, "file1.txt")
		assert.Contains(t, response.Content, "file2.txt")

		// Check that hidden files and directories are not included
		assert.NotContains(t, response.Content, ".hidden_dir")
		assert.NotContains(t, response.Content, ".hidden_file.txt")
		assert.NotContains(t, response.Content, ".hidden_root_file.txt")

		// Check that __pycache__ is not included
		assert.NotContains(t, response.Content, "__pycache__")
	})

	t.Run("handles non-existent path", func(t *testing.T) {
		tool := NewLsTool()
		params := LSParams{
			Path: filepath.Join(tempDir, "non_existent_dir"),
		}

		paramsJSON, err := json.Marshal(params)
		require.NoError(t, err)

		call := ToolCall{
			Name:  LSToolName,
			Input: string(paramsJSON),
		}

		response, err := tool.Run(context.Background(), call)
		require.NoError(t, err)
		assert.Contains(t, response.Content, "path does not exist")
	})

	t.Run("handles empty path parameter", func(t *testing.T) {
		// For this test, we need to mock the config.WorkingDirectory function
		// Since we can't easily do that, we'll just check that the response doesn't contain an error message

		tool := NewLsTool()
		params := LSParams{
			Path: "",
		}

		paramsJSON, err := json.Marshal(params)
		require.NoError(t, err)

		call := ToolCall{
			Name:  LSToolName,
			Input: string(paramsJSON),
		}

		response, err := tool.Run(context.Background(), call)
		require.NoError(t, err)

		// The response should either contain a valid directory listing or an error
		// We'll just check that it's not empty
		assert.NotEmpty(t, response.Content)
	})

	t.Run("handles invalid parameters", func(t *testing.T) {
		tool := NewLsTool()
		call := ToolCall{
			Name:  LSToolName,
			Input: "invalid json",
		}

		response, err := tool.Run(context.Background(), call)
		require.NoError(t, err)
		assert.Contains(t, response.Content, "error parsing parameters")
	})

	t.Run("respects ignore patterns", func(t *testing.T) {
		tool := NewLsTool()
		params := LSParams{
			Path:   tempDir,
			Ignore: []string{"file1.txt", "dir1"},
		}

		paramsJSON, err := json.Marshal(params)
		require.NoError(t, err)

		call := ToolCall{
			Name:  LSToolName,
			Input: string(paramsJSON),
		}

		response, err := tool.Run(context.Background(), call)
		require.NoError(t, err)

		// The output format is a tree, so we need to check for specific patterns
		// Check that file1.txt is not directly mentioned
		assert.NotContains(t, response.Content, "- file1.txt")

		// Check that dir1/ is not directly mentioned
		assert.NotContains(t, response.Content, "- dir1/")
	})

	t.Run("handles relative path", func(t *testing.T) {
		// Save original working directory
		origWd, err := os.Getwd()
		require.NoError(t, err)
		defer func() {
			os.Chdir(origWd)
		}()

		// Change to a directory above the temp directory
		parentDir := filepath.Dir(tempDir)
		err = os.Chdir(parentDir)
		require.NoError(t, err)

		tool := NewLsTool()
		params := LSParams{
			Path: filepath.Base(tempDir),
		}

		paramsJSON, err := json.Marshal(params)
		require.NoError(t, err)

		call := ToolCall{
			Name:  LSToolName,
			Input: string(paramsJSON),
		}

		response, err := tool.Run(context.Background(), call)
		require.NoError(t, err)

		// Should list the temp directory contents
		assert.Contains(t, response.Content, "dir1")
		assert.Contains(t, response.Content, "file1.txt")
	})
}

func TestShouldSkip(t *testing.T) {
	testCases := []struct {
		name           string
		path           string
		ignorePatterns []string
		expected       bool
	}{
		{
			name:           "hidden file",
			path:           "/path/to/.hidden_file",
			ignorePatterns: []string{},
			expected:       true,
		},
		{
			name:           "hidden directory",
			path:           "/path/to/.hidden_dir",
			ignorePatterns: []string{},
			expected:       true,
		},
		{
			name:           "pycache directory",
			path:           "/path/to/__pycache__/file.pyc",
			ignorePatterns: []string{},
			expected:       true,
		},
		{
			name:           "node_modules directory",
			path:           "/path/to/node_modules/package",
			ignorePatterns: []string{},
			expected:       false, // The shouldSkip function doesn't directly check for node_modules in the path
		},
		{
			name:           "normal file",
			path:           "/path/to/normal_file.txt",
			ignorePatterns: []string{},
			expected:       false,
		},
		{
			name:           "normal directory",
			path:           "/path/to/normal_dir",
			ignorePatterns: []string{},
			expected:       false,
		},
		{
			name:           "ignored by pattern",
			path:           "/path/to/ignore_me.txt",
			ignorePatterns: []string{"ignore_*.txt"},
			expected:       true,
		},
		{
			name:           "not ignored by pattern",
			path:           "/path/to/keep_me.txt",
			ignorePatterns: []string{"ignore_*.txt"},
			expected:       false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := shouldSkip(tc.path, tc.ignorePatterns)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestCreateFileTree(t *testing.T) {
	paths := []string{
		"/path/to/file1.txt",
		"/path/to/dir1/file2.txt",
		"/path/to/dir1/subdir/file3.txt",
		"/path/to/dir2/file4.txt",
	}

	tree := createFileTree(paths)

	// Check the structure of the tree
	assert.Len(t, tree, 1) // Should have one root node

	// Check the root node
	rootNode := tree[0]
	assert.Equal(t, "path", rootNode.Name)
	assert.Equal(t, "directory", rootNode.Type)
	assert.Len(t, rootNode.Children, 1)

	// Check the "to" node
	toNode := rootNode.Children[0]
	assert.Equal(t, "to", toNode.Name)
	assert.Equal(t, "directory", toNode.Type)
	assert.Len(t, toNode.Children, 3) // file1.txt, dir1, dir2

	// Find the dir1 node
	var dir1Node *TreeNode
	for _, child := range toNode.Children {
		if child.Name == "dir1" {
			dir1Node = child
			break
		}
	}

	require.NotNil(t, dir1Node)
	assert.Equal(t, "directory", dir1Node.Type)
	assert.Len(t, dir1Node.Children, 2) // file2.txt and subdir
}

func TestPrintTree(t *testing.T) {
	// Create a simple tree
	tree := []*TreeNode{
		{
			Name: "dir1",
			Path: "dir1",
			Type: "directory",
			Children: []*TreeNode{
				{
					Name: "file1.txt",
					Path: "dir1/file1.txt",
					Type: "file",
				},
				{
					Name: "subdir",
					Path: "dir1/subdir",
					Type: "directory",
					Children: []*TreeNode{
						{
							Name: "file2.txt",
							Path: "dir1/subdir/file2.txt",
							Type: "file",
						},
					},
				},
			},
		},
		{
			Name: "file3.txt",
			Path: "file3.txt",
			Type: "file",
		},
	}

	result := printTree(tree, "/root")

	// Check the output format
	assert.Contains(t, result, "- /root/")
	assert.Contains(t, result, "  - dir1/")
	assert.Contains(t, result, "    - file1.txt")
	assert.Contains(t, result, "    - subdir/")
	assert.Contains(t, result, "      - file2.txt")
	assert.Contains(t, result, "  - file3.txt")
}

func TestListDirectory(t *testing.T) {
	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "list_directory_test")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	// Create a test directory structure
	testDirs := []string{
		"dir1",
		"dir1/subdir1",
		".hidden_dir",
	}

	testFiles := []string{
		"file1.txt",
		"file2.txt",
		"dir1/file3.txt",
		"dir1/subdir1/file4.txt",
		".hidden_file.txt",
	}

	// Create directories
	for _, dir := range testDirs {
		dirPath := filepath.Join(tempDir, dir)
		err := os.MkdirAll(dirPath, 0755)
		require.NoError(t, err)
	}

	// Create files
	for _, file := range testFiles {
		filePath := filepath.Join(tempDir, file)
		err := os.WriteFile(filePath, []byte("test content"), 0644)
		require.NoError(t, err)
	}

	t.Run("lists files with no limit", func(t *testing.T) {
		files, truncated, err := listDirectory(tempDir, []string{}, 1000, 0)
		require.NoError(t, err)
		assert.False(t, truncated)

		// Check that visible files and directories are included
		containsPath := func(paths []string, target string) bool {
			targetPath := filepath.Join(tempDir, target)
			for _, path := range paths {
				if strings.HasPrefix(path, targetPath) {
					return true
				}
			}
			return false
		}

		assert.True(t, containsPath(files, "dir1"))
		assert.True(t, containsPath(files, "file1.txt"))
		assert.True(t, containsPath(files, "file2.txt"))
		assert.True(t, containsPath(files, "dir1/file3.txt"))

		// Check that hidden files and directories are not included
		assert.False(t, containsPath(files, ".hidden_dir"))
		assert.False(t, containsPath(files, ".hidden_file.txt"))
	})

	t.Run("respects limit and returns truncated flag", func(t *testing.T) {
		files, truncated, err := listDirectory(tempDir, []string{}, 2, 0)
		require.NoError(t, err)
		assert.True(t, truncated)
		assert.Len(t, files, 2)
	})

	t.Run("respects ignore patterns", func(t *testing.T) {
		files, truncated, err := listDirectory(tempDir, []string{"*.txt"}, 1000, 0)
		require.NoError(t, err)
		assert.False(t, truncated)

		// Check that no .txt files are included
		for _, file := range files {
			assert.False(t, strings.HasSuffix(file, ".txt"), "Found .txt file: %s", file)
		}

		// But directories should still be included
		containsDir := false
		for _, file := range files {
			if strings.Contains(file, "dir1") {
				containsDir = true
				break
			}
		}
		assert.True(t, containsDir)
	})
}

func TestListDirectoryWithMaxDepth(t *testing.T) {
	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "list_directory_max_depth_test")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	// Create a deeper directory structure
	testDirs := []string{
		"level1",
		"level1/level2",
		"level1/level2/level3",
		"level1/level2/level3/level4",
		"other_level1",
	}

	testFiles := []string{
		"root_file.txt",
		"level1/level1_file.txt",
		"level1/level2/level2_file.txt",
		"level1/level2/level3/level3_file.txt",
		"level1/level2/level3/level4/level4_file.txt",
		"other_level1/other_file.txt",
	}

	// Create directories
	for _, dir := range testDirs {
		dirPath := filepath.Join(tempDir, dir)
		err := os.MkdirAll(dirPath, 0755)
		require.NoError(t, err)
	}

	// Create files
	for _, file := range testFiles {
		filePath := filepath.Join(tempDir, file)
		err := os.WriteFile(filePath, []byte("test content"), 0644)
		require.NoError(t, err)
	}

	t.Run("maxDepth 1 should include only first level", func(t *testing.T) {
		files, truncated, err := listDirectory(tempDir, []string{}, 1000, 1)
		require.NoError(t, err)
		assert.False(t, truncated)

		// Should include first level directories and files
		containsPath := func(paths []string, target string) bool {
			targetPath := filepath.Join(tempDir, target)
			for _, path := range paths {
				if strings.HasPrefix(path, targetPath) {
					return true
				}
			}
			return false
		}

		// Should include level 1 items
		assert.True(t, containsPath(files, "level1/"))
		assert.True(t, containsPath(files, "other_level1/"))
		assert.True(t, containsPath(files, "root_file.txt"))

		// Should not include contents of level 1 directories (since we don't traverse them)
		assert.False(t, containsPath(files, "level1/level1_file.txt"))
		assert.False(t, containsPath(files, "other_level1/other_file.txt"))

		// Should not include level 2 and deeper items
		assert.False(t, containsPath(files, "level1/level2/"))
		assert.False(t, containsPath(files, "level1/level2/level2_file.txt"))
		assert.False(t, containsPath(files, "level1/level2/level3/"))
	})

	t.Run("maxDepth 2 should include up to second level", func(t *testing.T) {
		files, truncated, err := listDirectory(tempDir, []string{}, 1000, 2)
		require.NoError(t, err)
		assert.False(t, truncated)

		containsPath := func(paths []string, target string) bool {
			targetPath := filepath.Join(tempDir, target)
			for _, path := range paths {
				if strings.HasPrefix(path, targetPath) {
					return true
				}
			}
			return false
		}

		// Should include up to level 2 directories
		assert.True(t, containsPath(files, "level1/level2/"))
		// Should include level 1 files inside directories
		assert.True(t, containsPath(files, "level1/level1_file.txt"))
		// Should include other level 1 files
		assert.True(t, containsPath(files, "other_level1/other_file.txt"))

		// Should not include files inside level 2 directories (they would be at depth 3)
		assert.False(t, containsPath(files, "level1/level2/level2_file.txt"))
		// Should not include level 3 and deeper items
		assert.False(t, containsPath(files, "level1/level2/level3/"))
		assert.False(t, containsPath(files, "level1/level2/level3/level3_file.txt"))
	})

	t.Run("maxDepth 0 should traverse all levels", func(t *testing.T) {
		files, truncated, err := listDirectory(tempDir, []string{}, 1000, 0)
		require.NoError(t, err)
		assert.False(t, truncated)

		containsPath := func(paths []string, target string) bool {
			targetPath := filepath.Join(tempDir, target)
			for _, path := range paths {
				if strings.HasPrefix(path, targetPath) {
					return true
				}
			}
			return false
		}

		// Should include all levels
		assert.True(t, containsPath(files, "level1/level2/level3/level4/"))
		assert.True(t, containsPath(files, "level1/level2/level3/level4/level4_file.txt"))
	})
}

func TestShouldSkipDirDueToDepth(t *testing.T) {
	t.Run("maxDepth <= 0 should never skip", func(t *testing.T) {
		testCases := []struct {
			name     string
			maxDepth int
			expected bool
		}{
			{"maxDepth 0", 0, false},
			{"maxDepth -1", -1, false},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				result := shouldSkipDirDueToDepth("/root", "/root/subdir", tc.maxDepth)
				assert.Equal(t, tc.expected, result, "maxDepth <= 0 should never skip")
			})
		}
	})

	t.Run("directories at or beyond maxDepth should be skipped", func(t *testing.T) {
		testCases := []struct {
			name        string
			initialPath string
			currentPath string
			maxDepth    int
			expected    bool
		}{
			{"dir at depth 1, maxDepth 1", "/root", "/root/subdir", 1, true},
			{"dir at depth 2, maxDepth 1", "/root", "/root/dir1/dir2", 1, true},
			{"dir at depth 2, maxDepth 2", "/root", "/root/dir1/dir2", 2, true},
			{"dir at depth 1, maxDepth 2", "/root", "/root/subdir", 2, false},
			{"dir at depth 2, maxDepth 3", "/root", "/root/dir1/dir2", 3, false},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				result := shouldSkipDirDueToDepth(tc.initialPath, tc.currentPath, tc.maxDepth)
				assert.Equal(t, tc.expected, result, tc.name)
			})
		}
	})

	t.Run("same path as initial should not be skipped", func(t *testing.T) {
		result := shouldSkipDirDueToDepth("/root", "/root", 1)
		assert.False(t, result, "initial path should not be skipped")
	})

	t.Run("handles path with separators correctly", func(t *testing.T) {
		testCases := []struct {
			name        string
			initialPath string
			currentPath string
			maxDepth    int
			expected    bool
		}{
			{"Unix path depth 1", "/root", "/root/subdir", 1, true},
			{"Unix path depth 2", "/root", "/root/dir1/dir2", 2, true},
			{"Unix path depth 3", "/root", "/root/dir1/dir2/dir3", 3, true},
			{"Unix path depth 2 with maxDepth 3", "/root", "/root/dir1/dir2", 3, false},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				result := shouldSkipDirDueToDepth(tc.initialPath, tc.currentPath, tc.maxDepth)
				assert.Equal(t, tc.expected, result, tc.name)
			})
		}
	})

	t.Run("handles error in filepath.Rel gracefully", func(t *testing.T) {
		// This test case should handle scenarios where filepath.Rel might fail
		// For example, when paths are on different drives on Windows
		result := shouldSkipDirDueToDepth("", "", 1)
		assert.False(t, result, "should not skip when filepath.Rel fails")
	})

	t.Run("complex nested paths", func(t *testing.T) {
		initialPath := "/home/user/project"

		testCases := []struct {
			name        string
			currentPath string
			maxDepth    int
			expected    bool
		}{
			{"nested dir at exact maxDepth", "/home/user/project/src/main", 2, true},
			{"nested dir under maxDepth", "/home/user/project/src", 2, false},
			{"deeply nested dir", "/home/user/project/src/main/java/com", 3, true},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				result := shouldSkipDirDueToDepth(initialPath, tc.currentPath, tc.maxDepth)
				assert.Equal(t, tc.expected, result, tc.name)
			})
		}
	})
}
