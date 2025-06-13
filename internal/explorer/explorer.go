package explorer

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"

	"github.com/thilobro/gofileyourself/internal/formatter"
	"github.com/thilobro/gofileyourself/internal/helper"
	"github.com/thilobro/gofileyourself/internal/theme"
	"github.com/thilobro/gofileyourself/internal/widget"

	gostring "github.com/boyter/go-string"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

const (
	MAX_SCROLL_AMOUNT = 20
)

func init() {
	formatter.RegisterCustomFormatter()
}

// FileExplorer represents the state and behavior of the file explorer
type FileExplorer struct {
	context              *widget.Context
	currentList          *tview.List
	parentList           tview.Primitive
	selectedList         tview.Primitive
	rootFlex             *tview.Flex
	listFlex             *tview.Flex
	directoryToIndexMap  map[string]int
	footer               *tview.InputField
	isFooterActive       bool
	header               *tview.TextView
	searchInput          string
	currentSearchTerm    string
	currentSearchIndeces []int
	currentFocusedWidget tview.Primitive
	keyBuffer            string
	yankedFile           string
	markedFiles          []string
	yankedMarkedFiles    []string
	cycleRecentPosition  int
}

func (fe *FileExplorer) Root() tview.Primitive {
	return fe.rootFlex
}

func (fe *FileExplorer) applyTheme() {
	explorerTheme := theme.GetExplorerTheme()

	// Set global background through root flex
	fe.rootFlex.SetBackgroundColor(explorerTheme.Bg0)
	fe.listFlex.SetBackgroundColor(explorerTheme.Bg0)

	// Style the lists
	fe.currentList.
		SetMainTextColor(explorerTheme.Fg1).
		SetSelectedTextColor(explorerTheme.Black).
		SetSelectedBackgroundColor(explorerTheme.Aqua).
		SetBackgroundColor(explorerTheme.Bg0)

	if fe.parentList != nil {
		if list, ok := fe.parentList.(*tview.List); ok {
			list.
				SetMainTextColor(explorerTheme.Fg1).
				SetSelectedTextColor(explorerTheme.Black).
				SetSelectedBackgroundColor(explorerTheme.Blue).
				SetBackgroundColor(explorerTheme.Bg0)
		}
	}

	// Style the selected list/preview
	if list, ok := fe.selectedList.(*tview.List); ok {
		list.
			SetMainTextColor(explorerTheme.Fg1).
			SetSelectedTextColor(explorerTheme.Black).
			SetSelectedBackgroundColor(explorerTheme.Green).
			SetBackgroundColor(explorerTheme.Bg0)
	} else if textView, ok := fe.selectedList.(*tview.TextView); ok {
		textView.
			SetTextColor(explorerTheme.Fg0).
			SetBackgroundColor(explorerTheme.Bg0)
	}

	// Style the footer
	if fe.footer != nil {
		fe.footer.
			SetFieldBackgroundColor(explorerTheme.Bg1).
			SetFieldTextColor(explorerTheme.Fg0).
			SetBackgroundColor(explorerTheme.Bg0)
	}
}

func (fe *FileExplorer) highlightSearchInput() {
	for i := 0; i < fe.currentList.GetItemCount(); i++ {
		_, text := fe.currentList.GetItemText(i)
		indeces := gostring.IndexAll(text, fe.searchInput, -1)
		fe.currentList.SetItemText(i, gostring.HighlightString(text, indeces, "[red::b]", "[-::-]"), text)
	}
}

// NewFileExplorer creates and initializes a new FileExplorer
func NewFileExplorer(context *widget.Context) (*FileExplorer, error) {
	fe := &FileExplorer{
		context:             context,
		currentList:         tview.NewList(),
		parentList:          tview.NewList(),
		selectedList:        tview.NewList(),
		directoryToIndexMap: make(map[string]int),
		listFlex:            tview.NewFlex(),
		rootFlex:            tview.NewFlex(),
		footer:              tview.NewInputField(),
		isFooterActive:      false,
		header:              tview.NewTextView(),
		searchInput:         "",
		currentSearchTerm:   "",
		keyBuffer:           "",
		yankedFile:          "",
		markedFiles:         []string{},
		yankedMarkedFiles:   []string{},
		cycleRecentPosition: 0,
	}

	if err := fe.initialize(); err != nil {
		return nil, err
	}

	return fe, nil
}

// initialize sets up the initial state of the FileExplorer
func (fe *FileExplorer) initialize() error {
	fe.SetupKeyBindings()
	if fe.context.SelectedFilePath != nil {
		fe.context.CurrentPath = filepath.Dir(*fe.context.SelectedFilePath)
		fe.setCurrentDirectory(fe.context.CurrentPath)
		selectedFileIndex := helper.FindExactItem(fe.currentList, filepath.Base(*fe.context.SelectedFilePath))
		fe.currentList.SetCurrentItem(selectedFileIndex)
	} else {
		fe.setCurrentDirectory(fe.context.CurrentPath)
	}
	fe.currentFocusedWidget = fe.currentList
	fe.setLastDirectory()
	fe.Draw()
	return nil
}

// draw updates the UI
func (fe *FileExplorer) Draw() {
	fe.listFlex.Clear()
	if fe.parentList != nil {
		fe.listFlex.AddItem(fe.parentList, 0, 1, false)
		fe.listFlex.AddItem(tview.NewBox(), 2, 0, false)
	}
	if fe.currentList != nil {
		fe.listFlex.AddItem(fe.currentList, 0, 2, true)
		fe.listFlex.AddItem(tview.NewBox(), 2, 0, false)
	}
	if fe.selectedList != nil {
		fe.listFlex.AddItem(fe.selectedList, 0, 3, false)
	}
	fe.rootFlex.Clear()
	fe.rootFlex.SetDirection(tview.FlexRow)
	if fe.header != nil {
		fe.rootFlex.AddItem(fe.header, 3, 0, false)
	}
	fe.rootFlex.AddItem(fe.listFlex, 0, 1, true)
	if fe.footer != nil {
		fe.rootFlex.AddItem(fe.footer, 1, 0, false)
	}
	fe.context.App.SetRoot(fe.rootFlex, true)
	fe.context.App.SetFocus(fe.currentFocusedWidget)
	fe.applyTheme()
	fe.highlightSearchInput()
}

func (fe *FileExplorer) GetInputCapture() func(*tcell.EventKey) *tcell.EventKey {
	if fe.isFooterActive && fe.footer != nil {
		return fe.footer.GetInputCapture()
	}
	return fe.currentList.GetInputCapture()
}

// setSelectedDirectory updates the selected directory/file preview
func (fe *FileExplorer) setSelectedDirectory(selectedPath string) error {
	selectedAbsolutePath, _ := filepath.Abs(selectedPath)
	isDirEmpty, _ := helper.IsDirectoryEmpty(selectedAbsolutePath)
	if isDirEmpty {
		fe.selectedList = tview.NewTextArea().SetText("Directory is empty", false)
		return nil
	}
	selectedDirectoryIndex := fe.directoryToIndexMap[selectedAbsolutePath]

	newSelectedList, err := helper.LoadDirectory(selectedPath, fe.context.ShowHiddenFiles, false, fe.markedFiles)
	if err != nil {
		return err
	}

	if newSelectedList == nil {
		fe.selectedList, err = helper.LoadFilePreview(selectedPath)
		if err != nil {
			return err
		}
	} else {
		newSelectedList.SetCurrentItem(selectedDirectoryIndex)
		fe.selectedList = newSelectedList
	}
	return nil
}

func (fe *FileExplorer) setParentDirectory(path string) error {
	currentAbsolutePath, _ := filepath.Abs(path)
	if currentAbsolutePath == "/" {
		emptyList := tview.NewList().ShowSecondaryText(false)
		fe.parentList = emptyList
	} else {
		parentPath := filepath.Join(currentAbsolutePath, "..")
		newParentList, err := helper.LoadDirectory(parentPath, fe.context.ShowHiddenFiles, false, fe.markedFiles)
		if err != nil {
			return err
		}

		parentDirectoryIndex := helper.FindExactItem(newParentList, filepath.Base(currentAbsolutePath))

		parentAbsolutePath, _ := filepath.Abs(parentPath)
		fe.directoryToIndexMap[parentAbsolutePath] = parentDirectoryIndex
		newParentList.SetCurrentItem(parentDirectoryIndex)
		fe.parentList = newParentList
	}
	return nil
}

// setCurrentDirectory changes the current directory and updates related views
func (fe *FileExplorer) setCurrentDirectory(path string) error {
	isDirEmpty, _ := helper.IsDirectoryEmpty(path)
	if isDirEmpty {
		if fe.context.CurrentPath == path {
			fe.setCurrentDirectory(path + "/..")
		}
		return nil
	}

	// Update current directory
	currentAbsolutePath, _ := filepath.Abs(path)
	currentDirectoryIndex := fe.directoryToIndexMap[currentAbsolutePath]
	newCurrentList, err := helper.LoadDirectory(currentAbsolutePath, fe.context.ShowHiddenFiles, false, fe.markedFiles)
	if err != nil {
		return err
	}

	newCurrentList.SetInputCapture(fe.currentList.GetInputCapture())
	newCurrentList.SetCurrentItem(currentDirectoryIndex)
	currentDirectoryIndex = newCurrentList.GetCurrentItem()
	// update index in case it was clipped
	fe.currentList = newCurrentList

	// Update parent directory
	fe.setParentDirectory(currentAbsolutePath)

	// Update selected directory
	_, selectedName := fe.currentList.GetItemText(currentDirectoryIndex)
	selectedPath := filepath.Join(currentAbsolutePath, selectedName)
	if err := fe.setSelectedDirectory(selectedPath); err != nil {
		return err
	}

	// Update header
	fe.setHeader(currentAbsolutePath)

	fe.searchInCurrentDirectory()
	fe.context.CurrentPath = currentAbsolutePath
	fe.currentFocusedWidget = fe.currentList
	return nil
}

func (fe *FileExplorer) setHeader(text string) {
	fe.header.SetBorder(true).SetTitle("Explore").Blur()
	fe.header.SetText(text)
}

// setCurrentLine updates the current line selection
func (fe *FileExplorer) setCurrentLine(lineIndex int) error {
	if lineIndex < 0 {
		lineIndex = 0
	}
	if lineIndex >= fe.currentList.GetItemCount() {
		lineIndex = fe.currentList.GetItemCount() - 1
	}
	fe.currentList.SetCurrentItem(lineIndex)
	currentAbsolutePath, _ := filepath.Abs(fe.context.CurrentPath)
	fe.directoryToIndexMap[currentAbsolutePath] = lineIndex

	_, selectedName := fe.currentList.GetItemText(lineIndex)
	return fe.setSelectedDirectory(filepath.Join(fe.context.CurrentPath, selectedName))
}

func (fe *FileExplorer) searchInCurrentDirectory() {
	if fe.currentSearchTerm == "" {
		return
	}
	fe.currentSearchIndeces = fe.currentList.FindItems(fe.currentSearchTerm, "", false, true)
}

func (fe *FileExplorer) runFooterCommand(inputText string) {
	switch inputText[0] {
	case '/':
		fe.currentSearchTerm = inputText[1:]
		fe.searchInCurrentDirectory()
		if len(fe.currentSearchIndeces) > 0 {
			fe.setCurrentLine(fe.currentSearchIndeces[0])
		}
	case ':':
		command := inputText[1:]
		parts := strings.Split(command, " ")
		switch parts[0] {
		case "q":
			fe.context.App.Stop()
		case "mkdir":
			if len(parts) > 1 {
				helper.CreateDirectory(filepath.Join(fe.context.CurrentPath, parts[1]))
				fe.setCurrentDirectory(fe.context.CurrentPath)
			}
		case "rename":
			if len(parts) > 1 {
				_, currentName := fe.currentList.GetItemText(fe.currentList.GetCurrentItem())
				currentPath := filepath.Join(fe.context.CurrentPath, currentName)
				helper.RenameFile(currentPath, parts[1])
				fe.setCurrentDirectory(fe.context.CurrentPath)
			}
		case "mrename":
			fe.renameMarkedFiles()
		case "touch":
			if len(parts) > 1 {
				helper.TouchFile(filepath.Join(fe.context.CurrentPath, parts[1]))
				fe.setCurrentDirectory(fe.context.CurrentPath)
			}
		}
	}
	fe.currentFocusedWidget = fe.currentList
}

func (fe *FileExplorer) handleFooterInput(prompt string) {
	fe.isFooterActive = true
	fe.footer = tview.NewInputField().SetText(prompt)
	fe.footer.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		return event
	})
	fe.footer.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		currentText := fe.footer.GetText()
		if event.Key() == tcell.KeyBackspace2 {
			currentTextLen := len(currentText)
			if currentTextLen <= 1 {
				return nil
			}
			currentText = currentText[:currentTextLen-1]
		} else if event.Key() == tcell.KeyEnter {
			return event
		} else {
			currentText = currentText + string(event.Rune())
		}
		fe.footer.SetText(currentText)
		return nil
	})
	fe.footer.SetDoneFunc(
		func(key tcell.Key) {
			if key == tcell.KeyEnter {
				inputText := fe.footer.GetText()
				fe.runFooterCommand(inputText)
				fe.currentFocusedWidget = fe.currentList
			}
			fe.Draw()
			fe.isFooterActive = false
		},
	)
	fe.footer.SetChangedFunc(
		func(text string) {
			defer fe.Draw()
			fe.searchInput = strings.TrimPrefix(text, "/")
		},
	)
	fe.currentFocusedWidget = fe.footer
	fe.Draw()
}

func (fe *FileExplorer) setLastDirectory() error {
	// Write current path to a temporary file that can be sourced by shell
	tempFile := os.Getenv("HOME") + "/.gofileyourself_lastdir"
	if err := os.WriteFile(tempFile, []byte(fe.context.CurrentPath), 0o644); err != nil {
		return err
	}
	return nil
}

func (fe *FileExplorer) quitAndChangeDirectory() {
	err := fe.setLastDirectory()
	if err != nil {
		return
	}
	fe.context.App.Stop()
}

func (fe *FileExplorer) deleteCurrentFile(isForcedDelete bool) {
	_, currentName := fe.currentList.GetItemText(fe.currentList.GetCurrentItem())
	currentPath := filepath.Join(fe.context.CurrentPath, currentName)
	if isForcedDelete {
		if err := os.RemoveAll(currentPath); err != nil {
			return
		}
	} else {
		if err := os.Remove(currentPath); err != nil {
			return
		}
	}
	fe.setCurrentDirectory(fe.context.CurrentPath)
}

func (fe *FileExplorer) yankCurrentFile() {
	_, currentName := fe.currentList.GetItemText(fe.currentList.GetCurrentItem())
	fe.yankedFile = fe.context.CurrentPath + "/" + currentName
}

func (fe *FileExplorer) pasteYankedFile() {
	if fe.yankedFile == "" {
		return
	}
	destinationPath := filepath.Join(fe.context.CurrentPath, filepath.Base(fe.yankedFile))
	if err := helper.CopyFile(fe.yankedFile, destinationPath); err != nil {
		return
	}
	fe.setCurrentDirectory(fe.context.CurrentPath)
}

func (fe *FileExplorer) renameMarkedFiles() {
	tempFile, err := os.CreateTemp("", "gofileyourself_rm")
	if err != nil {
		return
	}
	defer tempFile.Close()
	defer os.Remove(tempFile.Name())

	if len(fe.markedFiles) == 0 {
		return
	}
	for _, file := range fe.markedFiles {
		fmt.Fprintln(tempFile, filepath.Base(file))
	}
	helper.OpenInNvim(tempFile.Name(), fe.context.ChooseFilePath, fe.context.App, fe.context.Config.HistoryLen)

	file, err := os.Open(tempFile.Name())
	if err != nil {
		return
	}
	fileReader := bufio.NewReader(file)
	lineIdx := 0
	for {
		line, _, err := fileReader.ReadLine()
		if len(line) > 0 {
			if lineIdx <= len(fe.markedFiles) {
				path := fe.markedFiles[lineIdx]
				helper.RenameFile(path, string(filepath.Join(filepath.Dir(path), string(line))))
			}
			lineIdx++
		}
		if err != nil {
			break
		}
	}
	fe.markedFiles = []string{}
	fe.setCurrentDirectory(fe.context.CurrentPath)
}

func (fe *FileExplorer) deleteMarkedFiles(isForcedDelete bool) {
	filesToRemove := []string{}
	for _, file := range fe.markedFiles {
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
		fe.markedFiles = helper.DeleteItem(fe.markedFiles, file)
	}
	fe.setCurrentDirectory(fe.context.CurrentPath)
}

func (fe *FileExplorer) toggleMarkForCurrentFile() {
	_, currentName := fe.currentList.GetItemText(fe.currentList.GetCurrentItem())
	filePath := filepath.Join(fe.context.CurrentPath, currentName)
	if slices.Contains(fe.markedFiles, filePath) {
		fe.markedFiles = helper.DeleteItem(fe.markedFiles, filePath)
	} else {
		fe.markedFiles = append(fe.markedFiles, filePath)
	}
	fe.setCurrentDirectory(fe.context.CurrentPath)
	fe.setCurrentLine(fe.currentList.GetCurrentItem() + 1)
}

func (fe *FileExplorer) unmarkAllFiles() {
	fe.markedFiles = []string{}
	fe.setCurrentDirectory(fe.context.CurrentPath)
}

func (fe *FileExplorer) yankMarkedFiles() {
	fe.yankedMarkedFiles = fe.markedFiles
}

func (fe *FileExplorer) pasteMarkedFiles() {
	for _, file := range fe.yankedMarkedFiles {
		destinationPath := filepath.Join(fe.context.CurrentPath, filepath.Base(file))
		helper.CopyFile(file, destinationPath)
	}
	fe.setCurrentDirectory(fe.context.CurrentPath)
}

func (fe *FileExplorer) setAnchor(key string) {
	_, currentName := fe.currentList.GetItemText(fe.currentList.GetCurrentItem())
	anchor := key + " > " + fe.context.CurrentPath + "/" + currentName
	homeDir, _ := os.UserHomeDir()
	anchorFilePath := homeDir + "/.gofileyourself_anchors"
	helper.AppendOrReplaceLineInFile(anchorFilePath, anchor)
}

func (fe *FileExplorer) jumpToAnchor(key string) {
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
	fe.setCurrentDirectory(anchorDir)
	fe.setCurrentLine(helper.FindExactItem(fe.currentList, anchorBase))
}

func (fe *FileExplorer) cycleRecent(isBackward bool) {
	if isBackward {
		fe.cycleRecentPosition--
	} else {
		fe.cycleRecentPosition++
	}
	if fe.cycleRecentPosition < 0 {
		fe.cycleRecentPosition = 0
	}
	recentFile, err := helper.GetRecentFile(fe.cycleRecentPosition, fe.context.Config.HistoryLen)
	if err != nil {
		if isBackward {
			fe.cycleRecentPosition = 0
		} else {
			fe.cycleRecentPosition--
		}
		return
	}
	fe.setCurrentDirectory(filepath.Dir(recentFile))
	fe.setCurrentLine(helper.FindExactItem(fe.currentList, filepath.Base(recentFile)))
}

// setupKeyBindings configures keyboard input handling
func (fe *FileExplorer) SetupKeyBindings() {
	fe.currentList.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		defer fe.Draw()
		switch event.Key() {
		case tcell.KeyCtrlH:
			fe.context.ShowHiddenFiles = !fe.context.ShowHiddenFiles

			// Remember current selection before refresh
			_, currentName := fe.currentList.GetItemText(fe.currentList.GetCurrentItem())

			// Remember selected directory name if we're showing a directory
			var selectedName string
			if list, ok := fe.selectedList.(*tview.List); ok {
				_, selectedName = list.GetItemText(list.GetCurrentItem())
			}

			// Refresh the view
			if err := fe.setCurrentDirectory(fe.context.CurrentPath); err != nil {
				return event
			}

			// Restore current selection
			if idx := helper.FindExactItem(fe.currentList, currentName); idx >= 0 {
				fe.setCurrentLine(idx)
			}

			// Restore selected directory selection if applicable
			if list, ok := fe.selectedList.(*tview.List); ok {
				if idx := helper.FindExactItem(list, selectedName); idx >= 0 {
					list.SetCurrentItem(idx)
					absoluteSelectedPath, _ := filepath.Abs(filepath.Join(fe.context.CurrentPath, currentName))
					fe.directoryToIndexMap[absoluteSelectedPath] = idx
				}
			}
			return nil
		case tcell.KeyCtrlD: // scroll 10 down
			scrollAmount := fe.currentList.GetItemCount() / 2
			if scrollAmount > MAX_SCROLL_AMOUNT {
				scrollAmount = MAX_SCROLL_AMOUNT
			}
			fe.setCurrentLine(fe.currentList.GetCurrentItem() + scrollAmount)
			return nil
		case tcell.KeyCtrlU: // scroll 10 up
			scrollAmount := fe.currentList.GetItemCount() / 2
			if scrollAmount > MAX_SCROLL_AMOUNT {
				scrollAmount = MAX_SCROLL_AMOUNT
			}
			fe.setCurrentLine(fe.currentList.GetCurrentItem() - scrollAmount)
			return nil
		}
		rune := event.Rune()
		fe.keyBuffer += string(rune)
		if len(fe.keyBuffer) > 5 {
			fe.keyBuffer = fe.keyBuffer[4:]
		}
		if strings.HasSuffix(fe.keyBuffer, "gg") {
			fe.keyBuffer = ""
			fe.setCurrentLine(0)
			return nil
		} else if strings.HasSuffix(fe.keyBuffer, "dd") {
			fe.keyBuffer = ""
			fe.deleteCurrentFile(false)
			return nil
		} else if strings.HasSuffix(fe.keyBuffer, "DD") {
			fe.keyBuffer = ""
			fe.deleteCurrentFile(true)
			return nil
		} else if strings.HasSuffix(fe.keyBuffer, "yy") {
			fe.keyBuffer = ""
			fe.yankCurrentFile()
			return nil
		} else if strings.HasSuffix(fe.keyBuffer, "pp") {
			fe.keyBuffer = ""
			fe.pasteYankedFile()
			return nil
		} else if strings.HasSuffix(fe.keyBuffer, "mm") {
			fe.keyBuffer = ""
			fe.toggleMarkForCurrentFile()
			return nil
		} else if strings.HasSuffix(fe.keyBuffer, "mu") {
			fe.keyBuffer = ""
			fe.unmarkAllFiles()
			return nil
		} else if strings.HasSuffix(fe.keyBuffer, "md") {
			fe.keyBuffer = ""
			fe.deleteMarkedFiles(false)
			return nil
		} else if strings.HasSuffix(fe.keyBuffer, "mD") {
			fe.keyBuffer = ""
			fe.deleteMarkedFiles(true)
			return nil
		} else if strings.HasSuffix(fe.keyBuffer, "my") {
			fe.keyBuffer = ""
			fe.yankMarkedFiles()
			return nil
		} else if strings.HasSuffix(fe.keyBuffer, "mp") {
			fe.keyBuffer = ""
			fe.pasteMarkedFiles()
			return nil
		} else if match := regexp.MustCompile(`A([a-zA-Z0-9]+)$`).FindStringSubmatch(fe.keyBuffer); match != nil {
			key := match[1]
			fe.keyBuffer = ""
			fe.setAnchor(key)
			return nil
		} else if match := regexp.MustCompile(`a([a-zA-Z0-9]+)$`).FindStringSubmatch(fe.keyBuffer); match != nil {
			key := match[1]
			fe.keyBuffer = ""
			fe.jumpToAnchor(key)
			return nil
		}
		switch rune {
		case 'r':
			fe.cycleRecent(false)
			return nil
		case 'R':
			fe.cycleRecent(true)
			return nil
		case 'M':
			fe.toggleMarkForCurrentFile()
			return nil
		case 'G':
			fe.setCurrentLine(fe.currentList.GetItemCount() - 1)
			return nil
		case 'S':
			fe.quitAndChangeDirectory()
			return nil
		case 'j': // scroll down
			fe.setCurrentLine(fe.currentList.GetCurrentItem() + 1)
			return nil
		case 'k': // scroll up
			fe.setCurrentLine(fe.currentList.GetCurrentItem() - 1)
			return nil
		case 'q': // quit
			fe.context.App.Stop()
			return nil
		case 'l': // open dir or file
			currentItem := fe.currentList.GetCurrentItem()
			_, fileName := fe.currentList.GetItemText(currentItem)
			filePath := filepath.Join(fe.context.CurrentPath, fileName)
			fileInfo, err := os.Stat(filePath)
			if err != nil {
				return event
			}
			if fileInfo.IsDir() {
				if err := fe.setCurrentDirectory(filePath); err != nil {
					return event
				}
			} else {
				helper.OpenInNvim(filePath, fe.context.ChooseFilePath, fe.context.App, fe.context.Config.HistoryLen)
				return nil
			}
			return nil
		case 'h': // go up directory
			dirPath := filepath.Join(fe.context.CurrentPath, "..")
			if err := fe.setCurrentDirectory(dirPath); err != nil {
				return event
			}
			return nil
		case '/': // search
			fe.handleFooterInput("/")
			return nil
		case ':': // command
			fe.handleFooterInput(":")
			return nil
		case 'n': // cycle search
			if len(fe.currentSearchIndeces) > 0 {
				currentIndex := fe.currentList.GetCurrentItem()
				for _, index := range fe.currentSearchIndeces {
					if index > currentIndex {
						fe.setCurrentLine(index)
						return nil
					}
				}
				fe.setCurrentLine(fe.currentSearchIndeces[0])
			}
		case 'N': // cycle search backwards
			if len(fe.currentSearchIndeces) > 0 {
				currentIndex := fe.currentList.GetCurrentItem()
				for i := len(fe.currentSearchIndeces) - 1; i >= 0; i-- {
					if fe.currentSearchIndeces[i] < currentIndex {
						fe.setCurrentLine(fe.currentSearchIndeces[i])
						return nil
					}
				}
				// If no smaller index found, wrap around to the last item
				fe.setCurrentLine(fe.currentSearchIndeces[len(fe.currentSearchIndeces)-1])
			}

			return nil
		}
		return nil
	})
}

// Run starts the file explorer
func (fe *FileExplorer) Run() error {
	return fe.context.App.SetRoot(fe.Root(), true).Run()
}

// GetCurrentList returns the current list widget
func (fe *FileExplorer) GetCurrentList() *tview.List {
	return fe.currentList
}
