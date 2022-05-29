package ui

import (

	"github.com/rivo/tview"
)

type PerfTextView struct {
	lineIndices []int
	currentViewHeight int
	fullText string
	gutterColumn GutterColumn
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

	perfTextView.gutterColumn.Render(line+1, line+perfTextView.currentViewHeight)

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

func (perfTextView *PerfTextView) SetGutterColumn(column GutterColumn) {
	perfTextView.gutterColumn = column
}

func (perfTextView *PerfTextView) GetGutterColumn() *tview.TextView {
	return perfTextView.gutterColumn.GetTextView()
}
