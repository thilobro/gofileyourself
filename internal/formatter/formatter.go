package formatter

import (
	"fmt"
	"gofileyourself/internal/theme"
	"io"

	"github.com/alecthomas/chroma"
	"github.com/alecthomas/chroma/formatters"
	"github.com/alecthomas/chroma/styles"
	"github.com/gdamore/tcell"
)

type TviewFormatter struct{}

func (f *TviewFormatter) Format(w io.Writer, style *chroma.Style, iterator chroma.Iterator) error {
	for token := iterator(); token != chroma.EOF; token = iterator() {
		entry := style.Get(token.Type)
		if entry.Colour.IsSet() {
			// Convert chroma color to tcell color
			color := chromaColorToTcell(entry.Colour)
			r, g, b := color.RGB()
			fmt.Fprintf(w, "[#%02x%02x%02x]%s", r, g, b, token.Value)
		} else {
			fmt.Fprint(w, token.Value)
		}
	}
	// Reset color at the end
	fmt.Fprint(w, "[white]")
	return nil
}

// chromaColorToTcell converts a chroma.Colour to tcell.Color
func chromaColorToTcell(c chroma.Colour) tcell.Color {
	if !c.IsSet() {
		return tcell.ColorWhite
	}
	return tcell.NewRGBColor(int32(c.Red()), int32(c.Green()), int32(c.Blue()))
}

// Register the formatter
func RegisterCustomFormatter() {
	formatters.Register("tview", &TviewFormatter{})
	styles.Register(theme.GetFormatterStyle())
}
