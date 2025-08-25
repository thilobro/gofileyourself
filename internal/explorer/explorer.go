package explorer

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/thilobro/gofileyourself/internal/helper"
	"github.com/thilobro/gofileyourself/internal/widget"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

const (
	constMaxScrollAmount = 20
)

// Explorer represents the state and behavior of the file explorer
type Explorer struct {
	context                 *widget.Context
	currentList             *tview.List
	parentList              tview.Primitive
	selectedList            tview.Primitive
	rootFlex                *tview.Flex
	listFlex                *tview.Flex
	directoryToIndexMap     map[string]int
	footer                  *tview.InputField
	isFooterActive          bool
	header                  *tview.TextView
	searchInput             string
	currentSearchTerm       string
	currentSearchIndeces    []int
	currentFocusedWidget    tview.Primitive
	keyBuffer               string
	yankedFile              string
	markedFiles             []string
	yankedMarkedFiles       []string
	cycleRecentlyVisitedIdx int
	cycleRecentCommandsIdx  int
	cycleRecentSearchesIdx  int
}

func (explorer *Explorer) Root() tview.Primitive {
	return explorer.rootFlex
}

func NewExplorer(context *widget.Context) (*Explorer, error) {
	explorer := &Explorer{
		context:                 context,
		currentList:             tview.NewList(),
		parentList:              tview.NewList(),
		selectedList:            tview.NewList(),
		directoryToIndexMap:     make(map[string]int),
		listFlex:                tview.NewFlex(),
		rootFlex:                tview.NewFlex(),
		footer:                  tview.NewInputField(),
		isFooterActive:          false,
		header:                  tview.NewTextView(),
		searchInput:             "",
		currentSearchTerm:       "",
		keyBuffer:               "",
		yankedFile:              "",
		markedFiles:             []string{},
		yankedMarkedFiles:       []string{},
		cycleRecentlyVisitedIdx: 0,
		cycleRecentSearchesIdx:  -1,
		cycleRecentCommandsIdx:  -1,
	}

	if err := explorer.initialize(); err != nil {
		return nil, err
	}

	return explorer, nil
}

func (explorer *Explorer) initialize() error {
	explorer.SetupKeyBindings()
	if explorer.context.SelectedFilePath != nil {
		explorer.context.CurrentPath = filepath.Dir(*explorer.context.SelectedFilePath)
		explorer.setCurrentDirectory(explorer.context.CurrentPath)
		selectedFileIndex := helper.FindExactItem(explorer.currentList, filepath.Base(*explorer.context.SelectedFilePath))
		explorer.setCurrentLine(selectedFileIndex)
	} else {
		explorer.setCurrentDirectory(explorer.context.CurrentPath)
	}
	explorer.currentFocusedWidget = explorer.currentList
	explorer.Draw()
	return nil
}

func (explorer *Explorer) Draw() {
	explorer.listFlex.Clear()
	if explorer.parentList != nil {
		explorer.listFlex.AddItem(explorer.parentList, 0, 1, false)
		explorer.listFlex.AddItem(tview.NewBox(), 2, 0, false)
	}
	if explorer.currentList != nil {
		explorer.listFlex.AddItem(explorer.currentList, 0, 2, true)
		explorer.listFlex.AddItem(tview.NewBox(), 2, 0, false)
	}
	if explorer.selectedList != nil {
		explorer.listFlex.AddItem(explorer.selectedList, 0, 3, false)
	}
	explorer.rootFlex.Clear()
	explorer.rootFlex.SetDirection(tview.FlexRow)
	if explorer.header != nil {
		explorer.rootFlex.AddItem(explorer.header, 3, 0, false)
	}
	explorer.rootFlex.AddItem(explorer.listFlex, 0, 1, true)
	if explorer.footer != nil {
		explorer.rootFlex.AddItem(explorer.footer, 3, 0, false)
	}
	explorer.context.App.SetRoot(explorer.rootFlex, true)
	explorer.context.App.SetFocus(explorer.currentFocusedWidget)
	explorer.applyTheme()
	explorer.highlightSearchInput()
}

func (explorer *Explorer) GetInputCapture() func(*tcell.EventKey) *tcell.EventKey {
	if explorer.isFooterActive && explorer.footer != nil {
		return explorer.footer.GetInputCapture()
	}
	return explorer.currentList.GetInputCapture()
}

// setSelectedDirectory updates the selected directory/file preview
func (explorer *Explorer) setSelectedDirectory(selectedPath string) error {
	selectedAbsolutePath, _ := filepath.Abs(selectedPath)
	isDirEmpty, _ := helper.IsDirectoryEmpty(selectedAbsolutePath)
	if isDirEmpty {
		explorer.selectedList = tview.NewTextArea().SetText("Directory is empty...", false)
		return nil
	}
	selectedDirectoryIndex := explorer.directoryToIndexMap[selectedAbsolutePath]

	newSelectedList, err := helper.LoadDirectory(selectedPath, explorer.context.ShowHiddenFiles, false, false, explorer.markedFiles)
	if err != nil {
		return err
	}

	if newSelectedList == nil {
		explorer.selectedList, err = helper.LoadFilePreview(selectedPath, nil)
		if err != nil {
			return err
		}
	} else {
		newSelectedList.SetCurrentItem(selectedDirectoryIndex)
		explorer.selectedList = newSelectedList
	}
	return nil
}

func (explorer *Explorer) setParentDirectory(path string) error {
	currentAbsolutePath, _ := filepath.Abs(path)
	if currentAbsolutePath == "/" {
		emptyList := tview.NewList().ShowSecondaryText(false)
		explorer.parentList = emptyList
	} else {
		parentPath := filepath.Join(currentAbsolutePath, "..")
		newParentList, err := helper.LoadDirectory(parentPath, explorer.context.ShowHiddenFiles, false, false, explorer.markedFiles)
		if err != nil {
			return err
		}

		parentDirectoryIndex := helper.FindExactItem(newParentList, filepath.Base(currentAbsolutePath))

		parentAbsolutePath, _ := filepath.Abs(parentPath)
		explorer.directoryToIndexMap[parentAbsolutePath] = parentDirectoryIndex
		newParentList.SetCurrentItem(parentDirectoryIndex)
		explorer.parentList = newParentList
	}
	return nil
}

func (explorer *Explorer) setCurrentDirectory(path string) error {
	// Update current directory
	currentAbsolutePath, _ := filepath.Abs(path)
	currentDirectoryIndex := explorer.directoryToIndexMap[currentAbsolutePath]
	newCurrentList, err := helper.LoadDirectory(currentAbsolutePath, explorer.context.ShowHiddenFiles, false, false, explorer.markedFiles)
	if err != nil {
		return err
	}

	newCurrentList.SetInputCapture(explorer.currentList.GetInputCapture())
	newCurrentList.SetCurrentItem(currentDirectoryIndex)
	currentDirectoryIndex = newCurrentList.GetCurrentItem()
	// update index in case it was clipped
	explorer.currentList = newCurrentList

	// Update parent directory
	explorer.setParentDirectory(currentAbsolutePath)

	// Update selected directory
	_, selectedName := explorer.currentList.GetItemText(currentDirectoryIndex)
	selectedPath := filepath.Join(currentAbsolutePath, selectedName)
	if err := explorer.setSelectedDirectory(selectedPath); err != nil {
		return err
	}

	// Update header
	explorer.setHeader(currentAbsolutePath)

	explorer.searchInCurrentDirectory()
	explorer.context.CurrentPath = currentAbsolutePath
	explorer.currentFocusedWidget = explorer.currentList
	return nil
}

func (explorer *Explorer) setHeader(text string) {
	explorer.header.SetBorder(true).SetTitle("Explore").Blur()
	explorer.header.SetText(text)
}

// setCurrentLine updates the current line selection
func (explorer *Explorer) setCurrentLine(lineIndex int) error {
	if lineIndex < 0 {
		lineIndex = 0
	}
	if lineIndex >= explorer.currentList.GetItemCount() {
		lineIndex = explorer.currentList.GetItemCount() - 1
	}
	explorer.currentList.SetCurrentItem(lineIndex)
	currentAbsolutePath, _ := filepath.Abs(explorer.context.CurrentPath)
	explorer.directoryToIndexMap[currentAbsolutePath] = lineIndex

	_, selectedName := explorer.currentList.GetItemText(lineIndex)
	return explorer.setSelectedDirectory(filepath.Join(explorer.context.CurrentPath, selectedName))
}

func (explorer *Explorer) searchInCurrentDirectory() {
	if explorer.currentSearchTerm == "" {
		return
	}
	explorer.currentSearchIndeces = explorer.currentList.FindItems(explorer.currentSearchTerm, "", false, true)
}

func (explorer *Explorer) runFooterCommand(inputText string) {
	switch inputText[0] {
	case '/':
		explorer.currentSearchTerm = inputText[1:]
		explorer.context.RecentSearches = helper.AppendStringToUniqueList(explorer.context.RecentSearches, explorer.currentSearchTerm)
		explorer.searchInCurrentDirectory()
		if len(explorer.currentSearchIndeces) > 0 {
			explorer.setCurrentLine(explorer.currentSearchIndeces[0])
		}
	case ':':
		command := inputText[1:]
		explorer.runCommand(command)
	}
	explorer.currentFocusedWidget = explorer.currentList
}

func (explorer *Explorer) cycleRecentSearches(backwards bool) (int, string) {
	return helper.CycleRecentList(explorer.context.RecentSearches, explorer.cycleRecentSearchesIdx, backwards)
}

func (explorer *Explorer) cycleRecentCommands(backwards bool) (int, string) {
	return helper.CycleRecentList(explorer.context.RecentCommands, explorer.cycleRecentCommandsIdx, backwards)
}

func (explorer *Explorer) handleFooterInput(prompt string) {
	explorer.isFooterActive = true
	explorer.footer = tview.NewInputField().SetText(prompt)
	explorer.footer.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		return event
	})
	explorer.footer.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		currentText := explorer.footer.GetText()
		if event.Key() == tcell.KeyUp {
			if strings.HasPrefix(currentText, "/") {
				var search string
				explorer.cycleRecentSearchesIdx, search = explorer.cycleRecentSearches(true)
				if search != "" {
					currentText = "/" + search
					explorer.footer.SetText(currentText)
				} else {
					return nil
				}
			} else if strings.HasPrefix(currentText, ":") {
				var command string
				explorer.cycleRecentCommandsIdx, command = explorer.cycleRecentCommands(true)
				if command != "" {
					currentText = ":" + command
					explorer.footer.SetText(currentText)
				} else {
					return nil
				}
			}
		}
		if event.Key() == tcell.KeyDown {
			if strings.HasPrefix(currentText, "/") {
				var search string
				explorer.cycleRecentSearchesIdx, search = explorer.cycleRecentSearches(false)
				if search != "" {
					currentText = "/" + search
					explorer.footer.SetText(currentText)
				} else {
					return nil
				}
			} else if strings.HasPrefix(currentText, ":") {
				var command string
				explorer.cycleRecentCommandsIdx, command = explorer.cycleRecentCommands(false)
				if command != "" {
					currentText = ":" + command
					explorer.footer.SetText(currentText)
				} else {
					return nil
				}
			}
		}
		return event
	})
	explorer.footer.SetDoneFunc(
		func(key tcell.Key) {
			defer explorer.Draw()
			if key == tcell.KeyEnter {
				inputText := explorer.footer.GetText()
				explorer.runFooterCommand(inputText)
				explorer.currentFocusedWidget = explorer.currentList
			}
			explorer.isFooterActive = false
		},
	)
	explorer.footer.SetChangedFunc(
		func(text string) {
			defer explorer.Draw()
			if strings.HasPrefix(text, "/") {
				explorer.searchInput = strings.TrimPrefix(text, "/")
			}
		},
	)
	explorer.currentFocusedWidget = explorer.footer
}

func (explorer *Explorer) setLastDirectory() error {
	// Write current path to a temporary file that can be sourced by shell
	tempFile := os.Getenv("HOME") + "/.gofileyourself_lastdir"
	if err := os.WriteFile(tempFile, []byte(explorer.context.CurrentPath), 0o644); err != nil {
		return err
	}
	return nil
}

func (explorer *Explorer) quitAndChangeDirectory() {
	err := explorer.setLastDirectory()
	if err != nil {
		return
	}
	explorer.context.App.Stop()
}

func (explorer *Explorer) deleteCurrentFile(isForcedDelete bool) {
	_, currentName := explorer.currentList.GetItemText(explorer.currentList.GetCurrentItem())
	currentPath := filepath.Join(explorer.context.CurrentPath, currentName)
	if isForcedDelete {
		if err := os.RemoveAll(currentPath); err != nil {
			return
		}
	} else {
		if err := os.Remove(currentPath); err != nil {
			return
		}
	}
	explorer.setCurrentDirectory(explorer.context.CurrentPath)
}

func (explorer *Explorer) yankCurrentFile() {
	_, currentName := explorer.currentList.GetItemText(explorer.currentList.GetCurrentItem())
	explorer.yankedFile = explorer.context.CurrentPath + "/" + currentName
}

func (explorer *Explorer) pasteYankedFile() {
	if explorer.yankedFile == "" {
		return
	}
	destinationPath := filepath.Join(explorer.context.CurrentPath, filepath.Base(explorer.yankedFile))
	if err := helper.CopyFile(explorer.yankedFile, destinationPath); err != nil {
		return
	}
	explorer.setCurrentDirectory(explorer.context.CurrentPath)
}

func (explorer *Explorer) renameMarkedFiles() {
	tempFile, err := os.CreateTemp("", "gofileyourself_rm")
	if err != nil {
		return
	}
	defer tempFile.Close()
	defer os.Remove(tempFile.Name())

	if len(explorer.markedFiles) == 0 {
		return
	}
	for _, file := range explorer.markedFiles {
		fmt.Fprintln(tempFile, filepath.Base(file))
	}
	helper.OpenInNvim(tempFile.Name(), explorer.context.ChooseFilePath, explorer.context.App, explorer.context.Config.HistoryLen)

	file, err := os.Open(tempFile.Name())
	if err != nil {
		return
	}
	fileReader := bufio.NewReader(file)
	lineIdx := 0
	for {
		line, _, err := fileReader.ReadLine()
		if len(line) > 0 {
			if lineIdx <= len(explorer.markedFiles) {
				path := explorer.markedFiles[lineIdx]
				helper.RenameFile(path, string(filepath.Join(filepath.Dir(path), string(line))))
			}
			lineIdx++
		}
		if err != nil {
			break
		}
	}
	explorer.markedFiles = []string{}
	explorer.setCurrentDirectory(explorer.context.CurrentPath)
}

func (explorer *Explorer) deleteMarkedFiles(isForcedDelete bool) {
	filesToRemove := []string{}
	for _, file := range explorer.markedFiles {
		if isForcedDelete {
			if err := os.RemoveAll(file); err != nil {
				return
			} else {
				filesToRemove = append(filesToRemove, file)
			}
		} else {
			if err := os.Remove(file); err != nil {
				return
			} else {
				filesToRemove = append(filesToRemove, file)
			}
		}
	}
	for _, file := range filesToRemove {
		explorer.markedFiles = helper.DeleteItem(explorer.markedFiles, file)
	}
	explorer.setCurrentDirectory(explorer.context.CurrentPath)
}

func (explorer *Explorer) toggleMarkForCurrentFile() {
	_, currentName := explorer.currentList.GetItemText(explorer.currentList.GetCurrentItem())
	filePath := filepath.Join(explorer.context.CurrentPath, currentName)
	if slices.Contains(explorer.markedFiles, filePath) {
		explorer.markedFiles = helper.DeleteItem(explorer.markedFiles, filePath)
	} else {
		explorer.markedFiles = append(explorer.markedFiles, filePath)
	}
	explorer.setCurrentDirectory(explorer.context.CurrentPath)
	explorer.setCurrentLine(explorer.currentList.GetCurrentItem() + 1)
}

func (explorer *Explorer) unmarkAllFiles() {
	explorer.markedFiles = []string{}
	explorer.setCurrentDirectory(explorer.context.CurrentPath)
}

func (explorer *Explorer) yankMarkedFiles() {
	explorer.yankedMarkedFiles = explorer.markedFiles
}

func (explorer *Explorer) pasteMarkedFiles() {
	for _, file := range explorer.yankedMarkedFiles {
		destinationPath := filepath.Join(explorer.context.CurrentPath, filepath.Base(file))
		helper.CopyFile(file, destinationPath)
	}
	explorer.setCurrentDirectory(explorer.context.CurrentPath)
}

func (explorer *Explorer) setAnchor(key string) {
	_, currentName := explorer.currentList.GetItemText(explorer.currentList.GetCurrentItem())
	anchor := key + " > " + explorer.context.CurrentPath + "/" + currentName
	homeDir, _ := os.UserHomeDir()
	anchorFilePath := homeDir + "/.gofileyourself_anchors"
	helper.AppendOrReplaceLineInFile(anchorFilePath, anchor)
}

func (explorer *Explorer) jumpToAnchor(key string) {
	homeDir, _ := os.UserHomeDir()
	anchor := homeDir + "/.gofileyourself_anchors"
	anchor, err := helper.GetLineWithKey(anchor, key)
	if err != nil {
		return
	}
	anchorPrefix := key + " > "
	anchorPath := strings.TrimPrefix(anchor, anchorPrefix)
	anchorBase := filepath.Base(anchorPath)
	anchorDir := filepath.Dir(anchorPath)
	explorer.setCurrentDirectory(anchorDir)
	explorer.setCurrentLine(helper.FindExactItem(explorer.currentList, anchorBase))
}

func (explorer *Explorer) cycleRecentlyVisited(isBackward bool) {
	if isBackward {
		explorer.cycleRecentlyVisitedIdx--
	} else {
		explorer.cycleRecentlyVisitedIdx++
	}
	if explorer.cycleRecentlyVisitedIdx < 0 {
		explorer.cycleRecentlyVisitedIdx = 0
	}
	recentFile, err := helper.GetRecentFile(explorer.cycleRecentlyVisitedIdx, explorer.context.Config.HistoryLen)
	if err != nil {
		if isBackward {
			explorer.cycleRecentlyVisitedIdx = 0
		} else {
			explorer.cycleRecentlyVisitedIdx--
		}
		return
	}
	explorer.setCurrentDirectory(filepath.Dir(recentFile))
	explorer.setCurrentLine(helper.FindExactItem(explorer.currentList, filepath.Base(recentFile)))
}

// Run starts the file explorer
func (explorer *Explorer) Run() error {
	return explorer.context.App.SetRoot(explorer.Root(), true).Run()
}

// GetCurrentList returns the current list widget
func (explorer *Explorer) GetCurrentList() *tview.List {
	return explorer.currentList
}
