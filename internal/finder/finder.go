package finder

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"sort"
	"strconv"
	"strings"
	"sync"

	"github.com/thilobro/gofileyourself/internal/helper"
	"github.com/thilobro/gofileyourself/internal/loader"
	"github.com/thilobro/gofileyourself/internal/widget"

	gostring "github.com/boyter/go-string"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

const (
	constNumSearchLines = 1000
	// constMaxResults caps how many candidates the left list shows, so a broad
	// query on a huge tree stays instant instead of building an enormous list.
	constMaxResults = 500
)

type Finder struct {
	context              *widget.Context
	rootFlex             *tview.Flex
	footer               *tview.InputField
	fileList             *tview.List
	searchedList         *tview.List
	previousSearchedList *tview.List
	selectedList         tview.Primitive
	currentFocusedWidget tview.Primitive
	searchTerm           string
	fuzzySearchQuit      chan bool
	listUpdateChan       chan *tview.List // Channel for list updates
	listUpdateMutex      sync.Mutex
	title                string
	fileListPrimary      []string // cached primary text of fileList items
	fileListSecondary    []string // cached secondary text of fileList items
	lastDrawWidth        int      // width at the last list re-shape
	listDirty            bool     // set when searchedList needs re-shaping
	isGrep               bool     // grep mode: search file contents via ripgrep
	isFind               bool     // find mode: rank files via rg --files + fzf --filter
	findCandidates       []string // cached file list for find mode (from rg --files)
	searchMu             sync.Mutex
	searchGen            int                // generation of the latest external search
	searchCancel         context.CancelFunc // cancels the in-flight search process
}

// cacheFileListTexts snapshots the fileList item texts so fuzzySearch does not
// have to call GetItemText (which locks and indexes the list) once per term per
// item on every keystroke. fileList does not change while the user is typing.
func (finder *Finder) cacheFileListTexts() {
	count := finder.fileList.GetItemCount()
	finder.fileListPrimary = make([]string, count)
	finder.fileListSecondary = make([]string, count)
	for i := 0; i < count; i++ {
		finder.fileListPrimary[i], finder.fileListSecondary[i] = finder.fileList.GetItemText(i)
	}
}

func (finder *Finder) setDrawFunc() {
	finder.searchedList.SetDrawFunc(func(screen tcell.Screen, x int, y int, width int, height int) (int, int, int, int) {
		finder.listUpdateMutex.Lock()
		defer finder.listUpdateMutex.Unlock()
		// The shaped output depends only on the current result list and the
		// width, so skip the O(items) rebuild on plain scrolls/cursor moves.
		if width == finder.lastDrawWidth && !finder.listDirty {
			return x, y, width, height
		}
		finder.lastDrawWidth = width
		finder.listDirty = false
		helper.CopyListContent(finder.previousSearchedList, finder.searchedList)
		helper.ShortenPathsIfNecessary(finder.searchedList, width)
		return x, y, width, height
	})
}

func (finder *Finder) showFind() {
	finder.title = "Find"
	finder.isFind = true
	// Find no longer walks the tree in-process; `rg --files` supplies candidates
	// and fzf ranks them per keystroke. Lists start empty and fill in.
	finder.loadFindCandidates()
	finder.fileList.Clear()
	finder.searchedList.Clear()
	finder.previousSearchedList.Clear()
	finder.searchInDirectory()
	// Show the full (unranked) candidate list until the user starts typing.
	go finder.runFindFilter("")
}

func NewFinder(context *widget.Context, finderType *string) (*Finder, error) {
	finder := &Finder{
		context:              context,
		rootFlex:             tview.NewFlex(),
		footer:               tview.NewInputField(),
		fileList:             tview.NewList(),
		selectedList:         tview.NewList().ShowSecondaryText(false),
		searchedList:         tview.NewList().ShowSecondaryText(false),
		previousSearchedList: tview.NewList().ShowSecondaryText(false),
		searchTerm:           "",
		fuzzySearchQuit:      make(chan bool),
		listUpdateChan:       make(chan *tview.List, 10), // Buffered channel
		title:                "",
	}
	finder.SetupKeyBindings()
	finder.currentFocusedWidget = finder.searchedList
	finder.setDrawFunc()

	// Start list update handler
	go finder.handleListUpdates()

	switch *finderType {
	case "find":
		finder.showFind()
	case "findrecent":
		finder.showRecentHistory()
	case "grep":
		finder.showGrep()
	}
	finder.Draw()

	return finder, nil
}

func (finder *Finder) handleListUpdates() {
	for newList := range finder.listUpdateChan {
		finder.listUpdateMutex.Lock()
		helper.CopyListContent(newList, finder.searchedList)
		helper.CopyListContent(finder.searchedList, finder.previousSearchedList)
		finder.listDirty = true
		finder.listUpdateMutex.Unlock()
		finder.context.App.QueueUpdateDraw(func() {
			finder.setCurrentLine(0)
			finder.Draw()
		})
	}
}

func (finder *Finder) setCurrentLine(lineIndex int) error {
	if finder.searchedList.GetItemCount() == 0 {
		// Nothing selected (e.g. grep before a query is typed): clear the preview.
		finder.selectedList = tview.NewTextArea().SetText("", false)
		return nil
	}
	if lineIndex < 0 {
		lineIndex = 0
	}
	if lineIndex >= finder.searchedList.GetItemCount() {
		lineIndex = finder.searchedList.GetItemCount() - 1
	}
	finder.searchedList.SetCurrentItem(lineIndex)

	_, selectedName := finder.searchedList.GetItemText(lineIndex)
	return finder.setSelectedDirectory(helper.GetAbsFilePath(selectedName, finder.context.CurrentPath))
}

// setSelectedDirectory updates the selected directory/file preview
func (finder *Finder) setSelectedDirectory(selectedPath string) error {
	selectedAbsolutePath, _ := filepath.Abs(selectedPath)
	isFileNotFound := helper.IsFileNotFound(selectedAbsolutePath)
	if isFileNotFound {
		finder.selectedList = tview.NewTextArea().SetText("Not found...", false)
		return nil
	}
	isDirEmpty, _ := helper.IsDirectoryEmpty(selectedAbsolutePath)
	if isDirEmpty {
		finder.selectedList = tview.NewTextArea().SetText("Directory is empty...", false)
		return nil
	}
	selectedDirectoryIndex := 0

	directoryLoader := loader.NewDirectoryLoader(finder.context.ShowHiddenFiles, false, []string{})
	newSelectedList, err := directoryLoader.LoadDirectory(selectedPath)
	if err != nil {
		return err
	}

	if newSelectedList == nil {
		finder.selectedList, err = helper.LoadFilePreview(selectedPath, &finder.searchTerm, constNumSearchLines)
		if err != nil {
			return err
		}
	} else {
		newSelectedList.SetCurrentItem(selectedDirectoryIndex)
		finder.selectedList = newSelectedList
	}
	return nil
}

func (finder *Finder) handleFooterInput() {
	finder.footer = tview.NewInputField().SetText("/")
	finder.footer.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		return event
	})
	helper.CopyListContent(finder.fileList, finder.searchedList)
	finder.footer.SetChangedFunc(
		func(text string) {
			defer finder.Draw()
			currentInput := strings.TrimPrefix(text, "/")
			switch {
			case finder.isGrep:
				go finder.runGrep(currentInput)
			case finder.isFind:
				go finder.runFindFilter(currentInput)
			default:
				go finder.manageFuzzySearch(currentInput)
			}
		},
	)
	finder.currentFocusedWidget = finder.footer
}

func (finder *Finder) manageFuzzySearch(text string) {
	// Signal any existing search to stop
	select {
	case finder.fuzzySearchQuit <- true:
	default:
	}

	// Check if we should stop before starting
	select {
	case <-finder.fuzzySearchQuit:
		return
	default:
		finder.fuzzySearch(text)
	}
}

func (finder *Finder) fuzzySearch(text string) {
	// Create new list with matches
	text, _ = strings.CutSuffix(text, " ")
	searchTerms := strings.FieldsFunc(text, func(r rune) bool { return r == ' ' || r == '/' })
	startHighlightMarker := "[red::b]"
	endHighlightMarker := "[-::-]"
	newList := tview.NewList().ShowSecondaryText(false)
	itemCount := len(finder.fileListPrimary)
	searchHits := slices.Repeat([]int{0}, itemCount)
	lineIdxs := make([]int, itemCount)
	for i := 0; i < itemCount; i++ {
		lineIdxs[i] = i
	}
	indeces := make(map[int][][]int, itemCount)
	for _, searchTerm := range searchTerms {
		for i := 0; i < itemCount; i++ {
			primaryText := finder.fileListPrimary[i]
			idxs := gostring.IndexAllIgnoreCase(primaryText, searchTerm, -1)
			if val, exists := indeces[i]; !exists {
				indeces[i] = idxs
			} else {
				indeces[i] = append(val, idxs...)
			}
			for _, idx := range idxs {
				searchHits[i] = searchHits[i] + len(idx)
			}
		}
	}

	sort.SliceStable(lineIdxs, func(i, j int) bool {
		return searchHits[lineIdxs[i]] > searchHits[lineIdxs[j]]
	})
	for j := 0; j < itemCount; j++ {
		displayName := finder.fileListPrimary[lineIdxs[j]]
		secondaryText := finder.fileListSecondary[lineIdxs[j]]
		index := indeces[lineIdxs[j]]
		newList.AddItem(gostring.HighlightString(displayName, index, startHighlightMarker, endHighlightMarker), secondaryText, 0, nil)
	}
	// Send the new list through the channel
	select {
	case finder.listUpdateChan <- newList:
	default:
		// If channel is full, skip this update
	}
	finder.searchTerm = text
}

// beginSearch starts a new external-search generation: it cancels any in-flight
// search process and returns this search's generation number and context. Only
// the most recent generation is allowed to publish results (see sendSearchList),
// so a slow, stale search never overwrites a newer one.
func (finder *Finder) beginSearch() (int, context.Context) {
	finder.searchMu.Lock()
	finder.searchGen++
	gen := finder.searchGen
	if finder.searchCancel != nil {
		finder.searchCancel()
	}
	ctx, cancel := context.WithCancel(context.Background())
	finder.searchCancel = cancel
	finder.searchMu.Unlock()
	return gen, ctx
}

// sendSearchList publishes a result list only if it is still the most recent
// search, so out-of-order completions don't clobber newer results.
func (finder *Finder) sendSearchList(gen int, newList *tview.List) {
	finder.searchMu.Lock()
	latest := gen == finder.searchGen
	finder.searchMu.Unlock()
	if !latest {
		return
	}
	select {
	case finder.listUpdateChan <- newList:
	default:
	}
}

func nonEmptyLines(out []byte) []string {
	var lines []string
	for _, line := range strings.Split(strings.TrimRight(string(out), "\n"), "\n") {
		if line != "" {
			lines = append(lines, line)
		}
	}
	return lines
}

// highlightTerms marks every occurrence of each term within text using the same
// markers the list-shaping code recognises ([red::b] is preserved by
// ShortenPathsIfNecessary), so matches stand out in the left candidate list.
func highlightTerms(text string, terms []string) string {
	var locs [][]int
	for _, term := range terms {
		locs = append(locs, gostring.IndexAllIgnoreCase(text, term, -1)...)
	}
	if len(locs) == 0 {
		return text
	}
	return gostring.HighlightString(text, locs, "[red::b]", "[-::-]")
}

// isNoMatch reports whether a ripgrep/fzf error is just "no matches" (exit 1),
// as opposed to a real failure (missing binary, bad args, exit 2).
func isNoMatch(err error) bool {
	exitErr, ok := err.(*exec.ExitError)
	return ok && exitErr.ExitCode() == 1
}

// loadFindCandidates caches the file list for find mode via `rg --files`, which
// is fast and respects .gitignore. --hidden opts hidden files back in.
func (finder *Finder) loadFindCandidates() {
	args := []string{"--files"}
	if finder.context.ShowHiddenFiles {
		args = append(args, "--hidden")
	}
	cmd := exec.Command("rg", args...)
	cmd.Dir = finder.context.CurrentPath
	out, err := cmd.Output()
	if err != nil {
		finder.findCandidates = nil
		return
	}
	finder.findCandidates = nonEmptyLines(out)
}

// runFindFilter is the find-mode search: it ranks the cached file list against
// the query with fzf's algorithm in non-interactive (--filter) mode, so the left
// list is sorted by match likelihood exactly like the interactive fzf finder.
func (finder *Finder) runFindFilter(query string) {
	finder.searchTerm = query
	gen, ctx := finder.beginSearch()

	newList := tview.NewList().ShowSecondaryText(false)
	var results []string
	if strings.TrimSpace(query) == "" {
		// No query: show the (unranked) candidate list.
		results = finder.findCandidates
	} else {
		// fzf's extended search treats space-separated terms as AND (each term
		// fuzzy-matched), so "test abc" keeps only paths matching both.
		cmd := exec.CommandContext(ctx, "fzf", "--filter", query)
		cmd.Dir = finder.context.CurrentPath
		cmd.Stdin = strings.NewReader(strings.Join(finder.findCandidates, "\n"))
		out, err := cmd.Output()
		if ctx.Err() != nil {
			return
		}
		if err != nil {
			if isNoMatch(err) {
				finder.sendSearchList(gen, newList) // no matches
				return
			}
			newList.AddItem("[red]fzf unavailable or errored[white]", "", 0, nil)
			finder.sendSearchList(gen, newList)
			return
		}
		results = nonEmptyLines(out) // fzf prints best-first
	}

	terms := strings.Fields(query)
	for i, line := range results {
		if i >= constMaxResults {
			break
		}
		newList.AddItem(highlightTerms(line, terms), line, 0, nil)
	}
	finder.sendSearchList(gen, newList)
}

// rgCountMatches runs ripgrep for a single term and returns a path->match-count
// map. --fixed-strings + --smart-case mirrors the old substring, case-insensitive
// match and avoids regex errors on partially-typed queries; ripgrep respects
// .gitignore by default, and --hidden opts hidden files back in when requested.
func (finder *Finder) rgCountMatches(ctx context.Context, term string) (map[string]int, error) {
	args := []string{"--color=never", "--fixed-strings", "--smart-case", "--count-matches"}
	if finder.context.ShowHiddenFiles {
		args = append(args, "--hidden")
	}
	args = append(args, "--", term)
	cmd := exec.CommandContext(ctx, "rg", args...)
	cmd.Dir = finder.context.CurrentPath
	out, err := cmd.Output()
	if err != nil {
		return nil, err
	}
	// Parse "path:count" lines. count is after the final colon (paths may contain
	// colons themselves).
	counts := make(map[string]int)
	for _, line := range nonEmptyLines(out) {
		sep := strings.LastIndex(line, ":")
		if sep < 0 {
			continue
		}
		count, _ := strconv.Atoi(line[sep+1:])
		counts[line[:sep]] = count
	}
	return counts, nil
}

// runGrep is the grep-mode search. Instead of pre-loading every file's content
// into memory, it shells out to ripgrep. Space-separated terms are ANDed: a file
// is kept only if it contains every term, and results are sorted by total match
// count (descending) so the most relevant files come first — matching
// gofileyourself's original hit-count ranking.
func (finder *Finder) runGrep(query string) {
	finder.searchTerm = query
	gen, ctx := finder.beginSearch()

	newList := tview.NewList().ShowSecondaryText(false)
	terms := strings.Fields(query)
	if len(terms) == 0 {
		// No query yet: show the file list, matching find's initial view.
		for i, line := range finder.findCandidates {
			if i >= constMaxResults {
				break
			}
			newList.AddItem(line, line, 0, nil)
		}
		finder.sendSearchList(gen, newList)
		return
	}

	// Intersect the per-term matches, summing counts.
	var counts map[string]int
	for i, term := range terms {
		termCounts, err := finder.rgCountMatches(ctx, term)
		if ctx.Err() != nil {
			// Superseded by a newer search (process was cancelled); drop results.
			return
		}
		if err != nil {
			if isNoMatch(err) {
				finder.sendSearchList(gen, newList) // a term matched nothing => no results
				return
			}
			newList.AddItem("[red]ripgrep (rg) unavailable or errored[white]", "", 0, nil)
			finder.sendSearchList(gen, newList)
			return
		}
		if i == 0 {
			counts = termCounts
			continue
		}
		for path := range counts {
			if c, ok := termCounts[path]; ok {
				counts[path] += c
			} else {
				delete(counts, path)
			}
		}
		if len(counts) == 0 {
			finder.sendSearchList(gen, newList)
			return
		}
	}

	type grepMatch struct {
		path  string
		count int
	}
	matches := make([]grepMatch, 0, len(counts))
	for path, count := range counts {
		matches = append(matches, grepMatch{path: path, count: count})
	}
	// Sort by count desc, then path asc so equal-count order is deterministic
	// (map iteration order is not) and the list doesn't flicker between redraws.
	sort.SliceStable(matches, func(i, j int) bool {
		if matches[i].count != matches[j].count {
			return matches[i].count > matches[j].count
		}
		return matches[i].path < matches[j].path
	})

	for i, m := range matches {
		if i >= constMaxResults {
			break
		}
		newList.AddItem(highlightTerms(m.path, terms), m.path, 0, nil)
	}
	finder.sendSearchList(gen, newList)
}

func (finder *Finder) resetFileList() error {
	finder.fileList.Clear()
	directoryLoader := loader.NewDirectoryLoader(finder.context.ShowHiddenFiles, true, []string{})
	fileList, err := directoryLoader.LoadDirectory(finder.context.CurrentPath)
	if err != nil {
		return err
	}
	finder.fileList = fileList
	finder.cacheFileListTexts()
	helper.CopyListContent(finder.fileList, finder.searchedList)
	helper.CopyListContent(finder.searchedList, finder.previousSearchedList)
	return nil
}

func (finder *Finder) searchInDirectory() error {
	finder.setCurrentLine(0)
	finder.handleFooterInput()
	return nil
}

func (finder *Finder) Root() tview.Primitive {
	return finder.rootFlex
}

func (finder *Finder) Draw() {
	finder.rootFlex.Clear()
	listFlex := tview.NewFlex()
	listFlex.AddItem(finder.searchedList, 0, 1, true)
	if finder.selectedList != nil {
		listFlex.AddItem(tview.NewBox(), 2, 0, false)
		listFlex.AddItem(finder.selectedList, 0, 1, true)
	}
	finder.rootFlex.SetDirection(tview.FlexRow)
	if finder.footer != nil {
		finder.rootFlex.AddItem(finder.footer, 3, 0, false)
	}
	finder.rootFlex.AddItem(listFlex, 0, 1, true)
	finder.context.App.SetFocus(finder.currentFocusedWidget)
	finder.applyTheme()
}

func (finder *Finder) Run() error {
	return finder.context.App.SetRoot(finder.Root(), true).Run()
}

func (finder *Finder) GetInputCapture() func(*tcell.EventKey) *tcell.EventKey {
	return finder.rootFlex.GetInputCapture()
}

func (finder *Finder) showRecentHistory() {
	finder.title = "Find Recent"
	finder.fileList.Clear()
	historyPath := filepath.Join(os.Getenv("HOME"), ".gofileyourselfhistory")
	data := make(map[string]int)
	file, _ := os.ReadFile(historyPath)
	json.Unmarshal(file, &data)
	recentFiles := make([]string, 0, len(data))
	for k := range data {
		recentFiles = append(recentFiles, k)
	}
	sort.SliceStable(recentFiles, func(i, j int) bool {
		return data[recentFiles[i]] < data[recentFiles[j]]
	})
	for _, fileName := range recentFiles {
		// Display relative to the current directory (like find/grep), but keep the
		// absolute path as the item value so opening/preview work for files that
		// live outside the current directory.
		display := fileName
		if rel, err := filepath.Rel(finder.context.CurrentPath, fileName); err == nil {
			display = rel
		}
		finder.fileList.InsertItem(0, display, fileName, 0, nil)
	}
	finder.cacheFileListTexts()
	helper.CopyListContent(finder.fileList, finder.searchedList)
	helper.CopyListContent(finder.searchedList, finder.previousSearchedList)
	finder.searchInDirectory()
}

func (finder *Finder) showGrep() {
	finder.title = "Grep"
	finder.isGrep = true
	// Grep no longer pre-loads file contents; ripgrep searches on each keystroke.
	// Show the current directory's file list initially (like find), then switch to
	// content matches once the user types a query.
	finder.loadFindCandidates()
	finder.fileList.Clear()
	finder.searchedList.Clear()
	finder.previousSearchedList.Clear()
	finder.searchInDirectory()
	go finder.runGrep("")
}
