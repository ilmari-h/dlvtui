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
	index          int
	pages          []Page
	pagesView      *tview.Pages

	codePage      *CodePage
	varsPage      *VarsPage
	stackPage     *StackPage
	goroutinePage *GoroutinePage
}

func NewPageView(cmdHdlr *CommandHandler, nav *nav.Nav, app *tview.Application) *PageView {
	pv := PageView{
		commandHandler: cmdHdlr,
		index:          0,
		pages:          []Page{},
		pagesView:      tview.NewPages(),
		codePage:       NewCodePage(app, nav),
		varsPage:       NewVarPage(),
		stackPage:      NewStackPage(),
		goroutinePage:  NewGoroutinePage(),
	}
	pv.pages = []Page{pv.codePage, pv.varsPage, pv.stackPage, pv.goroutinePage}

	for _, p := range pv.pages {
		pv.pagesView.AddPage(p.GetName(), p.GetWidget(), true, true)
		p.SetCommandHandler(cmdHdlr)
	}
	pv.pagesView.SwitchToPage(pv.pages[0].GetName())

	return &pv
}

func (pv *PageView) CurrentPage() Page {
	return pv.pages[pv.index]
}

func (pv *PageView) RefreshLineColumn() {
	pv.codePage.perfTextView.ReRender()
}

func (pv *PageView) RenderBreakpointHit(bpi *api.BreakpointInfo) {
	if bpi == nil {
		return
	}
	pv.varsPage.RenderVariables(bpi.Arguments, bpi.Locals, []api.Variable{})
}

func (pv *PageView) RenderStack(sf []api.Stackframe, csf *api.Stackframe) {
	pv.varsPage.RenderVariables(csf.Arguments, csf.Locals, []api.Variable{})
	pv.stackPage.RenderStack(sf, csf)
}

func (pv *PageView) RenderJumpToLine(toLine int) {
	pv.codePage.perfTextView.scrollTo(toLine, true)
}

func (pv *PageView) GetWidget() tview.Primitive {
	return pv.pagesView
}

func (pv *PageView) LoadFile(file *nav.File, atLine int) {
	pv.index = 0
	pv.pagesView.SwitchToPage(pv.codePage.GetName())
	pv.codePage.OpenFile(file, atLine)
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
		if pv.index < len(pv.pages)-1 {
			pv.index++
			pv.pagesView.SwitchToPage(pv.CurrentPage().GetName())
		}
		return nil // Consume event
	}
	// Delegate
	return pv.CurrentPage().HandleKeyEvent(event)
}
