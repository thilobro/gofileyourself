package finder

import (
	"github.com/thilobro/gofileyourself/internal/widget"
)

type Factory struct{}

func (f *Factory) New(ctx *widget.Context) (widget.WidgetInterface, error) {
	finderType := "find"
	return NewFinder(ctx, &finderType)
}

type GrepFactory struct{}

func (f *GrepFactory) New(ctx *widget.Context) (widget.WidgetInterface, error) {
	finderType := "grep"
	return NewFinder(ctx, &finderType)
}

type FindRecentFactory struct{}

func (f *FindRecentFactory) New(ctx *widget.Context) (widget.WidgetInterface, error) {
	finderType := "findrecent"
	return NewFinder(ctx, &finderType)
}
