package finder

import (
	"os"

	"github.com/gdamore/tcell/v2"
	"github.com/thilobro/gofileyourself/internal/helper"
	"github.com/thilobro/gofileyourself/internal/widget"
)

func (finder *Finder) handleKeys(event *tcell.EventKey) *tcell.EventKey {
	switch event.Key() {
	case tcell.KeyCtrlR:
		finder.showRecentHistory()
		return nil
	case tcell.KeyCtrlG:
		finder.showGrep()
		return nil
	case tcell.KeyCtrlH:
		finder.context.ShowHiddenFiles = !finder.context.ShowHiddenFiles

		// Remember current selection before refresh
		_, currentName := finder.searchedList.GetItemText(finder.searchedList.GetCurrentItem())
		finder.resetFileList(false)
		helper.CopyListContent(finder.fileList, finder.searchedList)

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
		filePath := helper.GetAbsFilePath(fileName, finder.context.CurrentPath)

		fileInfo, err := os.Stat(filePath)
		if err != nil {
			return event
		}

		if fileInfo.IsDir() {
			finder.context.CurrentPath = filePath
			finder.context.OnWidgetResult(widget.Find, filePath)
			return nil
		}
		helper.OpenInNvim(filePath, finder.context.ChooseFilePath, finder.context.App, finder.context.Config.HistoryLen)
		return nil
	}
	return event
}

func (finder *Finder) SetupKeyBindings() {
	finder.rootFlex.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		defer finder.Draw()
		if event = finder.handleKeys(event); event == nil {
			return nil
		}
		finder.footer.GetInputCapture()(event)
		return nil
	})
}
