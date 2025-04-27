package finder

import (
	"path/filepath"
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
	showHiddenFiles      bool
}

func NewFinder(context *widget.Context) (*Finder, error) {
	finder := &Finder{
		context:         context,
		rootFlex:        tview.NewFlex(),
		footer:          tview.NewInputField(),
		fileList:        tview.NewList(),
		selectedList:    tview.NewList().ShowSecondaryText(false),
		showHiddenFiles: false,
	}
	finder.resetFileList()
	finder.searchedList = finder.fileList
	finder.SetupKeyBindings()
	finder.currentFocusedWidget = finder.searchedList

	err := finder.searchInDirectory()
	if err != nil {
		return nil, err
	}

	return finder, nil
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

	newSelectedList, err := helper.LoadDirectory(selectedPath, finder.showHiddenFiles, false)
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
		case tcell.KeyUp:
			finder.setCurrentLine(finder.searchedList.GetCurrentItem() - 1)
			return nil
		case tcell.KeyDown:
			finder.setCurrentLine(finder.searchedList.GetCurrentItem() + 1)
			return nil
		case tcell.KeyEnter:
			currentItem := finder.fileList.GetCurrentItem()
			_, fileName := finder.fileList.GetItemText(currentItem)
			filePath := filepath.Join(finder.context.CurrentPath, fileName)
			helper.OpenInNvim(filePath, finder.context.App)
			return nil

		}

		if finder.footer == nil {
			finder.handleFooterInput()
		}
		finder.currentFocusedWidget = finder.footer
		return event
	})
}

func (finder *Finder) handleFooterInput() {
	finder.footer = tview.NewInputField().SetText("/")
	finder.searchedList = finder.fileList
	finder.footer.SetChangedFunc(
		func(text string) {
			defer finder.Draw()
			finder.resetFileList()
			if text == "/" {
				finder.searchedList = finder.fileList
				return
			}
			currentInput := strings.TrimPrefix(text, "/")
			finder.fuzzySearch(currentInput)
		},
	)
	finder.currentFocusedWidget = finder.footer
	finder.Draw()
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

	// Clear and rebuild the list with final matches
	finder.searchedList.Clear()
	line := ""
	for _, match := range matches {
		for i := 0; i < len(match.Str); i++ {
			if helper.Contains(i, allMatchedIndexes[match.Str]) {
				line = line + "[red::b]" + string(match.Str[i]) + "[-::-]"
			} else {
				line = line + string(match.Str[i])
			}
		}
		finder.searchedList.AddItem(line, match.Str, 0, nil)
		line = ""
	}

	if len(matches) > 0 {
		finder.setCurrentLine(0)
	}

	finder.Draw()
}

func (finder *Finder) resetFileList() error {
	finder.fileList.Clear()
	fileList, err := helper.LoadDirectory(finder.context.CurrentPath, finder.showHiddenFiles, true)
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
	finder.rootFlex.AddItem(listFlex, 0, 1, true)
	if finder.footer != nil {
		finder.rootFlex.AddItem(finder.footer, 1, 0, false)
	}
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
	finder.fileList.
		SetMainTextColor(explorerTheme.Fg1).
		SetSelectedTextColor(explorerTheme.Black).
		SetSelectedBackgroundColor(explorerTheme.Aqua).
		SetBackgroundColor(explorerTheme.Bg0)

	// Style the footer
	if finder.footer != nil {
		finder.footer.
			SetFieldBackgroundColor(explorerTheme.Bg1).
			SetFieldTextColor(explorerTheme.Fg0).
			SetBackgroundColor(explorerTheme.Bg0)
	}
}
