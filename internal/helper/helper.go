package helper

import (
	"bufio"
	"bytes"
	"encoding/json"
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

// ReadBoundedText opens path exactly once, decides whether it is a text file from
// its first chunk (null byte / invalid UTF-8 => binary), and, if so, returns up
// to maxNumLines lines. When isText is false the caller should skip the content
// (e.g. render "No preview..."). The whole file is never read into memory:
// reading stops as soon as the line budget is reached. Used for single-file
// previews.
func ReadBoundedText(path string, maxNumLines int) (content string, isText bool, err error) {
	file, err := os.Open(path)
	if err != nil {
		return "", false, err
	}
	defer file.Close()

	reader := bufio.NewReaderSize(file, 4096)

	// Detect binary content from the first chunk without consuming it.
	head, err := reader.Peek(4096)
	if err != nil && err != io.EOF && err != bufio.ErrBufferFull {
		return "", false, err
	}
	if bytes.IndexByte(head, 0) != -1 || !utf8.Valid(head) {
		return "", false, nil
	}

	var buffer bytes.Buffer
	numLines := 0
	for numLines <= maxNumLines {
		line, err := reader.ReadString('\n')
		buffer.WriteString(line)
		numLines++
		if err != nil {
			if err == io.EOF {
				break
			}
			return buffer.String(), true, err
		}
	}
	return buffer.String(), true, nil
}

// LoadFilePreview is a helper function that creates a text view for file contents
func LoadFilePreview(path string, searchTerm *string, maxNumLines int) (*tview.TextView, error) {
	// Create text view
	textView := tview.NewTextView().
		SetDynamicColors(true).
		SetRegions(true).
		SetWordWrap(true)
	content, isText, err := ReadBoundedText(path, maxNumLines)
	if err != nil {
		return nil, err
	}
	if !isText {
		textView.SetText("[gray::]No preview...[-::]")
		return textView, nil
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

func UpdateRecentLinesFile(filePath string, path *string, maxLines int) map[string]int {
	data := make(map[string]int)

	file, _ := os.ReadFile(filePath)
	json.Unmarshal(file, &data)

	// Bump the opened file. A freshly-opened file starts high enough to survive
	// ~maxLines subsequent opens before the per-open decay evicts it, so the list
	// can retain up to maxLines entries (a base of 5 previously kept only ~5).
	data[*path] += maxLines + 1

	// Decay every entry by one per open (this is what keeps the ordering "recent"),
	// dropping any that reach zero.
	type entry struct {
		name string
		rank int
	}
	entries := make([]entry, 0, len(data))
	for name, rank := range data {
		rank--
		if rank > 0 {
			entries = append(entries, entry{name: name, rank: rank})
		}
	}

	// Keep the highest-ranked maxLines entries. Map iteration order is random, so
	// sort explicitly rather than truncating an arbitrary subset.
	sort.SliceStable(entries, func(i, j int) bool { return entries[i].rank > entries[j].rank })
	if len(entries) > maxLines {
		entries = entries[:maxLines]
	}

	filteredData := make(map[string]int, len(entries))
	for _, e := range entries {
		filteredData[e.name] = e.rank
	}
	jsonString, _ := json.Marshal(filteredData)
	if err := writeStringsToFile(filePath, []string{string(jsonString)}); err != nil {
		return filteredData
	}
	return filteredData
}

func GetRecentFile(fileIndex int, maxHistoryLen int) (string, int) {
	historyPath := filepath.Join(os.Getenv("HOME"), ".gofileyourselfhistory")
	data := make(map[string]int)
	file, _ := os.ReadFile(historyPath)
	json.Unmarshal(file, &data)
	recentFiles := make([]string, 0, len(data))
	for k := range data {
		recentFiles = append(recentFiles, k)
	}
	sort.SliceStable(recentFiles, func(i, j int) bool {
		return data[recentFiles[i]] < data[recentFiles[j]]
	})

	lenFiles := len(recentFiles)
	if fileIndex >= lenFiles {
		return recentFiles[0], lenFiles
	}
	return recentFiles[lenFiles-fileIndex-1], lenFiles
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

type GitInfo struct {
	Branch           string
	HasUncommited    bool
	HasUntracked     bool
	UntrackedFiles   map[string]bool
	UncommittedFiles map[string]bool
	IsAhead          bool
	IsBehind         bool
}

func GetGitInfo(path string) GitInfo {
	gitStatusCmd := exec.Command("git", "status", "--porcelain", "--branch")
	gitStatusCmd.Dir = path
	var gitStatusOut strings.Builder
	gitStatusCmd.Stdout = &gitStatusOut
	err := gitStatusCmd.Run()
	hasUncommited := false
	branchName := ""
	hasUntracked := false
	isAhead := false
	isBehind := false
	uncommitedFiles := map[string]bool{}
	untrackedFiles := map[string]bool{}
	if err == nil {
		gitStatusOut := gitStatusOut.String()
		lines := strings.Split(gitStatusOut, "\n")
		firstLineParts := strings.Split(strings.TrimPrefix(lines[0], "## "), "...")
		branchName = firstLineParts[0]
		if len(firstLineParts) > 1 {
			if strings.Contains(firstLineParts[1], "ahead") {
				isAhead = true
			}
			if strings.Contains(firstLineParts[1], "behind") {
				isBehind = true
			}
		}
		if len(lines) > 1 {
			gitRootCmd := exec.Command("git", "rev-parse", "--show-toplevel")
			gitRootCmd.Dir = path
			var gitRootOut strings.Builder
			gitRootCmd.Stdout = &gitRootOut
			err = gitRootCmd.Run()
			if err == nil {
				gitRootPath := strings.TrimSuffix(gitRootOut.String(), "\n")
				for _, line := range lines[1:] {
					if line != "" {
						prefix := line[:2]
						linePath := filepath.Join(gitRootPath, line[3:])
						if prefix == " M" || prefix == "M " || prefix == "MM" {
							hasUncommited = true
							uncommitedFiles[linePath] = true
						} else if prefix == "??" {
							hasUntracked = true
							untrackedFiles[linePath] = true
						}
					}
				}
			}
		}
	}
	return GitInfo{
		Branch:           branchName,
		HasUncommited:    hasUncommited,
		HasUntracked:     hasUntracked,
		UntrackedFiles:   untrackedFiles,
		UncommittedFiles: uncommitedFiles,
		IsAhead:          isAhead,
		IsBehind:         isBehind,
	}
}
