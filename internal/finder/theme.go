package finder

func (finder *Finder) applyTheme() {
	theme := finder.context.Theme

	// Set global background through root flex
	finder.rootFlex.SetBackgroundColor(theme.Bg0)

	// Style the lists
	finder.searchedList.
		SetMainTextColor(theme.Fg1).
		SetSelectedTextColor(theme.Palette8).
		SetSelectedBackgroundColor(theme.Palette6).
		SetBackgroundColor(theme.Bg0)

	// Style the footer
	if finder.footer != nil {
		finder.footer.
			SetFieldBackgroundColor(theme.Bg1).
			SetFieldTextColor(theme.Fg0).
			SetBackgroundColor(theme.Bg0).
			SetBorder(true).SetTitle(finder.title)
	}
}
