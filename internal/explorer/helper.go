package explorer

import (
	"bytes"
	"os"
	"os/exec"

	"github.com/alecthomas/chroma/formatters"
	"github.com/alecthomas/chroma/lexers"
	"github.com/alecthomas/chroma/styles"
	"github.com/rivo/tview"
)

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
