package helper

import (
	"bytes"
	"errors"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"sort"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/alecthomas/chroma/formatters"
	"github.com/alecthomas/chroma/lexers"
	"github.com/alecthomas/chroma/styles"
	"github.com/otiai10/copy"
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

func generateDuplicateFileName(path string, duplicationNumber int) string {
	if _, err := os.Stat(path); err == nil {
		suffix := "_" + strconv.Itoa(duplicationNumber)
		if _, err := os.Stat(path + suffix); err == nil {
			duplicationNumber++
			return generateDuplicateFileName(path, duplicationNumber)
		}
		return path + suffix
	}
	return path
}

func CopyFile(src string, dst string) error {
	dst = generateDuplicateFileName(dst, 0)
	err := copy.Copy(src, dst)
	return err
}

// LoadDirectory is a helper function that loads directory contents into a list
func LoadDirectory(path string, showHiddenFiles bool, recursive bool, markedItems []string) (*tview.List, error) {
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
			absPath := filepath.Join(dirPath, file.Name())
			relPath, err := filepath.Rel(path, absPath)
			if err != nil {
				continue
			}

			displayName := relPath
			if slices.Contains(markedItems, absPath) {
				displayName = "m> " + displayName
			}
			if file.IsDir() {
				displayName += "/"
				if recursive {
					// Recursively process subdirectories
					err := processDir(filepath.Join(dirPath, file.Name()))
					if err != nil {
						return err
					}
				}
			} else if info.Mode()&0o111 != 0 {
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

func IsTextFile(path string) bool {
	file, err := os.Open(path)
	if err != nil {
		return false
	}
	defer file.Close()

	// Read first few KB
	buffer := make([]byte, 4096)
	n, err := file.Read(buffer)
	if err != nil && err != io.EOF {
		return false
	}

	// Check for null bytes (common in binary files)
	if bytes.IndexByte(buffer[:n], 0) != -1 {
		return false
	}

	// Verify it's valid UTF-8
	return utf8.Valid(buffer[:n])
}

// LoadFilePreview is a helper function that creates a text view for file contents
func LoadFilePreview(path string) (*tview.TextView, error) {
	// Create text view
	textView := tview.NewTextView().
		SetDynamicColors(true).
		SetRegions(true).
		SetWordWrap(true)

	if IsTextFile(path) {
		content, err := os.ReadFile(path)
		if err != nil {
			return nil, err
		}

		lexer := lexers.Match(path)
		if lexer == nil {
			lexer = lexers.Fallback
		}

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

		var buf bytes.Buffer
		err = formatter.Format(&buf, style, iterator)
		if err != nil {
			return nil, err
		}
		textView.SetText(buf.String())
		return textView, nil
	}
	textView.SetText("[gray::]No preview...[-::]")
	return textView, nil
}

// OpenInNvim is a helper function that opens a file in neovim
func OpenInNvim(path string, selectedFilePath *string, app *tview.Application, maxHistoryLen int) error {
	if selectedFilePath == nil {
		app.Suspend(func() {
			cmd := exec.Command("nvim", path)
			cmd.Stdin = os.Stdin
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
			cmd.Run()
		})
	} else {
		cmd := exec.Command("sh", "-c", "echo \""+path+"\" > "+*selectedFilePath)
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		cmd.Run()
		app.Stop()
	}
	historyPath := filepath.Join(os.Getenv("HOME"), ".gofileyourselfhistory")
	historyCmd := exec.Command("sh", "-c", "[ \"$(tail -n 1 "+historyPath+" 2>/dev/null)\" != "+path+" ] && echo \""+path+"\" >> "+historyPath)
	historyCmd.Stdin = os.Stdin
	historyCmd.Stdout = os.Stdout
	historyCmd.Stderr = os.Stderr
	historyCmd.Run()
	TrimAndGetRecentFiles(historyPath, maxHistoryLen)
	return nil
}

func TrimAndGetRecentFiles(path string, maxHistoryLen int) []string {
	content, err := os.ReadFile(path)
	if err != nil {
		return []string{}
	}
	lines := strings.Split(string(content), "\n")
	lenLines := len(lines)
	if lenLines > maxHistoryLen {
		historyCmd := exec.Command("sh", "-c", "sed '1,"+strconv.Itoa(lenLines-maxHistoryLen)+"d' -i "+path)
		historyCmd.Stdin = os.Stdin
		historyCmd.Stdout = os.Stdout
		historyCmd.Stderr = os.Stderr
		historyCmd.Run()
		return lines[lenLines-maxHistoryLen:]
	}
	return lines
}

func GetRecentFile(fileIndex int, maxHistoryLen int) (string, error) {
	historyPath := filepath.Join(os.Getenv("HOME"), ".gofileyourselfhistory")
	lines := TrimAndGetRecentFiles(historyPath, maxHistoryLen)
	lenLines := len(lines)
	if fileIndex >= lenLines {
		return "", errors.New("file index out of range")
	}
	return lines[lenLines-fileIndex-1], nil
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

func DeleteItem[T comparable](slice []T, element T) []T {
	newSlice := make([]T, 0)
	for _, v := range slice {
		if v != element {
			newSlice = append(newSlice, v)
		}
	}
	return newSlice
}

func CreateDirectory(path string) error {
	return os.MkdirAll(path, 0o755)
}

func RenameFile(oldPath string, newPath string) error {
	return os.Rename(oldPath, newPath)
}

func TouchFile(path string) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	return f.Close()
}

func GetLineWithKey(path string, key string) (string, error) {
	// Check if file exists
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return "", nil // File doesn't exist, return empty string
	}

	// Read file contents
	fileBytes, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}

	// Split into lines and search for key
	lines := strings.Split(string(fileBytes), "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, key) {
			return line, nil
		}
	}

	// Key not found
	return "", nil
}

func AppendOrReplaceLineInFile(path string, content string) error {
	// Create directory if it doesn't exist
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}

	// Ensure content ends with newline
	if !strings.HasSuffix(content, "\n") {
		content += "\n"
	}

	// Check if file exists
	if _, err := os.Stat(path); os.IsNotExist(err) {
		// File doesn't exist, create and write content
		return os.WriteFile(path, []byte(content), 0o644)
	}

	// File exists, read its contents
	fileBytes, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	// Get first letter of content
	if len(content) == 0 {
		return nil // Nothing to add
	}
	firstLetter := content[0]

	// Split file into lines
	lines := strings.Split(string(fileBytes), "\n")
	replaced := false

	// Check each line for matching first letter
	for i, line := range lines {
		if len(line) > 0 && line[0] == firstLetter {
			lines[i] = strings.TrimSuffix(content, "\n")
			replaced = true
			break
		}
	}

	// If no line was replaced, append the content
	if !replaced {
		// Remove empty last line if it exists
		if len(lines) > 0 && lines[len(lines)-1] == "" {
			lines = lines[:len(lines)-1]
		}
		lines = append(lines, strings.TrimSuffix(content, "\n"))
	}

	// Write back to file
	newContent := strings.Join(lines, "\n") + "\n"
	return os.WriteFile(path, []byte(newContent), 0o644)
}
