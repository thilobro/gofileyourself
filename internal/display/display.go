package display

import (
	"gofileyourself/internal/widget"
	"os"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

type Mode int

const (
	Explorer Mode = iota
	Find
)

// Display is the main struct for the display package.
type Display struct {
	context       *widget.Context
	mode          Mode
	activeWidget  widget.WidgetInterface
	widgetFactory map[Mode]widget.Factory
}

// setupKeyBindings configures keyboard input handling
func (display *Display) setupKeyBindings() {
	display.context.App.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyCtrlF:
			display.setMode(Find)
			return nil // Consume the event
		case tcell.KeyEscape:
			display.setMode(Explorer)
			return nil // Consume the event
		}
		// Let the active widget handle other keys
		return event
	})

	// Set up widget-specific key bindings
	display.activeWidget.SetupKeyBindings()
}

func (display *Display) setMode(mode Mode) {
	display.mode = mode
	display.setActiveWidgetBasedOnMode(mode)
	display.activeWidget.Draw()
}

func (display *Display) setActiveWidgetBasedOnMode(mode Mode) {
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

func NewDisplay(factories map[Mode]widget.Factory) (*Display, error) {
	app := tview.NewApplication()
	currentPath, err := os.Getwd()
	if err != nil {
		return nil, err
	}

	context := &widget.Context{
		App:         app,
		CurrentPath: currentPath,
	}

	explorerFactory := factories[Explorer]
	widget, err := explorerFactory.New(context)
	if err != nil {
		return nil, err
	}

	return &Display{
		context:       context,
		mode:          Explorer,
		activeWidget:  widget,
		widgetFactory: factories,
	}, nil
}

// Run starts the file explorer
func (display *Display) Run() error {
	display.setupKeyBindings()
	return display.context.App.SetRoot(display.activeWidget.Root(), true).Run()
}
