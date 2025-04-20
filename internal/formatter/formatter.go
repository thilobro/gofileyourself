package formatter

import (
	"fmt"
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
	styles.Register(chroma.MustNewStyle("gruvbox", chroma.StyleEntries{
		chroma.Text:               "#ebdbb2",
		chroma.Error:              "#fb4934",
		chroma.Comment:            "#928374",
		chroma.Keyword:            "#fb4934",
		chroma.KeywordConstant:    "#d3869b",
		chroma.KeywordDeclaration: "#fb4934",
		chroma.KeywordNamespace:   "#fb4934",
		chroma.KeywordType:        "#fabd2f",
		chroma.Operator:           "#ebdbb2",
		chroma.Punctuation:        "#ebdbb2",
		chroma.Name:               "#ebdbb2",
		chroma.NameAttribute:      "#b8bb26",
		chroma.NameBuiltin:        "#fabd2f",
		chroma.NameClass:          "#8ec07c",
		chroma.NameConstant:       "#d3869b",
		chroma.NameDecorator:      "#d3869b",
		chroma.NameFunction:       "#b8bb26",
		chroma.NameTag:            "#fb4934",
		chroma.NameVariable:       "#ebdbb2",
		chroma.Literal:            "#d3869b",
		chroma.LiteralNumber:      "#d3869b",
		chroma.LiteralString:      "#b8bb26",
		chroma.Background:         "#282828",
	}))
}
