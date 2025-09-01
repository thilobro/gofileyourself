package helper

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
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
func LoadDirectory(path string, showHiddenFiles bool, showContent bool, recursive bool, markedItems []string) (*tview.List, error) {
	absPath, _ := filepath.Abs(path)
	fileInfo, err := os.Stat(path)
	if err != nil {
		return nil, err
	}

	if !fileInfo.IsDir() {
		return nil, nil
	}
	if !recursive {
		exec.Command("zoxide", "add", absPath).Run()
	}
	list := tview.NewList().ShowSecondaryText(false)

	var processDir func(dirPath string) error
	processDir = func(dirPath string) error {
		files, err := os.ReadDir(dirPath)
		if err != nil {
			return err
		}

		fileSlice := make([]os.DirEntry, 0)
		for _, file := range files {
			fileName := file.Name()
			if !showHiddenFiles && len(fileName) > 0 && fileName[0] == '.' {
				continue
			}
			if IsInterestingFile(fileName) {
				fileSlice = append(fileSlice, file)
			}
		}
		if len(fileSlice) == 0 {
			if !recursive {
				list.AddItem("Directory is empty...", "", 0, nil)
			}
			return nil
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
				displayName += "x "
			}

			if showContent && IsInterestingFile(absPath) && IsTextFile(absPath) {
				content, _ := os.ReadFile(absPath)
				displayName += " >>> " + string(content)

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
func LoadFilePreview(path string, searchTerm *string) (*tview.TextView, error) {
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

		style := styles.Get("gofileyourself")
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
		formattedContent := buf.String()
		if searchTerm != nil && *searchTerm == "" {
			formattedContent = strings.ReplaceAll(formattedContent, "[\"hlrg\"]", "")
			formattedContent = strings.ReplaceAll(formattedContent, "[\"\"]", "")
			searchTerm = nil
		}
		if searchTerm != nil {
			terms := strings.Split(*searchTerm, " ")
			for _, term := range terms {
				pattern := regexp.MustCompile(`(?i)(\[#.*?\])|(` + regexp.QuoteMeta(term) + `)`)
				formattedContent = pattern.ReplaceAllStringFunc(formattedContent, func(match string) string {
					// The `match` variable is the substring that the regex found.
					// It will either be "[#...]" or your search term "en".

					// We check if the match is the part we want to IGNORE.
					// A simple check is to see if it starts with "[#".
					if strings.HasPrefix(match, "[#") {
						// If it's the excluded block, return it completely unchanged.
						return match
					} else {
						// Otherwise, it's the term we want to highlight.
						// Surround it with your desired markers.
						return `["hlrg"]` + match + `[""]`
					}
				})
			}
		}
		textView.SetText(formattedContent)
		if searchTerm != nil {
			firstLine, _ := findLineNumberScanner(formattedContent, "hlrg")
			textView.SetRegions(true)
			textView.Highlight("hlrg")
			_, _, _, height := textView.GetInnerRect()
			if firstLine >= height/2 {
				textView.ScrollToHighlight()
			}
		}
		return textView, nil
	}
	textView.SetText("[gray::]No preview...[-::]")
	return textView, nil
}

func findLineNumberScanner(content, substring string) (int, error) {
	scanner := bufio.NewScanner(strings.NewReader(content))

	lineNumber := 1
	for scanner.Scan() { // .Scan() advances to the next line
		if strings.Contains(scanner.Text(), substring) {
			return lineNumber, nil
		}
		lineNumber++
	}

	// Check for errors during scanning (e.g., if the token is too long)
	if err := scanner.Err(); err != nil {
		return 0, err
	}

	// If we get here, the substring was not found
	return 0, fmt.Errorf("substring %q not found", substring)
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
	UpdateRecentLinesFile(historyPath, &path, maxHistoryLen)
	return nil
}

func writeStringsToFile(filename string, strings []string) error {
	// Open the file in write mode
	os.Remove(filename)
	file, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer file.Close()

	// Handle empty slice case
	if len(strings) == 0 {
		return nil
	}

	// Write all but the last string with newlines
	for _, str := range strings[:len(strings)-1] {
		_, err := fmt.Fprintf(file, "%s\n", str)
		if err != nil {
			return fmt.Errorf("failed to write to file: %w", err)
		}
	}

	// Write the last string without a newline
	_, err = fmt.Fprint(file, strings[len(strings)-1])
	if err != nil {
		return fmt.Errorf("failed to write final string: %w", err)
	}

	return nil
}

func UpdateRecentLinesFile(filePath string, path *string, maxLines int) []string {
	// Read existing content
	content, err := os.ReadFile(filePath)
	if err != nil {
		os.Create(filePath)
	}

	// Create map for unique values
	linesSet := map[string]int{}

	// Split content and add to set
	lines := strings.Split(string(content), "\n")
	offset := 0
	if path != nil && *path != "" {
		offset = 1
	}
	for idx, line := range lines {
		if line != "" { // Skip empty lines
			linesSet[line] = idx + offset
		}
	}

	// Add new path if provided
	if path != nil && *path != "" {
		linesSet[*path] = 0
	}

	// Convert back to slice and limit length
	var result []string
	for line := range linesSet {
		if len(result) >= maxLines {
			break
		}
		result = append(result, line)
	}
	// Sort by original order
	sort.SliceStable(result, func(i, j int) bool {
		return linesSet[result[i]] > linesSet[result[j]]
	})

	// Write back to file
	if err := writeStringsToFile(filePath, result); err != nil {
		return result // Return current state despite write error
	}

	return result
}

func GetRecentFile(fileIndex int, maxHistoryLen int) (string, error) {
	historyPath := filepath.Join(os.Getenv("HOME"), ".gofileyourselfhistory")
	lines := UpdateRecentLinesFile(historyPath, nil, maxHistoryLen)
	lenLines := len(lines)
	if fileIndex >= lenLines {
		return "", errors.New("file index out of range")
	}
	return lines[lenLines-fileIndex-1], nil
}

func IsFileNotFound(path string) bool {
	if _, err := os.Stat(path); errors.Is(err, os.ErrNotExist) {
		return true
	}
	return false
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
	path = generateDuplicateFileName(path, 0)
	return os.MkdirAll(path, 0o755)
}

func RenameFile(oldPath string, newPath string) error {
	newPath = generateDuplicateFileName(newPath, 0)
	return os.Rename(oldPath, newPath)
}

func TouchFile(path string) error {
	path = generateDuplicateFileName(path, 0)
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

func GetAbsFilePath(filePath string, dirPath string) string {
	if filepath.IsAbs(filePath) {
		return filePath
	} else {
		return filepath.Join(dirPath, filePath)
	}
}

func AppendStringToUniqueList(recentList []string, newString string) []string {
	keep := func(element string) bool {
		return element != newString && element != ""
	}
	n := 0
	for _, x := range recentList {
		if keep(x) {
			recentList[n] = x
			n++
		}
	}
	recentList = recentList[:n]
	recentList = append(recentList, newString)
	return recentList
}

func CycleRecentList(recentList []string, index int, backwards bool) (int, string) {
	if backwards {
		index++
	} else {
		index--
	}
	if index < 0 {
		index = -1
	} else if index >= len(recentList) {
		index = len(recentList) - 1
	}
	if index == -1 {
		return index, ""
	}
	return index, recentList[index]
}

func CopyListContent(original *tview.List, copy *tview.List) {
	currentItem := copy.GetCurrentItem()
	copy.Clear()
	for i := 0; i < original.GetItemCount(); i++ {
		primaryText, secondaryText := original.GetItemText(i)
		copy.AddItem(primaryText, secondaryText, 0, nil)
	}
	copy.SetCurrentItem(currentItem)
}

func CopyListView(original *tview.List) *tview.List {
	// Create a new ListView instance
	newList := tview.NewList().ShowSecondaryText(false)

	// Copy items from original to new list
	//
	for j := 0; j < original.GetItemCount(); j++ {
		primaryText, secondaryText := original.GetItemText(j)
		newList.AddItem(primaryText, secondaryText, 0, nil)
	}

	// Copy current selection state
	newList.SetCurrentItem(original.GetCurrentItem())

	return newList
}

func ShortenPathsIfNecessary(pathList *tview.List, maxPathLen int) {
	startHighlightMarker := "[red::b]"
	for i := 0; i < pathList.GetItemCount(); i++ {
		displayName, secondaryText := pathList.GetItemText(i)
		displayLen := len(displayName)
		if displayLen > maxPathLen {
			parts := strings.Split(displayName, "/")
			shortDisplayName := parts[0]
			if len(parts) >= 2 {
				for _, part := range parts[1 : len(parts)-1] {
					if !strings.Contains(part, startHighlightMarker) {
						part = "..."
					}
					shortDisplayName = shortDisplayName + "/" + part
				}
				shortDisplayName = shortDisplayName + "/" + parts[len(parts)-1]
			}
			pathList.SetItemText(i, shortDisplayName, secondaryText)
		}
	}
}

func IsInterestingFile(path string) bool {
	uninterestingFileTypes := []string{".pyc", ".mod", "typed", ".lock"}
	extension := filepath.Ext(path)
	return !slices.Contains(uninterestingFileTypes, extension)
}
