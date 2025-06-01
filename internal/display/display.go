package display

import (
	"os"

	"github.com/thilobro/gofileyourself/internal/widget"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

// Display is the main struct for the display package.
type Display struct {
	context       *widget.Context
	mode          widget.Mode
	activeWidget  widget.WidgetInterface
	widgetFactory map[widget.Mode]widget.Factory
}

// setupKeyBindings configures keyboard input handling
func (display *Display) setupKeyBindings() {
	display.context.App.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyCtrlC:
			display.context.App.Stop()
		case tcell.KeyCtrlF:
			display.setMode(widget.Find)
			return nil // Consume the event
		case tcell.KeyEscape:
			display.setMode(widget.Explorer)
			return nil // Consume the event
		}
		// Let the active widget handle other keys
		if display.activeWidget != nil {
			inputHandler := display.activeWidget.GetInputCapture()
			return inputHandler(event)
		}
		return event
	})
}

func (display *Display) setMode(mode widget.Mode) {
	display.mode = mode
	display.context.App.SetInputCapture(nil) // Clear any existing input capture
	display.setActiveWidgetBasedOnMode(mode)
	display.setupKeyBindings()
	display.context.App.SetFocus(display.activeWidget.Root())
	display.activeWidget.Draw()
}

func (display *Display) setActiveWidgetBasedOnMode(mode widget.Mode) {
	factory, exists := display.widgetFactory[mode]
	if !exists {
		panic("no factory for mode")
	}

	widget, err := factory.New(display.context)
	if err != nil {
		panic(err)
	}
	display.activeWidget = widget
	display.context.App.SetRoot(display.activeWidget.Root(), true)
}

func NewDisplay(factories map[widget.Mode]widget.Factory, chooseFilePath *string) (*Display, error) {
	app := tview.NewApplication()
	currentPath, err := os.Getwd()
	if err != nil {
		return nil, err
	}
	display := &Display{}

	explorerFactory := factories[widget.Explorer]
	context := &widget.Context{
		App:             app,
		CurrentPath:     currentPath,
		ShowHiddenFiles: false,
		OnWidgetResult:  display.onWidgetResult,
		ChooseFilePath:  chooseFilePath,
	}
	explorerWidget, err := explorerFactory.New(context)
	if err != nil {
		return nil, err
	}
	display.context = context
	display.activeWidget = explorerWidget
	display.widgetFactory = factories
	display.mode = widget.Explorer

	return display, nil
}

func (display *Display) onWidgetResult(mode widget.Mode, result string) {
	display.setMode(widget.Explorer)
}

// Run starts the file explorer
func (display *Display) Run() error {
	display.setupKeyBindings()
	return display.context.App.SetRoot(display.activeWidget.Root(), true).Run()
}
