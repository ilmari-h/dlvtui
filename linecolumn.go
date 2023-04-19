package main

import (
	"github.com/ilmari-h/dlvtui/nav"
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

func NewLineColumn(navState *nav.Nav) *LineColumn {
	return &LineColumn{
		navState: navState,
		textView: tview.NewTextView(),
	}
}

func (lc *LineColumn) SetWidth(nw int) {
	lc.width = nw
}

func (lc *LineColumn) Render(lineStart, lineEnd, current int) {
	lc.textView.SetBackgroundColor(iToColorTcell(gConfig.Colors.LineColumnBg))

	if lc.navState == nil || lc.navState.CurrentFile == nil {
		return
	}

	// Set line numbers in gutter.
	lineNumbers := ""
	breakpoints := lc.navState.Breakpoints[lc.navState.CurrentFile.Path]
	for i := lineStart; i <= lineEnd; i++ {
		bp := " "
		if fbp, ok := breakpoints[i]; ok && fbp.ID >= 0 {
			if breakpoints[i].Disabled {
				bp = fmt.Sprintf("[%s]%s[-::-]",
					iToColorS(gConfig.Colors.BpFg),
					gConfig.Icons.BpDisabled,
				)
			} else if i == lc.navState.CurrentDebuggerPos.Line &&
				lc.navState.CurrentFile.Path == lc.navState.CurrentDebuggerPos.File {
				bp = fmt.Sprintf("[%s:%s]%s[-::-]",
					iToColorS(gConfig.Colors.BpActiveFg),
					iToColorS(gConfig.Colors.LineActiveBg),
					gConfig.Icons.BpActive,
				)
			} else {
				bp = fmt.Sprintf("[%s]%s[-::-]",
					iToColorS(gConfig.Colors.BpFg),
					gConfig.Icons.Bp,
				)
			}
		}

		// TODO: Also: if some stack frame has this line
		if i == lc.navState.CurrentDebuggerPos.Line &&
			lc.navState.CurrentFile.Path == lc.navState.CurrentDebuggerPos.File {

			lineNumbers += fmt.Sprintf(`[%s:%s]%s[%s]%*d [-:-:-]`,
				iToColorS(gConfig.Colors.LineSelectedFg),
				iToColorS(gConfig.Colors.LineActiveBg),
				bp,
				iToColorS(gConfig.Colors.LineSelectedFg),
				lc.width-2,
				i,
			)

		} else if i == current {
			lineNumbers += fmt.Sprintf(`[%s:%s]%s[%s]%*d [-:-:-]`,
				iToColorS(gConfig.Colors.LineSelectedFg),
				iToColorS(gConfig.Colors.LineSelectedBg),
				bp,
				iToColorS(gConfig.Colors.LineSelectedFg),
				lc.width-2,
				i,
			)
		} else {
			lineNumbers += fmt.Sprintf(`[%s]%s%*d `,
				iToColorS(gConfig.Colors.LineFg),
				bp,
				lc.width-2,
				i,
			)
		}
	}
	lc.textView.SetText(lineNumbers)
}

func (lc *LineColumn) GetTextView() *tview.TextView {
	return lc.textView
}
