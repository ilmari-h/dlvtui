package main

import (
	"math"

	"github.com/rivo/tview"
	log "github.com/sirupsen/logrus"
)

type PerfTextView struct {
	lineIndices       []int
	currentViewHeight int
	virtualLine       int
	scroll            int
	fullText          string
	lineColumn        *LineColumn
	tview.TextView
}

func NewPerfTextView() *PerfTextView {
	pv := &PerfTextView{[]int{}, 1, 1, 1, "", nil, *tview.NewTextView()}
	pv.SetScrollable(false)
	return pv
}

var offset = 5

func (perfTextView *PerfTextView) ReRender() {

	_, _, _, h := perfTextView.GetInnerRect()
	perfTextView.render(perfTextView.scroll, perfTextView.virtualLine, h)
}

func (perfTextView *PerfTextView) ReRenderWithHeight(height int) {
	perfTextView.render(perfTextView.scroll, perfTextView.virtualLine, height)
}

func (perfTextView *PerfTextView) render(scroll int, line int, maxHeight int) {

	// No text loaded, don't render.
	if len(perfTextView.lineIndices) == 0 {
		return
	}

	firstLine := perfTextView.lineIndices[scroll]
	var endIdx = perfTextView.lineIndices[len(perfTextView.lineIndices)-1]
	if scroll+maxHeight < len(perfTextView.lineIndices) {
		endIdx = perfTextView.lineIndices[scroll+maxHeight] - 1
	}

	perfTextView.lineColumn.Render(scroll+1, scroll+maxHeight, perfTextView.virtualLine)
	perfTextView.SetText(perfTextView.fullText[firstLine:endIdx])
	perfTextView.lineColumn.textView.ScrollToBeginning()
	perfTextView.ScrollToBeginning()

}

func (perfTextView *PerfTextView) scrollTo(line int, center bool) {
	perfTextView.virtualLine = line + 1 // File line numbers start at 1.

	if center { // Scroll selected line at the center of the view.
		perfTextView.scroll = int(math.Max(0,
			float64(line)-math.Max(1, float64(perfTextView.currentViewHeight/2)),
		))
		log.Debugf("Jumping to line %d", perfTextView.scroll)
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
	perfTextView.currentViewHeight = h
	perfTextView.SetMaxLines(h)

	perfTextView.render(scroll, perfTextView.virtualLine, perfTextView.currentViewHeight)
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
	perfTextView.SetText(perfTextView.fullText[:strIndx])
}

func (perfTextView *PerfTextView) SetLineColumn(column *LineColumn) {
	perfTextView.lineColumn = column
}
