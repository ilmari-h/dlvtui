package main

import (
	"log"
	"math"

	"github.com/rivo/tview"
)

type PerfTextView struct {
	lineIndices       []int
	currentViewHeight int
	virtualLine       int
	scroll            int
	fullText          string
	gutterColumn      GutterColumn
	tview.TextView
}

func NewPerfTextView() *PerfTextView {
	pv := &PerfTextView{[]int{}, 1, 1, 1, "", nil, *tview.NewTextView()}
	pv.SetScrollable(false)
	return pv
}

var offset = 5

func (perfTextView *PerfTextView) ReRender() {
	perfTextView.gutterColumn.Render(perfTextView.scroll+1, perfTextView.scroll+perfTextView.currentViewHeight, perfTextView.virtualLine)
}

func (perfTextView *PerfTextView) scrollTo(line int, center bool) {
	perfTextView.virtualLine = line + 1 // File line numbers start at 1.

	if center { // Scroll selected line at the center of the view.
		perfTextView.scroll = int(math.Max(0,
			float64(line)-math.Max(1, float64(perfTextView.currentViewHeight/2)),
		))
		log.Printf("Jumping to line %d", perfTextView.scroll)
	} else {
		// Scroll down
		if perfTextView.virtualLine > offset &&
			perfTextView.scroll+perfTextView.currentViewHeight-perfTextView.virtualLine < offset {

			perfTextView.scroll++
		}
		// Scroll up
		if perfTextView.scroll != 0 && perfTextView.virtualLine-perfTextView.scroll < offset {
			perfTextView.scroll--
		}
	}
	scroll := perfTextView.scroll

	_, _, _, h := perfTextView.GetInnerRect()
	perfTextView.currentViewHeight = h - 1
	perfTextView.SetMaxLines(h)

	// Index out of bounds!!
	startIdx := perfTextView.lineIndices[scroll]
	var endIdx = perfTextView.lineIndices[len(perfTextView.lineIndices)-1]
	if scroll+perfTextView.currentViewHeight < len(perfTextView.lineIndices) {
		endIdx = perfTextView.lineIndices[scroll+perfTextView.currentViewHeight]
	}

	perfTextView.gutterColumn.Render(scroll+1, scroll+perfTextView.currentViewHeight, perfTextView.virtualLine)
	perfTextView.SetText(perfTextView.fullText[startIdx:endIdx])
}

func (perfTextView *PerfTextView) ScrollTo(line int) {
	perfTextView.scrollTo(line, false)
}

func (perfTextView *PerfTextView) JumpTo(line int) {
	perfTextView.scrollTo(line, true)
}

func (perfTextView *PerfTextView) SetTextP(text string, lineIndices []int) {
	perfTextView.fullText = text
	perfTextView.lineIndices = lineIndices
	_, _, _, h := perfTextView.GetInnerRect()
	perfTextView.currentViewHeight = h - 1
	perfTextView.SetMaxLines(h)

	var strIndx = 0
	if len(lineIndices) > perfTextView.currentViewHeight {
		strIndx = lineIndices[perfTextView.currentViewHeight]
	} else {
		strIndx = len(lineIndices) - 1
	}
	perfTextView.SetText(text[:strIndx])
}

func (perfTextView *PerfTextView) SetGutterColumn(column GutterColumn) {
	perfTextView.gutterColumn = column
}

func (perfTextView *PerfTextView) GetGutterColumn() *tview.TextView {
	return perfTextView.gutterColumn.GetTextView()
}
