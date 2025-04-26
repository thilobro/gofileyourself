package explorer

import (
	"gofileyourself/internal/formatter"
	"gofileyourself/internal/theme"
	"gofileyourself/internal/widget"
	"os"
	"path/filepath"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
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
	header               *tview.TextView
	showHiddenFiles      bool
	currentSearchTerm    string
	currentSearchIndeces []int
	currentFocusedWidget tview.Primitive
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

	// Style the header
	if fe.header != nil {
		fe.header.
			SetBackgroundColor(explorerTheme.Blue)
		fe.header.SetTextColor(explorerTheme.Bg0)
	}
}

func (fe *FileExplorer) UpdateCurrentPath(path string) error {
	return fe.setCurrentDirectory(path)
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
		header:              tview.NewTextView(),
		showHiddenFiles:     false,
		currentSearchTerm:   "",
	}

	if err := fe.initialize(); err != nil {
		return nil, err
	}

	return fe, nil
}

// initialize sets up the initial state of the FileExplorer
func (fe *FileExplorer) initialize() error {
	fe.SetupKeyBindings()
	fe.setCurrentDirectory(".")
	fe.currentFocusedWidget = fe.currentList
	fe.Draw()
	return nil
}

// draw updates the UI
func (fe *FileExplorer) Draw() {
	fe.listFlex.Clear()
	if fe.parentList != nil {
		fe.listFlex.AddItem(fe.parentList, 0, 1, false)
	}
	if fe.currentList != nil {
		fe.listFlex.AddItem(fe.currentList, 0, 2, true)
	}
	if fe.selectedList != nil {
		fe.listFlex.AddItem(fe.selectedList, 0, 3, false)
	}
	fe.rootFlex.Clear()
	fe.rootFlex.SetDirection(tview.FlexRow)
	if fe.header != nil {
		fe.rootFlex.AddItem(fe.header, 1, 0, false)
	}
	fe.rootFlex.AddItem(fe.listFlex, 0, 1, true)
	if fe.footer != nil {
		fe.rootFlex.AddItem(fe.footer, 1, 0, false)
	}
	fe.context.App.SetRoot(fe.rootFlex, true)
	fe.context.App.SetFocus(fe.currentFocusedWidget)
	fe.applyTheme()
}

// setSelectedDirectory updates the selected directory/file preview
func (fe *FileExplorer) setSelectedDirectory(selectedPath string) error {
	selectedAbsolutePath, _ := filepath.Abs(selectedPath)
	selectedDirectoryIndex := fe.directoryToIndexMap[selectedAbsolutePath]

	newSelectedList, err := loadDirectory(selectedPath, fe.showHiddenFiles)
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
	return nil
}

func (fe *FileExplorer) setParentDirectory(path string) error {
	currentAbsolutePath, _ := filepath.Abs(path)
	if currentAbsolutePath == "/" {
		emptyList := tview.NewList().ShowSecondaryText(false)
		fe.parentList = emptyList
	} else {
		parentPath := filepath.Join(currentAbsolutePath, "..")
		newParentList, err := loadDirectory(parentPath, fe.showHiddenFiles)
		if err != nil {
			return err
		}

		parentDirectoryIndex := findExactItem(newParentList, filepath.Base(currentAbsolutePath))

		parentAbsolutePath, _ := filepath.Abs(parentPath)
		fe.directoryToIndexMap[parentAbsolutePath] = parentDirectoryIndex
		newParentList.SetCurrentItem(parentDirectoryIndex)
		fe.parentList = newParentList
	}
	return nil
}

// setCurrentDirectory changes the current directory and updates related views
func (fe *FileExplorer) setCurrentDirectory(path string) error {
	// Update current directory
	currentAbsolutePath, _ := filepath.Abs(path)
	currentDirectoryIndex := fe.directoryToIndexMap[currentAbsolutePath]
	newCurrentList, err := loadDirectory(currentAbsolutePath, fe.showHiddenFiles)
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
	fe.header.SetText(text)
}

// setCurrentLine updates the current line selection
func (fe *FileExplorer) setCurrentLine(lineIndex int) error {
	if lineIndex < 0 || lineIndex >= fe.currentList.GetItemCount() {
		return nil
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
		switch command {
		case "q":
			fe.context.App.Stop()
		}
	}
	fe.currentFocusedWidget = fe.currentList
}

func (fe *FileExplorer) handleFooterInput(prompt string) {
	fe.footer = tview.NewInputField().SetText(prompt)
	fe.footer.SetDoneFunc(
		func(key tcell.Key) {
			if key == tcell.KeyEnter {
				inputText := fe.footer.GetText()
				fe.runFooterCommand(inputText)
				fe.currentFocusedWidget = fe.currentList
			}
			fe.Draw()
		},
	)
	fe.currentFocusedWidget = fe.footer
	fe.Draw()
}

// setupKeyBindings configures keyboard input handling
func (fe *FileExplorer) SetupKeyBindings() {
	fe.currentList.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		defer fe.Draw()
		switch event.Key() {
		case tcell.KeyCtrlH:
			fe.showHiddenFiles = !fe.showHiddenFiles

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
			if idx := findExactItem(fe.currentList, currentName); idx >= 0 {
				fe.setCurrentLine(idx)
			}

			// Restore selected directory selection if applicable
			if list, ok := fe.selectedList.(*tview.List); ok {
				if idx := findExactItem(list, selectedName); idx >= 0 {
					list.SetCurrentItem(idx)
					absoluteSelectedPath, _ := filepath.Abs(filepath.Join(fe.context.CurrentPath, currentName))
					fe.directoryToIndexMap[absoluteSelectedPath] = idx
				}
			}
			return nil
		}
		switch event.Rune() {
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
				openInNvim(filePath, fe.context.App)
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
		return event
	})
}

// Run starts the file explorer
func (fe *FileExplorer) Run() error {
	return fe.context.App.SetRoot(fe.rootFlex, true).Run()
}
