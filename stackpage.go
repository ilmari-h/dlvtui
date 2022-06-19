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

	selectedStyle := tcell.StyleDefault.
		Foreground(iToColorTcell(gConfig.Colors.LineFg)).
		Background(iToColorTcell(gConfig.Colors.ListSelectedBg)).
		Attributes(tcell.AttrBold)

	listView.SetSelectedStyle(selectedStyle)
	listView.SetInputCapture(listInputCaptureC)

	pageFrame := tview.NewFrame(listView).
		SetBorders(0, 0, 0, 0, 0, 0).
		AddText("[::b]Call stack:", true, tview.AlignLeft, iToColorTcell(gConfig.Colors.HeaderFg))
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
		header := fmt.Sprintf("[%s::-]%s[%s::-]%s",
			iToColorS(gConfig.Colors.VarValueFg),
			pkgName,
			iToColorS(gConfig.Colors.VarTypeFg),
			functionName,
		)

		sp.listView.AddItem(
			header,
			fmt.Sprintf("[%s]%s[white]:%d",
				iToColorS(gConfig.Colors.VarNameFg),
				frame.File,
				frame.Line,
			),
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
	if keyPressed(event, gConfig.Keys.LineDown) {
		sp.listView.SetCurrentItem(sp.listView.GetCurrentItem() + 1)
		return nil
	}
	if keyPressed(event, gConfig.Keys.LineUp) {
		if sp.listView.GetCurrentItem() > 0 {
			sp.listView.SetCurrentItem(sp.listView.GetCurrentItem() - 1)
		}
		return nil
	}
	sp.listView.InputHandler()(event, func(p tview.Primitive) {})
	return nil
}
