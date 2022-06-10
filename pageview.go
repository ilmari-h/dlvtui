package main

import (
	"dlvtui/nav"

	"github.com/gdamore/tcell/v2"
	"github.com/go-delve/delve/service/api"
	"github.com/rivo/tview"
)

type Page interface {
	GetName() string
	GetWidget() tview.Primitive
	HandleKeyEvent(event *tcell.EventKey) *tcell.EventKey
	SetCommandHandler(cmdHdlr *CommandHandler) // Each page will be given a reference to CommandHandler
}

type PageView struct {
	commandHandler *CommandHandler
	index int
	pages []Page
	pagesView *tview.Pages

	codePage *CodePage
	varsPage *VarsPage
	stackPage *StackPage
}

func NewPageView(cmdHdlr *CommandHandler, nav *nav.Nav, app *tview.Application) *PageView {
	pv := PageView{
		commandHandler: cmdHdlr,
		index: 0,
		pages: []Page{},
		pagesView: tview.NewPages(),
		codePage: NewCodePage(app, nav),
		varsPage: NewVarPage(),
		stackPage: NewStackPage(),
	}
	pv.pages = []Page{ pv.codePage, pv.varsPage, pv.stackPage }

	for _, p := range pv.pages {
		pv.pagesView.AddPage(p.GetName(),p.GetWidget(),true,true)
		p.SetCommandHandler(cmdHdlr)
	}
	pv.pagesView.SwitchToPage(pv.pages[0].GetName())

	return &pv
}

func (pv *PageView) CurrentPage() Page {
	return pv.pages[pv.index]
}

func (pv *PageView) RenderDebuggerMove( dbgMove *DebuggerMove ) {
	newState := dbgMove.DbgState
	line := newState.CurrentThread.Line
	// Variables in new state.
	var args []api.Variable
	var locals []api.Variable
	var globals []api.Variable
	var returns []api.Variable

	// If hit breakpoint.
	if newState.CurrentThread.BreakpointInfo != nil {

		locals = newState.CurrentThread.BreakpointInfo.Locals
		globals = newState.CurrentThread.BreakpointInfo.Variables
		args = newState.CurrentThread.BreakpointInfo.Arguments

	// If just step.
	} else if dbgMove.DbgStep != nil {
		locals = append(locals, dbgMove.DbgStep.locals...)
		args = append(args, dbgMove.DbgStep.args...)
	}
	returns = newState.CurrentThread.ReturnValues

	// Update pages.
	pv.varsPage.RenderDebuggerMove(args, locals, globals, returns)
	pv.codePage.perfTextView.scrollTo(line - 1, true)
	pv.stackPage.RenderStack(dbgMove.Stack)
}


func (pv *PageView) RefreshLineColumn() {
	pv.codePage.perfTextView.ReRender()
}

func (pv *PageView) GetWidget() tview.Primitive {
	return pv.pagesView
}

func (pv *PageView) LoadFile( file *nav.File ) {
	codePage := pv.pages[0].(*CodePage) // TODO: hardcode in constructor
	lineInNewFile := codePage.navState.EnterNewFile(file)
	pv.pagesView.SwitchToPage(codePage.GetName())
	codePage.perfTextView.SetTextP(file.Content, file.LineIndices)
	codePage.perfTextView.JumpTo(lineInNewFile)
	codePage.navState.SetLine(lineInNewFile)
}

// Consumes event if changing page. Otherwise delegates to active page.
func (pv *PageView) HandleKeyEvent(event *tcell.EventKey) *tcell.EventKey {
	rune := event.Rune()
	if rune == 'h' {
		if pv.index > 0 {
			pv.index--
			pv.pagesView.SwitchToPage(pv.CurrentPage().GetName())
		}
		return nil // Consume event
	} else if rune == 'l' {
		if pv.index < len(pv.pages) - 1 {
			pv.index++
			pv.pagesView.SwitchToPage(pv.CurrentPage().GetName())
		}
		return nil // Consume event
	}
	// Delegate
	return pv.CurrentPage().HandleKeyEvent(event)
}
