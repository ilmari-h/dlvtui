package ui

import (
	"fmt"

	"github.com/rivo/tview"
)

type PerfTextView struct {
	lineIndices []int
	currentViewHeight int
	fullText string
	gutterColumn *GutterColumn
	tview.TextView
}

func NewPerfTextView() *PerfTextView {
	pv := &PerfTextView{ []int{}, 1, "", nil, *tview.NewTextView() }
	pv.SetScrollable(false)
	return pv
}

func (perfTextView *PerfTextView) ScrollTo(line int) {
	_,_,_,h := perfTextView.GetInnerRect()
	perfTextView.currentViewHeight = h - 1
	perfTextView.SetMaxLines(h)

	startIdx := perfTextView.lineIndices[line]
	var endIdx = perfTextView.lineIndices[ len(perfTextView.lineIndices) - 1 ]
	if line + perfTextView.currentViewHeight < len(perfTextView.lineIndices) {
		endIdx = perfTextView.lineIndices[line + perfTextView.currentViewHeight]
	}
	perfTextView.SetText(perfTextView.fullText[startIdx:endIdx])

	// Set line numbers in gutter.
	lineNumbers := ""
	for i := line + 1; i <= line + perfTextView.currentViewHeight; i++ {
		if( i == line + 1) {
			lineNumbers += fmt.Sprintf(`[black:white] %3d [-:-:-]`, i)
		} else {
			lineNumbers += fmt.Sprintf(` %3d `, i)
		}
	}
	perfTextView.gutterColumn.SetText(lineNumbers)

}

func (perfTextView *PerfTextView) SetTextP(text string, lineIndices []int) {
	perfTextView.fullText = text
	perfTextView.lineIndices = lineIndices
	_,_,_,h := perfTextView.GetInnerRect()
	perfTextView.currentViewHeight = h - 1
	perfTextView.SetMaxLines(h)

	strIndx := lineIndices[perfTextView.currentViewHeight]
	perfTextView.SetText(text[:strIndx])
}

func (perfTextView *PerfTextView) SetGutterColumn(column *GutterColumn) {
	perfTextView.gutterColumn = column
}

func (perfTextView *PerfTextView) GetGutterColumn() *GutterColumn {
	return perfTextView.gutterColumn
}
