package theme

import (
	"log"
	"os"

	"github.com/alecthomas/chroma"
	"github.com/gdamore/tcell/v2"
	"github.com/thilobro/gofileyourself/internal/formatter"
	"gopkg.in/yaml.v3"
)

type Theme struct {
	Bg0    tcell.Color
	Bg1    tcell.Color
	Fg0    tcell.Color
	Fg1    tcell.Color
	Gray   tcell.Color
	Red    tcell.Color
	Green  tcell.Color
	Yellow tcell.Color
	Blue   tcell.Color
	Purple tcell.Color
	Aqua   tcell.Color
	Orange tcell.Color
	Black  tcell.Color
}

func NewTheme(themePath *string) *Theme {
	var theme *Theme
	var err error
	if themePath != nil {
		theme, err = GetThemeFromPath(themePath)
		if err != nil {
			theme = GetDefaultTheme()
		}
	} else {
		theme = GetDefaultTheme()
	}
	formatter.RegisterCustomFormatter(GetDefaultFormatterStyle())
	return theme
}

func GetThemeFromPath(themePath *string) (*Theme, error) {
	themeFile, err := os.ReadFile(*themePath)
	if err != nil {
		return nil, err
	}
	var theme Theme
	if err := yaml.Unmarshal(themeFile, &theme); err != nil {
		log.Fatal(err)
		panic(err)
	}
	return &theme, nil
}

func GetDefaultTheme() *Theme {
	return &Theme{
		Bg0:    tcell.NewRGBColor(40, 40, 40),    // #282828
		Bg1:    tcell.NewRGBColor(60, 56, 54),    // #3c3836
		Fg0:    tcell.NewRGBColor(251, 241, 199), // #fbf1c7
		Fg1:    tcell.NewRGBColor(235, 219, 178), // #ebdbb2
		Gray:   tcell.NewRGBColor(146, 131, 116), // #928374
		Red:    tcell.NewRGBColor(251, 73, 52),   // #fb4934
		Green:  tcell.NewRGBColor(184, 187, 38),  // #b8bb26
		Yellow: tcell.NewRGBColor(250, 189, 47),  // #fabd2f
		Blue:   tcell.NewRGBColor(131, 165, 152), // #83a598
		Purple: tcell.NewRGBColor(211, 134, 155), // #d3869b
		Aqua:   tcell.NewRGBColor(142, 192, 124), // #8ec07c
		Orange: tcell.NewRGBColor(254, 128, 25),  // #fe8019
		Black:  tcell.NewRGBColor(0, 0, 0),       // #000000
	}
}

func GetDefaultFormatterStyle() *chroma.Style {
	return chroma.MustNewStyle("gruvbox", chroma.StyleEntries{
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
	})
}
