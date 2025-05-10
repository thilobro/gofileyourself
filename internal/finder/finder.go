package finder

import (
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/thilobro/gofileyourself/internal/helper"
	"github.com/thilobro/gofileyourself/internal/theme"
	"github.com/thilobro/gofileyourself/internal/widget"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"github.com/sahilm/fuzzy"
)

type Finder struct {
	context              *widget.Context
	rootFlex             *tview.Flex
	footer               *tview.InputField
	fileList             *tview.List
	searchedList         *tview.List
	selectedList         tview.Primitive
	currentFocusedWidget tview.Primitive
	searchTerm           string
	fuzzySearchQuit      chan bool
	listUpdateChan       chan *tview.List // Channel for list updates
}

func NewFinder(context *widget.Context) (*Finder, error) {
	finder := &Finder{
		context:         context,
		rootFlex:        tview.NewFlex(),
		footer:          tview.NewInputField(),
		fileList:        tview.NewList(),
		selectedList:    tview.NewList().ShowSecondaryText(false),
		searchTerm:      "",
		fuzzySearchQuit: make(chan bool),
		listUpdateChan:  make(chan *tview.List, 1), // Buffered channel
	}
	finder.resetFileList()
	finder.searchedList = finder.fileList
	finder.SetupKeyBindings()
	finder.currentFocusedWidget = finder.searchedList

	// Start list update handler
	go finder.handleListUpdates()

	err := finder.searchInDirectory()
	if err != nil {
		return nil, err
	}
	finder.setCurrentLine(0)

	return finder, nil
}

func (finder *Finder) handleListUpdates() {
	for newList := range finder.listUpdateChan {
		finder.searchedList = newList
		finder.context.App.QueueUpdateDraw(func() {
			finder.Draw()
		})
	}
}

func (finder *Finder) setCurrentLine(lineIndex int) error {
	if lineIndex < 0 || lineIndex >= finder.searchedList.GetItemCount() {
		return nil
	}
	finder.searchedList.SetCurrentItem(lineIndex)

	_, selectedName := finder.searchedList.GetItemText(lineIndex)
	return finder.setSelectedDirectory(filepath.Join(finder.context.CurrentPath, selectedName))
}

// setSelectedDirectory updates the selected directory/file preview
func (finder *Finder) setSelectedDirectory(selectedPath string) error {
	selectedAbsolutePath, _ := filepath.Abs(selectedPath)
	isDirEmpty, _ := helper.IsDirectoryEmpty(selectedAbsolutePath)
	if isDirEmpty {
		finder.selectedList = tview.NewTextArea().SetText("Directory is empty", false)
		return nil
	}
	selectedDirectoryIndex := 0

	newSelectedList, err := helper.LoadDirectory(selectedPath, finder.context.ShowHiddenFiles, false, []string{})
	if err != nil {
		return err
	}

	if newSelectedList == nil {
		finder.selectedList, err = helper.LoadFilePreview(selectedPath)
		if err != nil {
			return err
		}
	} else {
		newSelectedList.SetCurrentItem(selectedDirectoryIndex)
		finder.selectedList = newSelectedList
	}
	return nil
}

func (finder *Finder) SetupKeyBindings() {
	finder.rootFlex.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		defer finder.Draw()
		switch event.Key() {
		case tcell.KeyCtrlH:
			finder.context.ShowHiddenFiles = !finder.context.ShowHiddenFiles

			// Remember current selection before refresh
			_, currentName := finder.searchedList.GetItemText(finder.searchedList.GetCurrentItem())
			finder.resetFileList()
			finder.searchedList = finder.fileList

			// Restore current selection
			if idx := helper.FindExactItem(finder.searchedList, currentName); idx >= 0 {
				finder.setCurrentLine(idx)
			}
			go finder.manageFuzzySearch(finder.searchTerm)
			return nil
		case tcell.KeyUp:
			finder.setCurrentLine(finder.searchedList.GetCurrentItem() - 1)
			return nil
		case tcell.KeyDown:
			finder.setCurrentLine(finder.searchedList.GetCurrentItem() + 1)
			return nil
		case tcell.KeyEnter:
			currentItem := finder.searchedList.GetCurrentItem()
			_, fileName := finder.searchedList.GetItemText(currentItem)
			filePath := filepath.Join(finder.context.CurrentPath, fileName)

			fileInfo, err := os.Stat(filePath)
			if err != nil {
				return event
			}

			if fileInfo.IsDir() {
				finder.context.CurrentPath = filePath
				finder.context.OnWidgetResult(widget.Find, filePath)
				return nil
			}
			helper.OpenInNvim(filePath, finder.context.App)
			return nil
		}

		if finder.footer == nil {
			finder.handleFooterInput()
		}
		finder.currentFocusedWidget = finder.footer
		finder.footer.GetInputCapture()(event)
		return nil
	})
}

func (finder *Finder) handleFooterInput() {
	finder.footer = tview.NewInputField().SetText("/")
	finder.footer.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		currentText := finder.footer.GetText()
		if event.Key() == tcell.KeyBackspace2 {
			currentTextLen := len(currentText)
			if currentTextLen <= 1 {
				return nil
			}
			currentText = currentText[:currentTextLen-1]
		} else {
			currentText = currentText + string(event.Rune())
		}
		finder.footer.SetText(currentText)
		return nil
	})
	finder.searchedList = finder.fileList
	finder.footer.SetChangedFunc(
		func(text string) {
			defer finder.Draw()
			if text == "/" {
				finder.searchedList = finder.fileList
				return
			}
			currentInput := strings.TrimPrefix(text, "/")
			go finder.manageFuzzySearch(currentInput)
		},
	)
	finder.currentFocusedWidget = finder.footer
	finder.Draw()
}

func (finder *Finder) manageFuzzySearch(text string) {
	// Signal any existing search to stop
	select {
	case finder.fuzzySearchQuit <- true:
	default:
	}

	// Check if we should stop before starting
	select {
	case <-finder.fuzzySearchQuit:
		return
	default:
		finder.fuzzySearch(text)
	}
}

func (finder *Finder) fuzzySearch(text string) {
	itemNames := make([]string, finder.fileList.GetItemCount())
	for i := 0; i < finder.fileList.GetItemCount(); i++ {
		_, itemName := finder.fileList.GetItemText(i)
		itemNames[i] = itemName
	}

	// Split the search text into patterns
	patterns := strings.Fields(text)
	if len(patterns) == 0 {
		return
	}

	// Start with all matches from the first pattern
	matches := fuzzy.Find(patterns[0], itemNames)
	matchedStrs := make([]string, len(matches))
	for i, match := range matches {
		matchedStrs[i] = match.Str
	}

	// Keep track of all matched indexes for each string
	allMatchedIndexes := make(map[string][]int)
	for _, match := range matches {
		allMatchedIndexes[match.Str] = match.MatchedIndexes
	}

	// Filter through remaining patterns and collect their matched indexes
	for _, pattern := range patterns[1:] {
		matches = fuzzy.Find(pattern, matchedStrs)
		newMatchedStrs := make([]string, len(matches))
		for i, match := range matches {
			newMatchedStrs[i] = match.Str
			// Append new matched indexes to existing ones
			allMatchedIndexes[match.Str] = append(allMatchedIndexes[match.Str], match.MatchedIndexes...)
		}
		matchedStrs = newMatchedStrs
	}

	// Create new list with matches
	newList := tview.NewList().ShowSecondaryText(false)
	line := ""
	for _, match := range matches {
		for i := 0; i < len(match.Str); i++ {
			if slices.Contains(allMatchedIndexes[match.Str], i) {
				line = line + "[red::b]" + string(match.Str[i]) + "[-::-]"
			} else {
				line = line + string(match.Str[i])
			}
		}
		newList.AddItem(line, match.Str, 0, nil)
		line = ""
	}

	if len(matches) > 0 {
		newList.SetCurrentItem(0)
	}

	// Send the new list through the channel
	select {
	case finder.listUpdateChan <- newList:
	default:
		// If channel is full, skip this update
	}

	finder.searchTerm = text
}

func (finder *Finder) resetFileList() error {
	finder.fileList.Clear()
	fileList, err := helper.LoadDirectory(finder.context.CurrentPath, finder.context.ShowHiddenFiles, true, []string{})
	if err != nil {
		return err
	}
	finder.fileList = fileList
	return nil
}

func (finder *Finder) searchInDirectory() error {
	finder.resetFileList()
	finder.setCurrentLine(0)
	finder.Draw()
	finder.handleFooterInput()
	return nil
}

func (finder *Finder) Root() tview.Primitive {
	return finder.rootFlex
}

func (finder *Finder) Draw() {
	finder.rootFlex.Clear()
	listFlex := tview.NewFlex()
	listFlex.AddItem(finder.searchedList, 0, 1, true)
	if finder.selectedList != nil {
		listFlex.AddItem(finder.selectedList, 0, 1, true)
	}
	finder.rootFlex.SetDirection(tview.FlexRow)
	if finder.footer != nil {
		finder.rootFlex.AddItem(finder.footer, 3, 0, false)
	}
	finder.rootFlex.AddItem(listFlex, 0, 1, true)
	finder.context.App.SetFocus(finder.currentFocusedWidget)
	finder.applyTheme()
}

func (finder *Finder) Run() error {
	return finder.context.App.SetRoot(finder.Root(), true).Run()
}

func (finder *Finder) applyTheme() {
	explorerTheme := theme.GetExplorerTheme()

	// Set global background through root flex
	finder.rootFlex.SetBackgroundColor(explorerTheme.Bg0)

	// Style the lists
	finder.searchedList.
		SetMainTextColor(explorerTheme.Fg1).
		SetSelectedTextColor(explorerTheme.Black).
		SetSelectedBackgroundColor(explorerTheme.Aqua).
		SetBackgroundColor(explorerTheme.Bg0)

	// Style the footer
	if finder.footer != nil {
		finder.footer.
			SetFieldBackgroundColor(explorerTheme.Bg1).
			SetFieldTextColor(explorerTheme.Fg0).
			SetBackgroundColor(explorerTheme.Bg0).
			SetBorder(true).SetTitle("Find").Blur()
	}
}

// GetInputCapture returns the input capture function for the finder
func (finder *Finder) GetInputCapture() func(*tcell.EventKey) *tcell.EventKey {
	return finder.rootFlex.GetInputCapture()
}
