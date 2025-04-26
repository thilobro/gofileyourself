package widget

import "github.com/rivo/tview"

type Context struct {
	App         *tview.Application
	CurrentPath string
}

type WidgetInterface interface {
	Run() error
	Draw()
	SetupKeyBindings()
	Root() tview.Primitive
	UpdateCurrentPath(newCurrentPath string) error
}

type Factory interface {
	New(ctx *Context) (WidgetInterface, error)
}
