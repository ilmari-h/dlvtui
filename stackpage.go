package main

import (
	"fmt"

	"github.com/gdamore/tcell/v2"
	"github.com/go-delve/delve/service/api"
	"github.com/rivo/tview"
)

type StackPage struct {
	commandHandler *CommandHandler
	listView       *tview.List
}

func NewStackPage() *StackPage {
	sp := StackPage{
		listView: tview.NewList(),
	}
	sp.listView.SetBackgroundColor(tcell.ColorDefault)
	return &sp
}

func (sp *StackPage) RenderStack(stack []api.Stackframe, curr *api.Stackframe) {
	sp.listView.Clear()
	selectedI := 0
	for i, frame := range stack {
		if curr.Line == frame.Line && curr.File == frame.File {
			selectedI = i
		}
		sp.listView.AddItem(
			frame.Function.Name(),
			fmt.Sprintf("%s[white]:%d", frame.File, frame.Line),
			rune(48+i),
			nil).
			SetSelectedFunc(func(i int, s1, s2 string, r rune) {
				sp.commandHandler.RunCommand(&OpenFile{
					File:   stack[i].File,
					AtLine: stack[i].Line - 1,
				})
			})
	}
	sp.listView.SetCurrentItem(selectedI)
}

func (sp *StackPage) GetWidget() tview.Primitive {
	return sp.listView
}

func (sp *StackPage) GetName() string {
	return "stack"
}

func (page *StackPage) SetCommandHandler(ch *CommandHandler) {
	page.commandHandler = ch
}

func (sp *StackPage) HandleKeyEvent(event *tcell.EventKey) *tcell.EventKey {
	rune := event.Rune()
	if rune == 'j' {
		sp.listView.SetCurrentItem(sp.listView.GetCurrentItem() + 1)
		return nil
	}
	if rune == 'k' {
		if sp.listView.GetCurrentItem() > 0 {
			sp.listView.SetCurrentItem(sp.listView.GetCurrentItem() - 1)
		}
		return nil
	}
	sp.listView.InputHandler()(event, func(p tview.Primitive) {})
	return nil
}
