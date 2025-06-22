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
	Bg0      tcell.Color
	Bg1      tcell.Color
	Fg0      tcell.Color
	Fg1      tcell.Color
	Palette0 tcell.Color
	Palette1 tcell.Color
	Palette2 tcell.Color
	Palette3 tcell.Color
	Palette4 tcell.Color
	Palette5 tcell.Color
	Palette6 tcell.Color
	Palette7 tcell.Color
	Palette8 tcell.Color
}

type ThemeConfig struct {
	Bg0      string `yaml:"bg0"`
	Bg1      string `yaml:"bg1"`
	Fg0      string `yaml:"fg0"`
	Fg1      string `yaml:"fg1"`
	Palette0 string `yaml:"palette0"`
	Palette1 string `yaml:"palette1"`
	Palette2 string `yaml:"palette2"`
	Palette3 string `yaml:"palette3"`
	Palette4 string `yaml:"palette4"`
	Palette5 string `yaml:"palette5"`
	Palette6 string `yaml:"palette6"`
	Palette7 string `yaml:"palette7"`
	Palette8 string `yaml:"palette8"`
}

func GetFormatterStyleForTheme(themeConfig *ThemeConfig) *chroma.Style {
	return chroma.MustNewStyle("gruvbox", chroma.StyleEntries{
		chroma.Text:               themeConfig.Fg1,
		chroma.Error:              themeConfig.Palette1,
		chroma.Comment:            themeConfig.Palette0,
		chroma.Keyword:            themeConfig.Palette1,
		chroma.KeywordConstant:    themeConfig.Palette5,
		chroma.KeywordDeclaration: themeConfig.Palette1,
		chroma.KeywordNamespace:   themeConfig.Palette1,
		chroma.KeywordType:        themeConfig.Palette3,
		chroma.Operator:           themeConfig.Fg1,
		chroma.Punctuation:        themeConfig.Fg1,
		chroma.Name:               themeConfig.Fg1,
		chroma.NameAttribute:      themeConfig.Palette2,
		chroma.NameBuiltin:        themeConfig.Palette3,
		chroma.NameClass:          themeConfig.Palette6,
		chroma.NameConstant:       themeConfig.Palette5,
		chroma.NameDecorator:      themeConfig.Palette5,
		chroma.NameFunction:       themeConfig.Palette2,
		chroma.NameTag:            themeConfig.Palette1,
		chroma.NameVariable:       themeConfig.Fg1,
		chroma.Literal:            themeConfig.Palette5,
		chroma.LiteralNumber:      themeConfig.Palette5,
		chroma.LiteralString:      themeConfig.Palette2,
		chroma.Background:         themeConfig.Bg0,
	})
}

func NewTheme(themePath *string) *Theme {
	var themeConfig *ThemeConfig
	var err error
	if themePath != nil {
		themeConfig, err = GetThemeConfigFromPath(themePath)
		if err != nil {
			themeConfig = GetDefaultThemeConfig()
		}
	} else {
		themeConfig = GetDefaultThemeConfig()
	}
	formatter.RegisterCustomFormatter(GetFormatterStyleForTheme(themeConfig))
	return &Theme{
		Bg0:      tcell.GetColor(themeConfig.Bg0),
		Bg1:      tcell.GetColor(themeConfig.Bg1),
		Fg0:      tcell.GetColor(themeConfig.Fg0),
		Fg1:      tcell.GetColor(themeConfig.Fg1),
		Palette0: tcell.GetColor(themeConfig.Palette0),
		Palette1: tcell.GetColor(themeConfig.Palette1),
		Palette2: tcell.GetColor(themeConfig.Palette2),
		Palette3: tcell.GetColor(themeConfig.Palette3),
		Palette4: tcell.GetColor(themeConfig.Palette4),
		Palette5: tcell.GetColor(themeConfig.Palette5),
		Palette6: tcell.GetColor(themeConfig.Palette6),
		Palette7: tcell.GetColor(themeConfig.Palette7),
		Palette8: tcell.GetColor(themeConfig.Palette8),
	}
}

func GetThemeConfigFromPath(themePath *string) (*ThemeConfig, error) {
	log.Println("themePath", *themePath)
	themeFile, err := os.ReadFile(*themePath)
	if err != nil {
		return nil, err
	}
	var themeConfig ThemeConfig
	if err := yaml.Unmarshal(themeFile, &themeConfig); err != nil {
		log.Fatal(err)
		panic(err)
	}
	return &themeConfig, nil
}

func GetDefaultThemeConfig() *ThemeConfig {
	return &ThemeConfig{
		Bg0:      "#282828",
		Bg1:      "#3c3836",
		Fg0:      "#fbf1c7",
		Fg1:      "#ebdbb2",
		Palette0: "#928374",
		Palette1: "#fb4934",
		Palette2: "#b8bb26",
		Palette3: "#fabd2f",
		Palette4: "#83a598",
		Palette5: "#d3869b",
		Palette6: "#8ec07c",
		Palette7: "#fe8019",
		Palette8: "#000000",
	}
}
