package main

import (
	"log"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

// FileExplorer represents the state and behavior of the file explorer
type FileExplorer struct {
	app                 *tview.Application
	currentPath         string
	currentList         *tview.List
	parentList          tview.Primitive
	selectedList        tview.Primitive
	rootFlex            *tview.Flex
	listFlex            *tview.Flex
	directoryToIndexMap map[string]int
	footer              *tview.InputField
}

// NewFileExplorer creates and initializes a new FileExplorer
func NewFileExplorer() (*FileExplorer, error) {
	currentPath, err := os.Getwd()
	if err != nil {
		return nil, err
	}

	fe := &FileExplorer{
		app:                 tview.NewApplication(),
		currentPath:         currentPath,
		directoryToIndexMap: make(map[string]int),
		listFlex:            tview.NewFlex(),
		rootFlex:            tview.NewFlex(),
	}

	if err := fe.initialize(); err != nil {
		return nil, err
	}

	return fe, nil
}

// findExactItem is a helper function that searches for an item in a list
func findExactItem(list *tview.List, searchTerm string) int {
	matchingIndeces := list.FindItems(searchTerm, "", false, true)
	if len(matchingIndeces) == 1 {
		return matchingIndeces[0]
	}
	for _, index := range matchingIndeces {
		if text, _ := list.GetItemText(index); text == searchTerm {
			return index
		}
	}
	return 0
}

// loadDirectory is a helper function that loads directory contents into a list
func loadDirectory(path string) (*tview.List, error) {
	fileInfo, err := os.Stat(path)
	if err != nil {
		return nil, err
	}
	if fileInfo.IsDir() {
		files, err := os.ReadDir(path)
		if err != nil {
			return nil, err
		}
		list := tview.NewList().ShowSecondaryText(false)
		for _, file := range files {
			list.AddItem(file.Name(), "", 0, nil)
		}
		return list, nil
	}
	return nil, nil
}

// loadFilePreview is a helper function that creates a text view for file contents
func loadFilePreview(path string) (*tview.TextView, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	textView := tview.NewTextView().
		SetDynamicColors(true).
		SetRegions(true).
		SetWordWrap(true).
		SetText(string(content))

	return textView, nil
}

// openInNvim is a helper function that opens a file in neovim
func openInNvim(path string, app *tview.Application) error {
	app.Suspend(func() {
		cmd := exec.Command("nvim", path)
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		cmd.Run()
	})
	return nil
}

// initialize sets up the initial state of the FileExplorer
func (fe *FileExplorer) initialize() error {
	var err error
	fe.currentList, err = loadDirectory(fe.currentPath)
	if err != nil {
		return err
	}

	parentPath := filepath.Join(fe.currentPath, "..")
	newParentList, err := loadDirectory(parentPath)
	if err != nil {
		return err
	}
	parentDirectoryIndex := findExactItem(newParentList, filepath.Base(fe.currentPath))
	newParentList.SetCurrentItem(parentDirectoryIndex)
	fe.parentList = newParentList
	parentAbsolutePath, _ := filepath.Abs(parentPath)
	fe.directoryToIndexMap[parentAbsolutePath] = parentDirectoryIndex

	selectedName, _ := fe.currentList.GetItemText(0)
	selectedPath := filepath.Join(fe.currentPath, selectedName)
	fe.selectedList, err = loadDirectory(selectedPath)
	if err != nil {
		return err
	}

	fe.setupKeyBindings()
	fe.draw()
	return nil
}

// draw updates the UI
func (fe *FileExplorer) draw() {
	fe.listFlex.Clear()
	if fe.parentList != nil {
		fe.listFlex.AddItem(fe.parentList, 0, 1, false)
	}
	if fe.currentList != nil {
		fe.listFlex.AddItem(fe.currentList, 0, 2, true)
	}
	if fe.selectedList != nil {
		fe.listFlex.AddItem(fe.selectedList, 0, 2, false)
	}
	fe.rootFlex.Clear()
	fe.rootFlex.SetDirection(tview.FlexRow)
	fe.rootFlex.AddItem(fe.listFlex, 0, 1, true)
	if fe.footer != nil {
		fe.rootFlex.AddItem(fe.footer, 1, 0, false)
	}
	fe.app.SetRoot(fe.rootFlex, true)
}

// updateSelectedDirectory updates the selected directory/file preview
func (fe *FileExplorer) updateSelectedDirectory(selectedPath string) error {
	selectedAbsolutePath, _ := filepath.Abs(selectedPath)
	selectedDirectoryIndex := fe.directoryToIndexMap[selectedAbsolutePath]

	newSelectedList, err := loadDirectory(selectedPath)
	if err != nil {
		return err
	}

	if newSelectedList == nil {
		fe.selectedList, err = loadFilePreview(selectedPath)
		if err != nil {
			return err
		}
	} else {
		newSelectedList.SetCurrentItem(selectedDirectoryIndex)
		fe.selectedList = newSelectedList
	}

	fe.draw()
	return nil
}

// changeCurrentDirectory changes the current directory and updates related views
func (fe *FileExplorer) changeCurrentDirectory(path string) error {
	// Update current directory
	currentAbsolutePath, _ := filepath.Abs(path)
	currentDirectoryIndex := fe.directoryToIndexMap[currentAbsolutePath]
	newCurrentList, err := loadDirectory(path)
	if err != nil {
		return err
	}

	newCurrentList.SetInputCapture(fe.currentList.GetInputCapture())
	newCurrentList.SetCurrentItem(currentDirectoryIndex)
	fe.currentList = newCurrentList

	// Update parent directory
	if currentAbsolutePath == "/" {
		emptyList := tview.NewList().ShowSecondaryText(false)
		fe.parentList = emptyList
	} else {
		parentPath := filepath.Join(path, "..")
		newParentList, err := loadDirectory(parentPath)
		if err != nil {
			return err
		}

		parentDirectoryIndex := findExactItem(newParentList, filepath.Base(path))

		parentAbsolutePath, _ := filepath.Abs(parentPath)
		fe.directoryToIndexMap[parentAbsolutePath] = parentDirectoryIndex
		newParentList.SetCurrentItem(parentDirectoryIndex)
		fe.parentList = newParentList
	}

	// Update selected directory
	selectedName, _ := fe.currentList.GetItemText(currentDirectoryIndex)
	selectedPath := filepath.Join(path, selectedName)
	if err := fe.updateSelectedDirectory(selectedPath); err != nil {
		return err
	}

	fe.currentPath = path
	return nil
}

// updateCurrentLine updates the current line selection
func (fe *FileExplorer) updateCurrentLine(lineIndex int) error {
	if lineIndex < 0 || lineIndex >= fe.currentList.GetItemCount() {
		return nil
	}
	fe.currentList.SetCurrentItem(lineIndex)
	currentAbsolutePath, _ := filepath.Abs(fe.currentPath)
	fe.directoryToIndexMap[currentAbsolutePath] = lineIndex

	selectedName, _ := fe.currentList.GetItemText(lineIndex)
	return fe.updateSelectedDirectory(filepath.Join(fe.currentPath, selectedName))
}

// setupKeyBindings configures keyboard input handling
func (fe *FileExplorer) setupKeyBindings() {
	fe.currentList.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Rune() {
		case 'j': // scroll down
			fe.updateCurrentLine(fe.currentList.GetCurrentItem() + 1)
			return nil
		case 'k': // scroll up
			fe.updateCurrentLine(fe.currentList.GetCurrentItem() - 1)
			return nil
		case 'q': // quit
			fe.app.Stop()
			return nil
		case 'l': // open dir or file
			currentItem := fe.currentList.GetCurrentItem()
			fileName, _ := fe.currentList.GetItemText(currentItem)
			filePath := filepath.Join(fe.currentPath, fileName)
			fileInfo, err := os.Stat(filePath)
			if err != nil {
				return event
			}
			if fileInfo.IsDir() {
				if err := fe.changeCurrentDirectory(filePath); err != nil {
					return event
				}
			} else {
				openInNvim(filePath, fe.app)
				return nil
			}
			return nil
		case 'h': // go up directory
			dirPath := filepath.Join(fe.currentPath, "..")
			if err := fe.changeCurrentDirectory(dirPath); err != nil {
				return event
			}
			return nil
		case '/': // search
			searchInput := tview.NewInputField().SetText("/")
			fe.footer = searchInput
			searchInput.SetDoneFunc(
				func(key tcell.Key) {
					if key == tcell.KeyEnter {
						searchTerm := searchInput.GetText()[1:]
						matchingIndeces := fe.currentList.FindItems(searchTerm, "", false, true)
						if len(matchingIndeces) > 0 {
							fe.updateCurrentLine(matchingIndeces[0])
						}
						fe.app.SetFocus(fe.currentList)
					}
				},
			)
			fe.draw()
			fe.app.SetFocus(searchInput)
			return nil
		}
		return event
	})
}

// Run starts the file explorer
func (fe *FileExplorer) Run() error {
	return fe.app.SetRoot(fe.rootFlex, true).Run()
}

func main() {
	logFile, err := os.OpenFile("debug.log", os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		log.Fatalf("Error opening log file: %v", err)
	}

	// Configure log package to write to file
	log.SetOutput(logFile)
	// Optional: include file and line number in logs
	log.SetFlags(log.Ltime | log.Lshortfile)
	fe, err := NewFileExplorer()
	if err != nil {
		panic(err)
	}

	if err := fe.Run(); err != nil {
		panic(err)
	}
}
