package finder

import (
	"bufio"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strings"
	"sync"

	"github.com/thilobro/gofileyourself/internal/helper"
	"github.com/thilobro/gofileyourself/internal/widget"

	gostring "github.com/boyter/go-string"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

type Finder struct {
	context              *widget.Context
	rootFlex             *tview.Flex
	footer               *tview.InputField
	fileList             *tview.List
	searchedList         *tview.List
	previousSearchedList *tview.List
	selectedList         tview.Primitive
	currentFocusedWidget tview.Primitive
	searchTerm           string
	fuzzySearchQuit      chan bool
	listUpdateChan       chan *tview.List // Channel for list updates
	listUpdateMutex      sync.Mutex
	title                string
}

func (finder *Finder) setDrawFunc() {
	finder.searchedList.SetDrawFunc(func(screen tcell.Screen, x int, y int, width int, height int) (int, int, int, int) {
		finder.listUpdateMutex.Lock()
		helper.CopyListContent(finder.previousSearchedList, finder.searchedList)
		finder.RemoveContentFromDisplayName()
		helper.ShortenPathsIfNecessary(finder.searchedList, width)
		finder.listUpdateMutex.Unlock()
		return x, y, width, height
	})
}

func NewFinder(context *widget.Context) (*Finder, error) {
	finder := &Finder{
		context:              context,
		rootFlex:             tview.NewFlex(),
		footer:               tview.NewInputField(),
		fileList:             tview.NewList(),
		selectedList:         tview.NewList().ShowSecondaryText(false),
		searchedList:         tview.NewList().ShowSecondaryText(false),
		previousSearchedList: tview.NewList().ShowSecondaryText(false),
		searchTerm:           "",
		fuzzySearchQuit:      make(chan bool),
		listUpdateChan:       make(chan *tview.List, 10), // Buffered channel
		title:                "Find",
	}
	finder.resetFileList(false)
	finder.SetupKeyBindings()
	finder.currentFocusedWidget = finder.searchedList
	finder.setDrawFunc()

	// Start list update handler
	go finder.handleListUpdates()

	err := finder.searchInDirectory()
	if err != nil {
		return nil, err
	}

	return finder, nil
}

func (finder *Finder) handleListUpdates() {
	for newList := range finder.listUpdateChan {
		finder.listUpdateMutex.Lock()
		helper.CopyListContent(newList, finder.searchedList)
		helper.CopyListContent(finder.searchedList, finder.previousSearchedList)
		finder.listUpdateMutex.Unlock()
		finder.context.App.QueueUpdateDraw(func() {
			finder.setCurrentLine(0)
			finder.Draw()
		})
	}
}

func (finder *Finder) setCurrentLine(lineIndex int) error {
	if lineIndex < 0 {
		lineIndex = 0
	}
	if lineIndex >= finder.searchedList.GetItemCount() {
		lineIndex = finder.searchedList.GetItemCount() - 1
	}
	finder.searchedList.SetCurrentItem(lineIndex)

	_, selectedName := finder.searchedList.GetItemText(lineIndex)
	return finder.setSelectedDirectory(helper.GetAbsFilePath(selectedName, finder.context.CurrentPath))
}

// setSelectedDirectory updates the selected directory/file preview
func (finder *Finder) setSelectedDirectory(selectedPath string) error {
	selectedAbsolutePath, _ := filepath.Abs(selectedPath)
	isFileNotFound := helper.IsFileNotFound(selectedAbsolutePath)
	if isFileNotFound {
		finder.selectedList = tview.NewTextArea().SetText("Not found...", false)
		return nil
	}
	isDirEmpty, _ := helper.IsDirectoryEmpty(selectedAbsolutePath)
	if isDirEmpty {
		finder.selectedList = tview.NewTextArea().SetText("Directory is empty...", false)
		return nil
	}
	selectedDirectoryIndex := 0

	newSelectedList, err := helper.LoadDirectory(selectedPath, finder.context.ShowHiddenFiles, false, false, []string{})
	if err != nil {
		return err
	}

	if newSelectedList == nil {
		finder.selectedList, err = helper.LoadFilePreview(selectedPath, &finder.searchTerm)
		if err != nil {
			return err
		}
	} else {
		newSelectedList.SetCurrentItem(selectedDirectoryIndex)
		finder.selectedList = newSelectedList
	}
	return nil
}

func (finder *Finder) handleFooterInput() {
	finder.footer = tview.NewInputField().SetText("/")
	finder.footer.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		return event
	})
	helper.CopyListContent(finder.fileList, finder.searchedList)
	finder.footer.SetChangedFunc(
		func(text string) {
			defer finder.Draw()
			currentInput := strings.TrimPrefix(text, "/")
			go finder.manageFuzzySearch(currentInput)
		},
	)
	finder.currentFocusedWidget = finder.footer
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
	// Create new list with matches
	text, _ = strings.CutSuffix(text, " ")
	searchTerms := strings.FieldsFunc(text, func(r rune) bool { return r == ' ' || r == '/' })
	startHighlightMarker := "[red::b]"
	endHighlightMarker := "[-::-]"
	newList := tview.NewList().ShowSecondaryText(false)
	itemCount := finder.fileList.GetItemCount()
	searchHits := slices.Repeat([]int{0}, itemCount)
	lineIdxs := make([]int, itemCount)
	for i := 0; i < itemCount; i++ {
		lineIdxs[i] = i
	}
	indeces := make(map[int][][]int, itemCount)
	for _, searchTerm := range searchTerms {
		for i := 0; i < itemCount; i++ {
			primaryText, _ := finder.fileList.GetItemText(i)
			idxs := gostring.IndexAllIgnoreCase(primaryText, searchTerm, -1)
			if val, exists := indeces[i]; !exists {
				indeces[i] = idxs
			} else {
				indeces[i] = append(val, idxs...)
			}
			for _, idx := range idxs {
				searchHits[i] = searchHits[i] + len(idx)
			}
		}
	}

	sort.SliceStable(lineIdxs, func(i, j int) bool {
		return searchHits[lineIdxs[i]] > searchHits[lineIdxs[j]]
	})
	for j := 0; j < itemCount; j++ {
		displayName, secondaryText := finder.fileList.GetItemText(lineIdxs[j])
		index := indeces[lineIdxs[j]]
		newList.AddItem(gostring.HighlightString(displayName, index, startHighlightMarker, endHighlightMarker), secondaryText, 0, nil)
	}
	// Send the new list through the channel
	select {
	case finder.listUpdateChan <- newList:
	default:
		// If channel is full, skip this update
	}
	finder.searchTerm = text
}

func (finder *Finder) RemoveContentFromDisplayName() {
	for i := 0; i < finder.searchedList.GetItemCount(); i++ {
		displayName, secondaryText := finder.searchedList.GetItemText(i)
		displayName = strings.Split(displayName, " >>> ")[0]
		if i >= finder.searchedList.GetItemCount() {
			return
		}
		finder.searchedList.SetItemText(i, displayName, secondaryText)
	}
}

func (finder *Finder) resetFileList(showContent bool) error {
	finder.fileList.Clear()
	fileList, err := helper.LoadDirectory(finder.context.CurrentPath, finder.context.ShowHiddenFiles, showContent, true, []string{})
	if err != nil {
		return err
	}
	finder.fileList = fileList
	helper.CopyListContent(finder.fileList, finder.searchedList)
	helper.CopyListContent(finder.searchedList, finder.previousSearchedList)
	return nil
}

func (finder *Finder) searchInDirectory() error {
	finder.setCurrentLine(0)
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
		listFlex.AddItem(tview.NewBox(), 2, 0, false)
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

func (finder *Finder) GetInputCapture() func(*tcell.EventKey) *tcell.EventKey {
	return finder.rootFlex.GetInputCapture()
}

func (finder *Finder) showRecentHistory() {
	finder.title = "Find Recent"
	finder.fileList.Clear()
	historyPath := filepath.Join(os.Getenv("HOME"), ".gofileyourselfhistory")
	historyFile, err := os.Open(historyPath)
	if err != nil {
		return
	}
	defer historyFile.Close()
	scanner := bufio.NewScanner(historyFile)
	for scanner.Scan() {
		line := scanner.Text()
		finder.fileList.InsertItem(0, line, line, 0, nil)
	}
	helper.CopyListContent(finder.fileList, finder.searchedList)
	helper.CopyListContent(finder.searchedList, finder.previousSearchedList)
	finder.searchInDirectory()
}

func (finder *Finder) showGrep() {
	finder.title = "Grep"
	finder.fileList.Clear()
	finder.resetFileList(true)
	helper.CopyListContent(finder.fileList, finder.searchedList)
	helper.CopyListContent(finder.searchedList, finder.previousSearchedList)
	finder.searchInDirectory()
}
