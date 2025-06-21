package widget

import (
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"github.com/thilobro/gofileyourself/internal/config"
	"github.com/thilobro/gofileyourself/internal/theme"
)

type Mode int

const (
	Explorer Mode = iota
	Find
	FindRecent
)

type Context struct {
	App              *tview.Application
	CurrentPath      string
	ShowHiddenFiles  bool
	OnWidgetResult   func(mode Mode, result string)
	ChooseFilePath   *string
	SelectedFilePath *string
	Config           *config.Config
	Theme            *theme.Theme
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
