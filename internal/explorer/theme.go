package explorer

import "github.com/gdamore/tcell/v2"

// GruvboxTheme contains the color definitions for Gruvbox dark theme
type GruvboxTheme struct {
	bg0    tcell.Color
	bg1    tcell.Color
	fg0    tcell.Color
	fg1    tcell.Color
	gray   tcell.Color
	red    tcell.Color
	green  tcell.Color
	yellow tcell.Color
	blue   tcell.Color
	purple tcell.Color
	aqua   tcell.Color
	orange tcell.Color
}

func newGruvboxTheme() *GruvboxTheme {
	return &GruvboxTheme{
		bg0:    tcell.NewRGBColor(40, 40, 40),    // #282828
		bg1:    tcell.NewRGBColor(60, 56, 54),    // #3c3836
		fg0:    tcell.NewRGBColor(251, 241, 199), // #fbf1c7
		fg1:    tcell.NewRGBColor(235, 219, 178), // #ebdbb2
		gray:   tcell.NewRGBColor(146, 131, 116), // #928374
		red:    tcell.NewRGBColor(251, 73, 52),   // #fb4934
		green:  tcell.NewRGBColor(184, 187, 38),  // #b8bb26
		yellow: tcell.NewRGBColor(250, 189, 47),  // #fabd2f
		blue:   tcell.NewRGBColor(131, 165, 152), // #83a598
		purple: tcell.NewRGBColor(211, 134, 155), // #d3869b
		aqua:   tcell.NewRGBColor(142, 192, 124), // #8ec07c
		orange: tcell.NewRGBColor(254, 128, 25),  // #fe8019
	}
}
