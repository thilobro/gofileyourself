package widget

import "github.com/rivo/tview"

type Context struct {
	App             *tview.Application
	CurrentPath     string
	ShowHiddenFiles bool
}

type WidgetInterface interface {
	Run() error
	Draw()
	SetupKeyBindings()
	Root() tview.Primitive
}

type Factory interface {
	New(ctx *Context) (WidgetInterface, error)
}
