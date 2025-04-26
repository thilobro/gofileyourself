package finder

import (
	"gofileyourself/internal/widget"

	"github.com/rivo/tview"
)

// Display is the main struct for the display package.
type Finder struct {
	context  *widget.Context
	rootFlex *tview.Flex
}

func NewFinder(context *widget.Context) (*Finder, error) {
	return &Finder{
		context:  context,
		rootFlex: tview.NewFlex(),
	}, nil
}

func (finder *Finder) Root() tview.Primitive {
	return finder.rootFlex
}

func (finder *Finder) Draw() {
}

func (finder *Finder) Run() error {
	return nil
}

func (finder *Finder) SetupKeyBindings() {
}

func (finder *Finder) UpdateCurrentPath(newCurrentPath string) error {
	finder.context.CurrentPath = newCurrentPath
	return nil
}
