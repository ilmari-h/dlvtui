package main

import (
	"dlvtui/nav"
	"fmt"
	"strconv"

	"github.com/rivo/tview"
)

type LineColumn struct {
	width    int
	navState *nav.Nav
	textView *tview.TextView
}

func getMaxLineColWidth(maxLine int) int {
	return len(strconv.Itoa(maxLine)) + 3
}

func NewLineColumn(width int, navState *nav.Nav) *LineColumn {
	return &LineColumn{
		navState: navState,
		textView: tview.NewTextView(),
	}
}

func (lc *LineColumn) SetWidth(nw int) {
	lc.width = nw
}

func (lc *LineColumn) Render(lineStart, lineEnd, current int) {
	if lc.navState == nil || lc.navState.CurrentFile == nil {
		return
	}

	// Set line numbers in gutter.
	lineNumbers := ""
	breakpoints := lc.navState.Breakpoints[lc.navState.CurrentFile.Path]
	for i := lineStart; i <= lineEnd; i++ {
		bp := " "
		if _, ok := breakpoints[i]; ok {
			if breakpoints[i].Disabled {
				bp = "[red]○[-]"
			} else {
				bp = "[red]●[-]"
			}
		}

		// TODO: Also: if some stack frame has this line
		if i == lc.navState.CurrentDebuggerPos.Line &&
			lc.navState.CurrentFile.Path == lc.navState.CurrentDebuggerPos.File {

			lineNumbers += fmt.Sprintf(`[black:green]%s[black]%*d [-:-:-]`, bp, lc.width-2, i)
		} else if i == current {
			lineNumbers += fmt.Sprintf(`[black:white]%s[black]%*d [-:-:-]`, bp, lc.width-2, i)
		} else {
			lineNumbers += fmt.Sprintf(`%s%*d `, bp, lc.width-2, i)
		}
	}
	lc.textView.SetText(lineNumbers)
}

func (lc *LineColumn) GetTextView() *tview.TextView {
	return lc.textView
}
