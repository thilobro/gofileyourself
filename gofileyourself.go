package main

import (
	"github.com/rivo/tview"
	"os"
	"os/exec"
	"path/filepath"
	"github.com/gdamore/tcell/v2"
)

func loadDirectory(path string) (*tview.List, error) {
	file_info, err := os.Stat(path)
	if err != nil {
		return nil, err
	}
	if file_info.IsDir() {
		files, err := os.ReadDir(path)
		if err != nil {
			return nil, err
		}
		list := tview.NewList().ShowSecondaryText(false)
		for _, file := range files {
			list.AddItem(file.Name(), "", 0, nil)
		}
		return list, nil
	} else {
		return nil, nil
	}
}

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

func openInNvim(path string, app *tview.Application) error {
    // Suspend the terminal UI
    app.Suspend(func() {
        cmd := exec.Command("nvim", path)
        cmd.Stdin = os.Stdin
        cmd.Stdout = os.Stdout
        cmd.Stderr = os.Stderr
        cmd.Run()
    })
    
    return nil
}

func main() {
	app := tview.NewApplication()
	directory_to_index_map := make(map[string]int)

	// current directory
	current_path, err := os.Getwd()
	current_list, err := loadDirectory(current_path)
	if err != nil {
		panic(err)
	}

	// parent directory
	parent_path := filepath.Join(current_path, "..")
	parent_list, err := loadDirectory(parent_path)
	if err != nil {
		panic(err)
	}

	// selected directory
	var selected_list tview.Primitive
	selected_name, _ := current_list.GetItemText(0)
	selected_path := filepath.Join(current_path, selected_name)
	selected_list, err = loadDirectory(selected_path)
	if err != nil {
		panic(err)
	}

	flex := tview.NewFlex()

	draw := func() {
		flex.Clear()
		if parent_list != nil {
			flex.AddItem(parent_list, 0, 1, false)
		}
		if current_list != nil {
			flex.AddItem(current_list, 0, 1, true)
		}
		if selected_list != nil {
			flex.AddItem(selected_list, 0, 1, false)
		}
		app.SetRoot(flex, true)
	}
	draw()


	updateSelectedDirectory := func(selected_path string, event *tcell.EventKey) (*tcell.EventKey, error) {
		selected_absolute_path, _ := filepath.Abs(selected_path)
		selected_directory_index := directory_to_index_map[selected_absolute_path]
		new_selected_list, err := loadDirectory(selected_path)
		if err != nil {
			return nil, err
		}
		if new_selected_list == nil {
			selected_list, _ = loadFilePreview(selected_path)
			draw()
			return event, nil
		}
		new_selected_list.SetCurrentItem(selected_directory_index)
		if err != nil {
			return nil, err
		}
		selected_list = new_selected_list
		draw()
		return event, nil
	}

	changeCurrentDirectory := func(path string, event *tcell.EventKey) (*tcell.EventKey, error) {
		// current directory
		current_absolute_path, _ := filepath.Abs(path)
		current_directory_index := directory_to_index_map[current_absolute_path]
		new_current_list, err := loadDirectory(path)
		new_current_list.SetInputCapture(current_list.GetInputCapture())
		new_current_list.SetCurrentItem(current_directory_index)
		current_list = new_current_list

		// parent directory
		parent_path := filepath.Join(path, "..")
		new_parent_list, err := loadDirectory(parent_path)
		if err != nil {
			return nil, err
		}
		parent_directory_index := 0
		for i := 0; i < new_parent_list.GetItemCount(); i++ {
			if text, _ := new_parent_list.GetItemText(i); text == filepath.Base(path) {
				parent_directory_index = i
				break
			}
		}
		new_parent_list.SetCurrentItem(parent_directory_index)
		parent_list = new_parent_list


		// selected directory
		selected_name, _ := current_list.GetItemText(current_directory_index)
		selected_path := filepath.Join(path, selected_name)
		updateSelectedDirectory(selected_path, event)
		current_path = path
		
		return event, nil
	}

	updateCurrentLine := func(line_index int, event *tcell.EventKey) (*tcell.EventKey, error) {
		current_list.SetCurrentItem(line_index)
		current_absolute_path, _ := filepath.Abs(current_path)
		directory_to_index_map[current_absolute_path] = line_index
		selected_name, _ := current_list.GetItemText(line_index)
		updateSelectedDirectory(filepath.Join(current_path, selected_name), event)
		return event, nil
	}

	if _, err := changeCurrentDirectory(current_path, nil); err != nil {
		panic(err)
	}

	current_list.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Rune() {
		case 'j': // scroll down
			updateCurrentLine(current_list.GetCurrentItem() + 1, event)
			return nil
		case 'k': // scroll up
			updateCurrentLine(current_list.GetCurrentItem() - 1, event)
			return nil
		case 'q': // quit
			app.Stop()
			return nil
		case 'l': // open dir or file
			current_item := current_list.GetCurrentItem()
			file_name, _ := current_list.GetItemText(current_item)
			file_path := filepath.Join(current_path, file_name)
			file_info, err := os.Stat(file_path)
			if err != nil {
				return event
			}
			if file_info.IsDir() {
				event, err := changeCurrentDirectory(file_path, event)
				if err != nil {
					return event
				}
			} else {
				openInNvim(file_path, app)
				return nil
			}
			current_path = file_path
			return nil
		case 'h': // go up directory
			dir_path := filepath.Join(current_path, "..")
			event, err := changeCurrentDirectory(dir_path, event)
			if err != nil {
				return event
			}
			current_path = dir_path
			return nil
		}
		return event
	})

	if err := app.SetRoot(flex, true).Run(); err != nil {
		panic(err)
	}
}
