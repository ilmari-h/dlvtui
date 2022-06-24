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

type PageIndex int8

const (
	ICodePage        PageIndex = 0
	IBreakPointsPage           = 1
	IVarsPage                  = 2
	IStackPage                 = 3
	IGoroutinePage             = 4
)

type PageView struct {
	commandHandler *CommandHandler
	index          int
	pages          []Page
	pagesView      *tview.Pages

	codePage        *CodePage
	breakpointsPage *BreakpointsPage
	varsPage        *VarsPage
	stackPage       *StackPage
	goroutinePage   *GoroutinePage
}

func NewPageView(cmdHdlr *CommandHandler, nav *nav.Nav, app *tview.Application) *PageView {
	pv := PageView{
		commandHandler:  cmdHdlr,
		index:           0,
		pages:           []Page{},
		pagesView:       tview.NewPages(),
		codePage:        NewCodePage(app, nav),
		breakpointsPage: NewBreakpointsPage(),
		varsPage:        NewVarPage(),
		stackPage:       NewStackPage(),
		goroutinePage:   NewGoroutinePage(),
	}
	pv.pages = []Page{pv.codePage, pv.breakpointsPage, pv.varsPage, pv.stackPage, pv.goroutinePage}

	for _, p := range pv.pages {
		pv.pagesView.AddPage(p.GetName(), p.GetWidget(), true, true)
		p.SetCommandHandler(cmdHdlr)
	}
	pv.pagesView.SwitchToPage(pv.pages[0].GetName())

	return &pv
}

func (pv *PageView) SwitchToPage(pi PageIndex) {
	pv.index = int(pi)
	page := pv.pages[pi]
	pv.pagesView.SwitchToPage(page.GetName())
}

func (pv *PageView) CurrentPage() Page {
	return pv.pages[pv.index]
}

func (pv *PageView) RefreshCodePage() {
	pv.codePage.perfTextView.ReRender()
}

func (pv *PageView) RenderBreakpoints(bps []*nav.UiBreakpoint) {
	if bps == nil {
		return
	}
	pv.breakpointsPage.RenderBreakpoints(bps)
}

func (pv *PageView) RenderBreakpointHit(bp *api.BreakpointInfo) {
	if bp == nil {
		return
	}
	pv.varsPage.RenderVariables(bp.Arguments, bp.Locals, []api.Variable{})
}

func (pv *PageView) RenderStack(sf []api.Stackframe, csf *api.Stackframe, returns []api.Variable) {
	if sf == nil || len(sf) == 0 || csf == nil {
		return
	}
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
	if keyPressed(event, gConfig.Keys.PrevTab) {
		if pv.index > 0 {
			pv.index--
			pv.pagesView.SwitchToPage(pv.CurrentPage().GetName())
		}
		return nil // Consume event
	} else if keyPressed(event, gConfig.Keys.NextTab) {
		if pv.index < len(pv.pages)-1 {
			pv.index++
			pv.pagesView.SwitchToPage(pv.CurrentPage().GetName())
		}
		return nil // Consume event
	}
	// Delegate
	return pv.CurrentPage().HandleKeyEvent(event)
}
