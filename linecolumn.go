package main

import (
	"fmt"

	"dlvtui/nav"

	"github.com/rivo/tview"
)

type LineColumn struct {
	width int
	navState *nav.Nav
	textView *tview.TextView
}

func NewLineColumn(width int, navState *nav.Nav) *LineColumn {
	return &LineColumn{
		width: width,
		navState: navState,
		textView: tview.NewTextView(),
	}
}

func (lc *LineColumn) Render(lineStart int, lineEnd int) {
	// Set line numbers in gutter.
	lineNumbers := ""
	breakpoints := lc.navState.Breakpoints[lc.navState.CurrentFile.Path]
	for i := lineStart; i <= lineEnd; i++ {
		bp := " "
		if lc.navState.CurrentBreakpoint != nil && lc.navState.CurrentBreakpoint.Line == i {
			bp = "●"
		} else if _, ok := breakpoints[i] ; ok {
			bp = "○"
		}

		if( i == lineStart) { // TODO: use current line
			lineNumbers += fmt.Sprintf(`[black:white]%s%*d [-:-:-]`,bp, lc.width -2, i)
		} else {
			lineNumbers += fmt.Sprintf(`%s%*d `,bp,lc.width - 2, i)
		}
	}
	lc.textView.SetText(lineNumbers)
}

func (lc *LineColumn) GetTextView() *tview.TextView {
	return lc.textView
}
