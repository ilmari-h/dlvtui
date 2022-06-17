package main

import (
	"fmt"
	"strings"

	"github.com/gdamore/tcell/v2"
	"github.com/go-delve/delve/service/api"
	"github.com/rivo/tview"
)

type StackPage struct {
	commandHandler *CommandHandler
	listView       *tview.List
	widget         *tview.Frame
}

func NewStackPage() *StackPage {
	listView := tview.NewList()
	listView.SetBackgroundColor(tcell.ColorDefault)
	listView.SetSelectedBackgroundColor(tcell.ColorBlack)
	listView.SetSelectedTextColor(tcell.ColorWhite)

	pageFrame := tview.NewFrame(listView).
		SetBorders(0, 0, 0, 0, 0, 0).
		AddText("[::b]Call stack:", true, tview.AlignLeft, tcell.ColorWhite)
	pageFrame.SetBackgroundColor(tcell.ColorDefault)

	sp := StackPage{
		listView: listView,
		widget:   pageFrame,
	}
	return &sp
}

func (sp *StackPage) RenderStack(stack []api.Stackframe, curr *api.Stackframe) {
	sp.listView.Clear()
	selectedI := 0
	for i, frame := range stack {
		if curr.Line == frame.Line && curr.File == frame.File {
			selectedI = i
		}

		// Format header
		fullName := frame.Function.Name()
		dotIdx := strings.Index(fullName, ".")
		pkgName := fullName[:dotIdx]
		functionName := fullName[dotIdx:]
		header := fmt.Sprintf("[white::b]%s[blue::-]%s", pkgName, functionName)

		sp.listView.AddItem(
			header,
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
	return sp.widget
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
