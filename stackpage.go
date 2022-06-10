package main

import (
	"fmt"

	"github.com/gdamore/tcell/v2"
	"github.com/go-delve/delve/service/api"
	"github.com/rivo/tview"
)

type StackPage struct {
	commandHandler *CommandHandler
	listView *tview.List
}

func NewStackPage() *StackPage {
	sp := StackPage{
		listView: tview.NewList(),
	}
	sp.listView.SetBackgroundColor(tcell.ColorDefault)
	return &sp
}

func (sv *StackPage) RenderStack(stack []api.Stackframe) {
	sv.listView.Clear()
	for i, frame := range stack {
		sv.listView.AddItem(
			frame.Function.Name(),
			fmt.Sprintf("%s[white]:%d",frame.File,frame.Line),
			rune(48+i),
		nil)
	}
}

func (sv *StackPage) GetWidget() tview.Primitive {
	return sv.listView
}

func (sv *StackPage) GetName() string {
	return "stack"
}

func (page *StackPage) SetCommandHandler(ch *CommandHandler) {
	page.commandHandler = ch
}

func (sv *StackPage) HandleKeyEvent(event *tcell.EventKey) *tcell.EventKey {
	handler := sv.listView.InputHandler()
	handler(event, func(p tview.Primitive) {})
	return nil
}
