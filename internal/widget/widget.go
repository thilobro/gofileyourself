package widget

import (
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

type Mode int

const (
	Explorer Mode = iota
	Find
)

type Context struct {
	App             *tview.Application
	CurrentPath     string
	ShowHiddenFiles bool
	OnWidgetResult  func(mode Mode, result string)
}

type WidgetInterface interface {
	Run() error
	Draw()
	SetupKeyBindings()
	Root() tview.Primitive
	GetInputCapture() func(*tcell.EventKey) *tcell.EventKey
}

type Factory interface {
	New(ctx *Context) (WidgetInterface, error)
}
