package helper

import (
	"bytes"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"

	"github.com/alecthomas/chroma/formatters"
	"github.com/alecthomas/chroma/lexers"
	"github.com/alecthomas/chroma/styles"
	"github.com/rivo/tview"
)

// FindExactItem is a helper function that searches for an item in a list
func FindExactItem(list *tview.List, searchTerm string) int {
	matchingIndeces := list.FindItems(searchTerm, "", false, true)
	if len(matchingIndeces) == 1 {
		return matchingIndeces[0]
	}
	for _, index := range matchingIndeces {
		if _, secondaryText := list.GetItemText(index); secondaryText == searchTerm {
			return index
		}
	}
	return 0
}

// LoadDirectory is a helper function that loads directory contents into a list
func LoadDirectory(path string, showHiddenFiles bool, recursive bool) (*tview.List, error) {
	fileInfo, err := os.Stat(path)
	if err != nil {
		return nil, err
	}

	if !fileInfo.IsDir() {
		return nil, nil
	}

	list := tview.NewList().ShowSecondaryText(false)

	var processDir func(dirPath string) error
	processDir = func(dirPath string) error {
		files, err := os.ReadDir(dirPath)
		if err != nil {
			return err
		}
		if len(files) == 0 {
			return nil
		}

		fileSlice := make([]os.DirEntry, 0)
		for _, file := range files {
			fileName := file.Name()
			if !showHiddenFiles && len(fileName) > 0 && fileName[0] == '.' {
				continue
			}
			fileSlice = append(fileSlice, file)
		}

		// Sort: directories first, then alphabetically
		sort.Slice(fileSlice, func(i, j int) bool {
			iIsDir := fileSlice[i].IsDir()
			jIsDir := fileSlice[j].IsDir()
			if iIsDir == jIsDir {
				return fileSlice[i].Name() < fileSlice[j].Name()
			}
			return iIsDir
		})

		for _, file := range fileSlice {
			info, err := file.Info()
			if err != nil {
				continue
			}

			// Get relative path from the root directory
			relPath, err := filepath.Rel(path, filepath.Join(dirPath, file.Name()))
			if err != nil {
				continue
			}

			displayName := relPath
			if file.IsDir() {
				displayName += "/"
				if recursive {
					// Recursively process subdirectories
					err := processDir(filepath.Join(dirPath, file.Name()))
					if err != nil {
						return err
					}
				}
			} else if info.Mode()&0111 != 0 {
				displayName += "*"
			}

			list.AddItem(displayName, relPath, 0, nil)
		}
		return nil
	}

	err = processDir(path)
	if err != nil {
		return nil, err
	}

	return list, nil
}

// LoadFilePreview is a helper function that creates a text view for file contents
func LoadFilePreview(path string) (*tview.TextView, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	// Create text view
	textView := tview.NewTextView().
		SetDynamicColors(true).
		SetRegions(true).
		SetWordWrap(true)

	// Detect language based on file extension
	lexer := lexers.Match(path)
	if lexer == nil {
		lexer = lexers.Fallback
	}

	// Use gruvbox style
	style := styles.Get("gruvbox")
	if style == nil {
		style = styles.Fallback
	}

	formatter := formatters.Get("tview")
	if formatter == nil {
		formatter = formatters.Fallback
	}

	iterator, err := lexer.Tokenise(nil, string(content))
	if err != nil {
		return nil, err
	}

	// Create buffer to store formatted output
	var buf bytes.Buffer
	err = formatter.Format(&buf, style, iterator)
	if err != nil {
		return nil, err
	}

	textView.SetText(buf.String())
	return textView, nil
}

// OpenInNvim is a helper function that opens a file in neovim
func OpenInNvim(path string, app *tview.Application) error {
	app.Suspend(func() {
		cmd := exec.Command("nvim", path)
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		cmd.Run()
	})
	return nil
}

func IsDirectoryEmpty(path string) (bool, error) {
	f, err := os.Open(path)
	if err != nil {
		return false, err
	}
	defer f.Close()

	// Read just one entry. If error is EOF, directory is empty
	_, err = f.Readdirnames(1)
	if err == io.EOF {
		return true, nil
	}
	return false, err
}

func Contains(needle int, haystack []int) bool {
	for _, i := range haystack {
		if needle == i {
			return true
		}
	}
	return false
}
