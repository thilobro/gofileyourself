package finder

import (
	"gofileyourself/internal/helper"
	"gofileyourself/internal/theme"
	"gofileyourself/internal/widget"
	"sort"
	"strings"

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
	currentFocusedWidget tview.Primitive
	showHiddenFiles      bool
}

func NewFinder(context *widget.Context) (*Finder, error) {
	finder := &Finder{
		context:         context,
		rootFlex:        tview.NewFlex(),
		footer:          tview.NewInputField(),
		fileList:        tview.NewList(),
		showHiddenFiles: false,
	}
	finder.currentFocusedWidget = finder.fileList

	err := finder.searchInDirectory(finder.context.CurrentPath)
	if err != nil {
		return nil, err
	}

	return finder, nil
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

func (finder *Finder) searchInDirectory(path string) error {
	finder.resetFileList()
	finder.Draw()
	finder.handleFooterInput()
	return nil
}

func (finder *Finder) Root() tview.Primitive {
	return finder.rootFlex
}

func (finder *Finder) Draw() {
	finder.rootFlex.Clear()
	finder.rootFlex.AddItem(finder.fileList, 0, 1, true)
	finder.rootFlex.SetDirection(tview.FlexRow)
	if finder.footer != nil {
		finder.rootFlex.AddItem(finder.footer, 1, 0, false)
	}
	finder.context.App.SetFocus(finder.currentFocusedWidget)
	finder.applyTheme()
}

func (finder *Finder) Run() error {
	return finder.context.App.SetRoot(finder.Root(), true).Run()
}

func (finder *Finder) SetupKeyBindings() {
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
