package explorer

import (
	"strings"

	gostring "github.com/boyter/go-string"
	"github.com/rivo/tview"
)

func (explorer *Explorer) applyTheme() {
	theme := explorer.context.Theme

	// Set global background through root flex
	explorer.rootFlex.SetBackgroundColor(theme.Bg0)
	explorer.listFlex.SetBackgroundColor(theme.Bg0)

	// Style the lists
	explorer.currentList.
		SetMainTextColor(theme.Fg1).
		SetSelectedTextColor(theme.Palette8).
		SetSelectedBackgroundColor(theme.Palette6).
		SetBackgroundColor(theme.Bg0)

	if explorer.parentList != nil {
		if list, ok := explorer.parentList.(*tview.List); ok {
			list.
				SetMainTextColor(theme.Fg1).
				SetSelectedTextColor(theme.Palette8).
				SetSelectedBackgroundColor(theme.Palette4).
				SetBackgroundColor(theme.Bg0)
		}
	}

	// Style the selected list/preview
	if list, ok := explorer.selectedList.(*tview.List); ok {
		list.
			SetMainTextColor(theme.Fg1).
			SetSelectedTextColor(theme.Palette8).
			SetSelectedBackgroundColor(theme.Palette2).
			SetBackgroundColor(theme.Bg0)
	} else if textView, ok := explorer.selectedList.(*tview.TextView); ok {
		textView.
			SetTextColor(theme.Fg0).
			SetBackgroundColor(theme.Bg0)
	}

	// Style the footer
	if explorer.footer != nil {
		explorer.footer.
			SetFieldBackgroundColor(theme.Bg1).
			SetFieldTextColor(theme.Fg0).
			SetBackgroundColor(theme.Bg0)
	}
}

func (explorer *Explorer) highlightSearchInput() {
	for i := 0; i < explorer.currentList.GetItemCount(); i++ {
		displayName, text := explorer.currentList.GetItemText(i)
		startHighlightMarker := "[red::b]"
		endHighlightMarker := "[-::-]"
		displayNameWithoutHighlight := strings.Replace(displayName, startHighlightMarker, "", -1)
		displayNameWithoutHighlight = strings.Replace(displayNameWithoutHighlight, endHighlightMarker, "", -1)
		indeces := gostring.IndexAllIgnoreCase(displayNameWithoutHighlight, explorer.searchInput, -1)
		explorer.currentList.SetItemText(i, gostring.HighlightString(displayNameWithoutHighlight, indeces, startHighlightMarker, endHighlightMarker), text)
	}
}
