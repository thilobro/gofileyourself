package explorer

import (
	"github.com/thilobro/gofileyourself/internal/widget"
)

type Factory struct{}

func (f *Factory) New(ctx *widget.Context) (widget.WidgetInterface, error) {
	return NewFileExplorer(ctx)
}
