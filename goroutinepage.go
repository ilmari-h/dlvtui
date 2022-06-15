package main

import (
	"fmt"

	"github.com/gdamore/tcell/v2"
	"github.com/go-delve/delve/service/api"
	"github.com/rivo/tview"
)

// TODO: on select just call "switchgoroutine"
// in addition, map debugger state behind current goroutine ID

type GoroutinePage struct {
	commandHandler *CommandHandler
	listView       *tview.List
	widget         *tview.Frame
}

func NewGoroutinePage() *GoroutinePage {
	listView := tview.NewList()
	listView.SetBackgroundColor(tcell.ColorDefault)
	pageFrame := tview.NewFrame(listView).
		SetBorders(0, 0, 0, 0, 0, 0).
		AddText("[::b]Goroutines:", true, tview.AlignLeft, tcell.ColorWhite)
	pageFrame.SetBackgroundColor(tcell.ColorDefault)
	gp := GoroutinePage{
		listView: listView,
		widget:   pageFrame,
	}
	return &gp
}

func (page *GoroutinePage) SetCommandHandler(ch *CommandHandler) {
	page.commandHandler = ch
}

func (page *GoroutinePage) RenderGoroutines(grs []*api.Goroutine, curr *api.Goroutine) {
	page.listView.Clear()
	selectedI := 0
	for i, gor := range grs {
		if curr != nil && curr.ID == gor.ID {
			selectedI = i
		}
		page.listView.AddItem(
			fmt.Sprint(gor.ID),
			fmt.Sprintf("%s[white]:%d", gor.CurrentLoc.File, gor.CurrentLoc.Line),
			rune(48+i),
			nil).
			SetSelectedFunc(func(i int, s1, s2 string, r rune) {
				//sp.commandHandler.RunCommand(&OpenFile{
				//	File:   grs[i].File,
				//	AtLine: grs[i].Line - 1,
				//})
			})
	}
	page.listView.SetCurrentItem(selectedI)
}

func (sp *GoroutinePage) GetWidget() tview.Primitive {
	return sp.widget
}

func (sp *GoroutinePage) GetName() string {
	return "goroutines"
}

func (sp *GoroutinePage) HandleKeyEvent(event *tcell.EventKey) *tcell.EventKey {
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
