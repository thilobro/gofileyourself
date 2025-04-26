package finder

import (
	"gofileyourself/internal/helper"
	"gofileyourself/internal/widget"
	"strings"

	"github.com/rivo/tview"
)

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
			text = strings.TrimPrefix(text, "/")
			finder.fuzzySearch(text)
			finder.Draw()
		},
	)
	finder.currentFocusedWidget = finder.footer
	finder.Draw()
}

func (finder *Finder) fuzzySearch(text string) {
	for i := 0; i < finder.fileList.GetItemCount(); i++ {
		_, itemName := finder.fileList.GetItemText(i)
		if itemName == text {
			finder.fileList.SetCurrentItem(i)
		}
	}
	finder.Draw()
}

func (finder *Finder) searchInDirectory(path string) error {
	fileList, err := helper.LoadDirectory(path, finder.showHiddenFiles, true)
	if err != nil {
		return err
	}
	finder.fileList = fileList
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
}

func (finder *Finder) Run() error {
	return finder.context.App.SetRoot(finder.Root(), true).Run()
}

func (finder *Finder) SetupKeyBindings() {
}
