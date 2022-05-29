package ui

import (
	"fmt"
	"github.com/rivo/tview"
)

type GutterColumn interface {
	Render(lineStart int, lineEnd int)
	GetTextView() *tview.TextView
}

type DefaultGC struct {
	textView *tview.TextView
}

func NewGutterColumn() *DefaultGC {

	return &DefaultGC{ textView: tview.NewTextView() }
}

func (gc *DefaultGC) Render(lineStart int, lineEnd int) {
	// Set line numbers in gutter.
	lineNumbers := ""
	for i := lineStart; i <= lineEnd; i++ {
		if( i == lineStart) {
			lineNumbers += fmt.Sprintf(`[black:white] %3d [-:-:-]`, i)
		} else {
			lineNumbers += fmt.Sprintf(` %3d `, i)
		}
	}
	gc.textView.SetText(lineNumbers)
}

func (gc *DefaultGC) GetTextView() *tview.TextView {
	return gc.textView
}
