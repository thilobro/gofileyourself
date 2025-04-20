package explorer

import (
	"bytes"
	"gofileyourself/internal/formatter"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/alecthomas/chroma/formatters"
	"github.com/alecthomas/chroma/lexers"
	"github.com/alecthomas/chroma/styles"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

func init() {
	formatter.RegisterCustomFormatter()
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
func loadDirectory(path string, showHiddenFiles bool) (*tview.List, error) {
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
			fileName := file.Name()
			if !showHiddenFiles && len(fileName) > 0 && fileName[0] == '.' {
				continue
			}
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

	// Create text view
	textView := tview.NewTextView().
		SetDynamicColors(true).
		SetRegions(true).
		SetWordWrap(true)

	// Detect language based on file extension
	lexer := lexers.Match(path)
	if lexer == nil {
		lexer = lexers.Fallback
	}

	// Use gruvbox style
	style := styles.Get("gruvbox")
	if style == nil {
		style = styles.Fallback
	}

	formatter := formatters.Get("tview")
	if formatter == nil {
		formatter = formatters.Fallback
	}

	iterator, err := lexer.Tokenise(nil, string(content))
	if err != nil {
		return nil, err
	}

	// Create buffer to store formatted output
	var buf bytes.Buffer
	err = formatter.Format(&buf, style, iterator)
	if err != nil {
		return nil, err
	}

	textView.SetText(buf.String())
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
	header              *tview.TextView
	showHiddenFiles     bool
}

func (fe *FileExplorer) applyGruvboxTheme() {
	theme := newGruvboxTheme()

	// Set global background through root flex
	fe.rootFlex.SetBackgroundColor(theme.bg0)
	fe.listFlex.SetBackgroundColor(theme.bg0)

	// Style the lists
	fe.currentList.
		SetMainTextColor(theme.fg1).
		SetSelectedTextColor(theme.bg0).
		SetSelectedBackgroundColor(theme.aqua).
		SetBackgroundColor(theme.bg0)

	if fe.parentList != nil {
		if list, ok := fe.parentList.(*tview.List); ok {
			list.
				SetMainTextColor(theme.fg1).
				SetSelectedTextColor(theme.bg0).
				SetSelectedBackgroundColor(theme.blue).
				SetBackgroundColor(theme.bg0)
		}
	}

	// Style the selected list/preview
	if list, ok := fe.selectedList.(*tview.List); ok {
		list.
			SetMainTextColor(theme.fg1).
			SetSelectedTextColor(theme.bg0).
			SetSelectedBackgroundColor(theme.green).
			SetBackgroundColor(theme.bg0)
	} else if textView, ok := fe.selectedList.(*tview.TextView); ok {
		textView.
			SetTextColor(theme.fg0).
			SetBackgroundColor(theme.bg0)
	}

	// Style the footer
	if fe.footer != nil {
		fe.footer.
			SetFieldBackgroundColor(theme.bg1).
			SetFieldTextColor(theme.fg0).
			SetBackgroundColor(theme.bg0)
	}

	// Style the header
	if fe.header != nil {
		fe.header.
			SetBackgroundColor(theme.blue)
		fe.header.SetTextColor(theme.bg0)
	}
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
		showHiddenFiles:     false,
	}

	if err := fe.initialize(); err != nil {
		return nil, err
	}

	return fe, nil
}

// initialize sets up the initial state of the FileExplorer
func (fe *FileExplorer) initialize() error {
	var err error
	fe.currentList, err = loadDirectory(fe.currentPath, fe.showHiddenFiles)
	if err != nil {
		return err
	}

	parentPath := filepath.Join(fe.currentPath, "..")
	newParentList, err := loadDirectory(parentPath, fe.showHiddenFiles)
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
	fileInfo, err := os.Stat(selectedPath)
	if err != nil {
		return err
	}
	if fileInfo.IsDir() {
		fe.selectedList, err = loadDirectory(selectedPath, fe.showHiddenFiles)
		if err != nil {
			return err
		}
	} else {
		fe.selectedList, err = loadFilePreview(selectedPath)
		if err != nil {
			return err
		}
	}

	currentAbsolutePath, _ := filepath.Abs(fe.currentPath)
	fe.header = tview.NewTextView().
		SetDynamicColors(true).
		SetRegions(true).
		SetWordWrap(true).
		SetText(currentAbsolutePath)

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
	fe.app.SetRoot(fe.rootFlex, true)
	fe.applyGruvboxTheme()
}

// updateSelectedDirectory updates the selected directory/file preview
func (fe *FileExplorer) updateSelectedDirectory(selectedPath string) error {
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

	fe.draw()
	return nil
}

// changeCurrentDirectory changes the current directory and updates related views
func (fe *FileExplorer) changeCurrentDirectory(path string) error {
	// Update current directory
	currentAbsolutePath, _ := filepath.Abs(path)
	currentDirectoryIndex := fe.directoryToIndexMap[currentAbsolutePath]
	newCurrentList, err := loadDirectory(path, fe.showHiddenFiles)
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
		newParentList, err := loadDirectory(parentPath, fe.showHiddenFiles)
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

	// Update header
	fe.updateHeader(currentAbsolutePath)

	fe.currentPath = path
	return nil
}

func (fe *FileExplorer) updateHeader(text string) {
	fe.header.SetText(text)
	fe.draw()
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

func (fe *FileExplorer) runFooterCommand(inputText string) {
	switch inputText[0] {
	case '/':
		searchTerm := inputText[1:]
		matchingIndeces := fe.currentList.FindItems(searchTerm, "", false, true)
		if len(matchingIndeces) > 0 {
			fe.updateCurrentLine(matchingIndeces[0])
		}
	case ':':
		command := inputText[1:]
		switch command {
		case "q":
			fe.app.Stop()
		}
	}
	fe.app.SetFocus(fe.currentList)
}

func (fe *FileExplorer) handleFooterInput(prompt string) {
	fe.footer = tview.NewInputField().SetText(prompt)
	fe.footer.SetDoneFunc(
		func(key tcell.Key) {
			if key == tcell.KeyEnter {
				inputText := fe.footer.GetText()
				fe.runFooterCommand(inputText)
				fe.app.SetFocus(fe.currentList)
			}
			fe.draw()
		},
	)
	fe.draw()
	fe.app.SetFocus(fe.footer)
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
			fe.handleFooterInput("/")
			return nil
		case ':': // command
			fe.handleFooterInput(":")
			return nil
		}
		return event
	})
}

// Run starts the file explorer
func (fe *FileExplorer) Run() error {
	return fe.app.SetRoot(fe.rootFlex, true).Run()
}
