package main

import (
	"dlvtui/nav"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

type CodePage struct {
	commandHandler *CommandHandler
	navState     *nav.Nav
	flex         *tview.Flex
	perfTextView *PerfTextView
	gutterColumn GutterColumn
}


func NewCodePage(app *tview.Application, navState *nav.Nav) *CodePage {

	textView := NewPerfTextView()
	lineColumn := NewLineColumn(5,navState)
	textView.SetGutterColumn(lineColumn)
	lineColumn.textView.
		SetRegions(true).
		SetDynamicColors(true).
		SetChangedFunc(func() {
			app.Draw()
		})
	textView.SetBackgroundColor(tcell.ColorDefault)
	textView.SetWrap(false)
	textView.SetText("")

	flex := tview.NewFlex().SetDirection(tview.FlexColumn).
			AddItem(lineColumn.textView, lineColumn.width, 1, false).
			AddItem(textView, 0, 1, false)
	return &CodePage{
		navState: navState,
		flex: flex,
		perfTextView: textView,
		gutterColumn: lineColumn,
	}
}

func (page *CodePage) GetName() string {
	return "code"
}

func (page *CodePage) SetCommandHandler(cmdHdlr *CommandHandler) {
	page.commandHandler = cmdHdlr
}

func (page *CodePage) GetWidget() tview.Primitive {
	return page.flex
}

func (page *CodePage) HandleKeyEvent(event *tcell.EventKey) *tcell.EventKey {
	rune := event.Rune()
	if rune == 'j' {
		line := page.navState.SetLine(page.navState.CurrentLine() + 1)
		page.perfTextView.scrollTo(line, false)
		return nil
	}
	if rune == 'k' {
		line := page.navState.SetLine(page.navState.CurrentLine() - 1)
		page.perfTextView.scrollTo(line,false)
		return nil
	}
	if rune == 'g' {
		line := page.navState.SetLine(0)
		page.perfTextView.scrollTo(line,true)
		return nil
	}
	if rune == 'G' {
		line := page.navState.SetLine(page.navState.CurrentFile.LineCount - 2)
		page.perfTextView.scrollTo(line,true)
		return nil
	}
	if rune == 'b' {
		bps := page.navState.Breakpoints
		// If breakpoint on this line, remove it.
		if len( bps[page.navState.CurrentFile.Path]) != 0 { // Using 1 based indices on the backend.
			if bp, ok := bps[page.navState.CurrentFile.Path][page.navState.CurrentLine() + 1] ; ok {
				page.commandHandler.RunCommand(&ClearBreakpoint{ bp })
				return nil;
			}
		}

		page.commandHandler.RunCommand(&CreateBreakpoint{
			Line: page.navState.CurrentLine() + 1, // Using 1 based indices on the backend.
			File: page.navState.CurrentFile.Path,
		})
		return nil
	}
	return event // Propagate.
}
