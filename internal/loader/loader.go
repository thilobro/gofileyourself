package loader

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"sort"

	"github.com/rivo/tview"
	"github.com/thilobro/gofileyourself/internal/helper"
)

// DirectoryLoader handles loading directory contents into a tview.List
type DirectoryLoader struct {
	showHiddenFiles bool
	showContent     bool
	recursive       bool
	markedItems     []string
}

// NewDirectoryLoader returns a new DirectoryLoader instance
func NewDirectoryLoader(showHiddenFiles bool, showContent bool, recursive bool, markedItems []string) *DirectoryLoader {
	return &DirectoryLoader{
		showHiddenFiles: showHiddenFiles,
		showContent:     showContent,
		recursive:       recursive,
		markedItems:     markedItems,
	}
}

// LoadDirectory loads directory contents into a tview.List
func (dl *DirectoryLoader) LoadDirectory(path string) (*tview.List, error) {
	// Validate and normalize path
	// Verify directory exists
	fileInfo, err := os.Stat(path)
	if err != nil {
		return nil, fmt.Errorf("directory stat failed: %w", err)
	}

	if !fileInfo.IsDir() {
		return nil, nil
	}

	absPath, err := filepath.Abs(path)
	if err != nil {
		return nil, fmt.Errorf("failed to get absolute path: %w", err)
	}

	// Initialize list and git info
	list := tview.NewList().ShowSecondaryText(false)
	gitInfo := helper.GetGitInfo(absPath)

	// Add current directory to zoxide if not recursive
	if !dl.recursive {
		if err := exec.Command("zoxide", "add", absPath).Run(); err != nil {
			log.Printf("Failed to add directory to zoxide: %v", err)
		}
	}

	// Process directory contents
	if err := dl.processDirectory(absPath, path, list, &gitInfo); err != nil {
		return nil, fmt.Errorf("directory processing failed: %w", err)
	}

	return list, nil
}

// processDirectory processes a directory and its contents recursively
func (dl *DirectoryLoader) processDirectory(dirPath string, rootPath string, list *tview.List, gitInfo *helper.GitInfo) error {
	entries, err := dl.getFilteredEntries(dirPath)
	if err != nil {
		return fmt.Errorf("failed to get directory entries: %w", err)
	}

	if len(entries) == 0 {
		if !dl.recursive {
			list.AddItem("Directory is empty...", "", 0, nil)
		}
		return nil
	}

	// Sort entries: directories first, then alphabetical
	sort.Slice(entries, func(i, j int) bool {
		iIsDir := entries[i].IsDir()
		jIsDir := entries[j].IsDir()
		if iIsDir == jIsDir {
			return entries[i].Name() < entries[j].Name()
		}
		return iIsDir
	})

	for _, entry := range entries {
		if err := dl.processEntry(entry, dirPath, rootPath, list, gitInfo); err != nil {
			return fmt.Errorf("entry processing failed: %w", err)
		}
	}

	return nil
}

// getFilteredEntries retrieves filtered directory entries
func (dl *DirectoryLoader) getFilteredEntries(dirPath string) ([]os.DirEntry, error) {
	files, err := os.ReadDir(dirPath)
	if err != nil {
		return nil, err
	}

	var filtered []os.DirEntry
	for _, file := range files {
		fileName := file.Name()

		// Skip hidden files if not requested
		if !dl.showHiddenFiles && len(fileName) > 0 && fileName[0] == '.' {
			continue
		}

		// Only include interesting files
		if helper.IsInterestingFile(fileName) {
			filtered = append(filtered, file)
		}
	}

	return filtered, nil
}

// processEntry processes a single directory entry
func (dl *DirectoryLoader) processEntry(entry os.DirEntry, dirPath string, rootPath string, list *tview.List, gitInfo *helper.GitInfo) error {
	info, err := entry.Info()
	if err != nil {
		return fmt.Errorf("failed to get entry info: %w", err)
	}

	// Get relative path
	absPath := filepath.Join(dirPath, entry.Name())
	relPath, err := filepath.Rel(rootPath, absPath)
	if err != nil {
		return fmt.Errorf("failed to get relative path: %w", err)
	}

	displayName := relPath

	// Add marker if item is marked
	if slices.Contains(dl.markedItems, absPath) {
		displayName = "m> " + displayName
	}

	// Process directories
	if entry.IsDir() {
		displayName += "/"
		if dl.recursive {
			if err := dl.processDirectory(absPath, rootPath, list, gitInfo); err != nil {
				return fmt.Errorf("recursive processing failed: %w", err)
			}
		}
	} else {
		// Add executable marker
		if info.Mode()&0o111 != 0 {
			displayName = "x " + displayName
		}

		// Add git status indicators
		if !dl.recursive {
			displayName += dl.getGitStatusString(gitInfo, absPath)
		}

		// Add file content preview if configured
		if dl.showContent && helper.IsInterestingFile(absPath) && helper.IsTextFile(absPath) {
			content, err := os.ReadFile(absPath)
			if err != nil {
				return fmt.Errorf("failed to read file content: %w", err)
			}
			displayName += " >>> " + string(content)
		}
	}

	list.AddItem(displayName, relPath, 0, nil)
	return nil
}

// getGitStatusString generates git status indicator string
func (dl *DirectoryLoader) getGitStatusString(gitInfo *helper.GitInfo, path string) string {
	var status string
	if _, exists := gitInfo.UncommittedFiles[path]; exists {
		status += "[red]*[white]"
	}
	if _, exists := gitInfo.UntrackedFiles[path]; exists {
		status += "[red]?[white]"
	}
	return status
}
