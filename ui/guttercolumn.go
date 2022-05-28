package ui

import (
	"github.com/rivo/tview"
)

type GutterColumn struct {
	tview.TextView
}

func NewGutterColumn() *GutterColumn {
	return &GutterColumn{ *tview.NewTextView() }
}

