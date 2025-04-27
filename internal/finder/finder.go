package finder

import (
	"gofileyourself/internal/helper"
	"gofileyourself/internal/theme"
	"gofileyourself/internal/widget"
	"path/filepath"
	"sort"
	"strings"

	"github.com/gdamore/tcell/v2"
	"github.com/lithammer/fuzzysearch/fuzzy"
	"github.com/rivo/tview"
)

type rankedItem struct {
	index       int
	rank        int
	displayText string
	text        string
}

type Finder struct {
	context              *widget.Context
	rootFlex             *tview.Flex
	footer               *tview.InputField
	fileList             *tview.List
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
		selectedList:    tview.NewList(),
		showHiddenFiles: false,
	}
	finder.SetupKeyBindings()
	finder.currentFocusedWidget = finder.fileList

	err := finder.searchInDirectory()
	if err != nil {
		return nil, err
	}

	return finder, nil
}

func (finder *Finder) setCurrentLine(lineIndex int) error {
	if lineIndex < 0 || lineIndex >= finder.fileList.GetItemCount() {
		return nil
	}
	finder.fileList.SetCurrentItem(lineIndex)

	_, selectedName := finder.fileList.GetItemText(lineIndex)
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
			finder.setCurrentLine(finder.fileList.GetCurrentItem() - 1)
			return nil
		case tcell.KeyDown:
			finder.setCurrentLine(finder.fileList.GetCurrentItem() + 1)
			return nil
		case tcell.KeyEnter:
			currentItem := finder.fileList.GetCurrentItem()
			_, fileName := finder.fileList.GetItemText(currentItem)
			filePath := filepath.Join(finder.context.CurrentPath, fileName)
			helper.OpenInNvim(filePath, finder.context.App)
			return nil
		}
		return event
	})
}

func (finder *Finder) handleFooterInput() {
	finder.footer = tview.NewInputField().SetText("/")
	finder.footer.SetChangedFunc(
		func(text string) {
			currentInput := strings.TrimPrefix(text, "/")
			finder.resetFileList()
			finder.fuzzySearch(currentInput)
			finder.Draw()
		},
	)
	finder.currentFocusedWidget = finder.footer
	finder.Draw()
}

func (finder *Finder) fuzzySearch(text string) {
	items := make([]rankedItem, finder.fileList.GetItemCount())

	// Collect all items with their ranks
	for i := 0; i < finder.fileList.GetItemCount(); i++ {
		itemDisplayName, itemName := finder.fileList.GetItemText(i)
		rank := fuzzy.RankMatch(text, itemName)
		items[i] = rankedItem{
			index:       i,
			rank:        rank,
			text:        itemName,
			displayText: itemDisplayName,
		}
	}

	// Sort items by rank (higher rank = better match)
	sort.Slice(items, func(i, j int) bool {
		return items[i].rank > items[j].rank
	})

	// Clear and rebuild the list in sorted order
	finder.fileList.Clear()
	for _, item := range items {
		if item.rank > -1 { // Only show matching items
			finder.fileList.AddItem(item.displayText, item.text, 0, nil)
		}
	}
	finder.setCurrentLine(0)

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
	listFlex.AddItem(finder.fileList, 0, 1, true)
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
