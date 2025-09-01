package explorer

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"github.com/thilobro/gofileyourself/internal/helper"
)

func (explorer *Explorer) handleKeyCombinations(event *tcell.EventKey) *tcell.EventKey {
	switch event.Key() {
	case tcell.KeyCtrlS:
		explorer.context.ShowHiddenFiles = !explorer.context.ShowHiddenFiles

		// Remember current selection before refresh
		_, currentName := explorer.currentList.GetItemText(explorer.currentList.GetCurrentItem())

		// Remember selected directory name if we're showing a directory
		var selectedName string
		if list, ok := explorer.selectedList.(*tview.List); ok {
			_, selectedName = list.GetItemText(list.GetCurrentItem())
		}

		// Refresh the view
		if err := explorer.setCurrentDirectory(explorer.context.CurrentPath); err != nil {
			return event
		}

		// Restore current selection
		if idx := helper.FindExactItem(explorer.currentList, currentName); idx >= 0 {
			explorer.setCurrentLine(idx)
		}

		// Restore selected directory selection if applicable
		if list, ok := explorer.selectedList.(*tview.List); ok {
			if idx := helper.FindExactItem(list, selectedName); idx >= 0 {
				list.SetCurrentItem(idx)
				absoluteSelectedPath, _ := filepath.Abs(filepath.Join(explorer.context.CurrentPath, currentName))
				explorer.directoryToIndexMap[absoluteSelectedPath] = idx
			}
		}
		return nil
	case tcell.KeyCtrlD: // scroll 10 down
		scrollAmount := explorer.currentList.GetItemCount() / 2
		if scrollAmount > constMaxScrollAmount {
			scrollAmount = constMaxScrollAmount
		}
		explorer.setCurrentLine(explorer.currentList.GetCurrentItem() + scrollAmount)
		return nil
	case tcell.KeyCtrlU: // scroll 10 up
		scrollAmount := explorer.currentList.GetItemCount() / 2
		if scrollAmount > constMaxScrollAmount {
			scrollAmount = constMaxScrollAmount
		}
		explorer.setCurrentLine(explorer.currentList.GetCurrentItem() - scrollAmount)
		return nil
	}
	return event
}

func (explorer *Explorer) handleMultipleKey(event *tcell.EventKey) *tcell.EventKey {
	rune := event.Rune()
	explorer.keyBuffer += string(rune)
	if len(explorer.keyBuffer) > 5 {
		explorer.keyBuffer = explorer.keyBuffer[4:]
	}
	if strings.HasSuffix(explorer.keyBuffer, "gg") {
		explorer.keyBuffer = ""
		explorer.setCurrentLine(0)
		return nil
	} else if strings.HasSuffix(explorer.keyBuffer, "dd") {
		explorer.keyBuffer = ""
		explorer.deleteCurrentFile(false)
		return nil
	} else if strings.HasSuffix(explorer.keyBuffer, "DD") {
		explorer.keyBuffer = ""
		explorer.deleteCurrentFile(true)
		return nil
	} else if strings.HasSuffix(explorer.keyBuffer, "yy") {
		explorer.keyBuffer = ""
		explorer.yankCurrentFile()
		return nil
	} else if strings.HasSuffix(explorer.keyBuffer, "pp") {
		explorer.keyBuffer = ""
		explorer.pasteYankedFile()
		return nil
	} else if strings.HasSuffix(explorer.keyBuffer, "mm") {
		explorer.keyBuffer = ""
		explorer.toggleMarkForCurrentFile()
		return nil
	} else if strings.HasSuffix(explorer.keyBuffer, "mu") {
		explorer.keyBuffer = ""
		explorer.unmarkAllFiles()
		return nil
	} else if strings.HasSuffix(explorer.keyBuffer, "md") {
		explorer.keyBuffer = ""
		explorer.deleteMarkedFiles(false)
		return nil
	} else if strings.HasSuffix(explorer.keyBuffer, "mD") {
		explorer.keyBuffer = ""
		explorer.deleteMarkedFiles(true)
		return nil
	} else if strings.HasSuffix(explorer.keyBuffer, "my") {
		explorer.keyBuffer = ""
		explorer.yankMarkedFiles()
		return nil
	} else if strings.HasSuffix(explorer.keyBuffer, "mp") {
		explorer.keyBuffer = ""
		explorer.pasteMarkedFiles()
		return nil
	} else if match := regexp.MustCompile(`A([a-zA-Z0-9]+)$`).FindStringSubmatch(explorer.keyBuffer); match != nil {
		key := match[1]
		explorer.keyBuffer = ""
		explorer.setAnchor(key)
		return nil
	} else if match := regexp.MustCompile(`a([a-zA-Z0-9]+)$`).FindStringSubmatch(explorer.keyBuffer); match != nil {
		key := match[1]
		explorer.keyBuffer = ""
		explorer.jumpToAnchor(key)
		return nil
	}
	return event
}

func (explorer *Explorer) handleSingleKey(event *tcell.EventKey) *tcell.EventKey {
	rune := event.Rune()
	switch rune {
	case 'r':
		explorer.cycleRecentlyVisited(false)
		return nil
	case 'R':
		explorer.cycleRecentlyVisited(true)
		return nil
	case 'M':
		explorer.toggleMarkForCurrentFile()
		return nil
	case 'G':
		explorer.setCurrentLine(explorer.currentList.GetItemCount() - 1)
		return nil
	case 'S':
		explorer.quitAndChangeDirectory()
		return nil
	case 'j': // scroll down
		explorer.setCurrentLine(explorer.currentList.GetCurrentItem() + 1)
		return nil
	case 'k': // scroll up
		explorer.setCurrentLine(explorer.currentList.GetCurrentItem() - 1)
		return nil
	case 'q': // quit
		explorer.context.App.Stop()
		return nil
	case 'l': // open dir or file
		currentItem := explorer.currentList.GetCurrentItem()
		_, fileName := explorer.currentList.GetItemText(currentItem)
		filePath := filepath.Join(explorer.context.CurrentPath, fileName)
		fileInfo, err := os.Stat(filePath)
		if err != nil {
			return event
		}
		if fileInfo.IsDir() {
			if err := explorer.setCurrentDirectory(filePath); err != nil {
				return event
			}
		} else {
			helper.OpenInNvim(filePath, explorer.context.ChooseFilePath, explorer.context.App, explorer.context.Config.HistoryLen)
			return nil
		}
		return nil
	case 'h': // go up directory
		dirPath := filepath.Join(explorer.context.CurrentPath, "..")
		if err := explorer.setCurrentDirectory(dirPath); err != nil {
			return event
		}
		return nil
	case '/': // search
		explorer.handleFooterInput("/")
		return nil
	case ':': // command
		explorer.handleFooterInput(":")
		return nil
	case 'n': // cycle search
		if len(explorer.currentSearchIndeces) > 0 {
			currentIndex := explorer.currentList.GetCurrentItem()
			for _, index := range explorer.currentSearchIndeces {
				if index > currentIndex {
					explorer.setCurrentLine(index)
					return nil
				}
			}
			explorer.setCurrentLine(explorer.currentSearchIndeces[0])
		}
	case 'N': // cycle search backwards
		if len(explorer.currentSearchIndeces) > 0 {
			currentIndex := explorer.currentList.GetCurrentItem()
			for i := len(explorer.currentSearchIndeces) - 1; i >= 0; i-- {
				if explorer.currentSearchIndeces[i] < currentIndex {
					explorer.setCurrentLine(explorer.currentSearchIndeces[i])
					return nil
				}
			}
			// If no smaller index found, wrap around to the last item
			explorer.setCurrentLine(explorer.currentSearchIndeces[len(explorer.currentSearchIndeces)-1])
		}

		return nil
	}
	return event
}

func (explorer *Explorer) SetupKeyBindings() {
	explorer.currentList.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		defer explorer.Draw()
		if event = explorer.handleSingleKey(event); event == nil {
			return nil
		}
		if event = explorer.handleKeyCombinations(event); event == nil {
			return nil
		}
		if event = explorer.handleMultipleKey(event); event == nil {
			return nil
		}
		return nil
	})
}
