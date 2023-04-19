package main

import (
	"github.com/ilmari-h/dlvtui/nav"
	"fmt"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

type CodePage struct {
	pageFrame      *tview.Frame
	flex           *tview.Flex
	commandHandler *CommandHandler
	navState       *nav.Nav
	perfTextView   *PerfTextView
	lineColumn     *LineColumn
}

func NewCodePage(app *tview.Application, navState *nav.Nav) *CodePage {

	textView := NewPerfTextView()

	lineColumn := NewLineColumn(navState)
	lineColumn.textView.
		SetRegions(true).
		SetDynamicColors(true).
		SetChangedFunc(func() {
			app.Draw()
		}).
		SetBackgroundColor(tcell.ColorDefault)

	textView.SetLineColumn(lineColumn)
	textView.SetDynamicColors(true)
	textView.SetBackgroundColor(tcell.ColorDefault)
	textView.SetWrap(false)

	flex := tview.NewFlex().SetDirection(tview.FlexColumn).
		AddItem(lineColumn.textView, 1, 1, false).
		AddItem(textView, 0, 1, false)

	pageFrame := tview.NewFrame(flex).
		SetBorders(0, 0, 0, 0, 0, 0).
		AddText("[::b]No file loaded.", true, tview.AlignLeft, iToColorTcell(gConfig.Colors.HeaderFg))
	pageFrame.SetBackgroundColor(tcell.ColorDefault)

	return &CodePage{
		pageFrame:    pageFrame,
		navState:     navState,
		flex:         flex,
		perfTextView: textView,
		lineColumn:   lineColumn,
	}
}

func (page *CodePage) OpenFile(file *nav.File, atLine int) {

	// Reset header.
	page.pageFrame.Clear()
	page.pageFrame.AddText(fmt.Sprintf("[::b]%s", file.Path), true, tview.AlignLeft, iToColorTcell(gConfig.Colors.CodeHeaderFg))

	// Redraw flex view with new column width.
	mv := getMaxLineColWidth(file.LineCount)
	page.lineColumn.SetWidth(mv)
	page.flex.ResizeItem(page.lineColumn.GetTextView(), mv, 1)

	page.perfTextView.SetTextP(file.Content, file.LineIndices)
	page.perfTextView.JumpTo(atLine)
}

func (page *CodePage) GetName() string {
	return "code"
}

func (page *CodePage) SetCommandHandler(cmdHdlr *CommandHandler) {
	page.commandHandler = cmdHdlr
}

func (page *CodePage) GetWidget() tview.Primitive {
	return page.pageFrame
}

func (page *CodePage) HandleKeyEvent(event *tcell.EventKey) *tcell.EventKey {
	if keyPressed(event, gConfig.Keys.LineDown) {
		if page.navState.CurrentFile != nil {
			line := page.navState.SetLine(page.navState.CurrentLine() + 1)
			page.perfTextView.scrollTo(line, false)
		}
		return nil
	}
	if keyPressed(event, gConfig.Keys.LineUp) {
		if page.navState.CurrentFile != nil {
			line := page.navState.SetLine(page.navState.CurrentLine() - 1)
			page.perfTextView.scrollTo(line, false)
		}
		return nil
	}
	if keyPressed(event, gConfig.Keys.PageTop) {
		line := page.navState.SetLine(0)
		page.perfTextView.scrollTo(line, true)
		return nil
	}
	if keyPressed(event, gConfig.Keys.PageEnd) {
		line := page.navState.SetLine(page.navState.CurrentFile.LineCount - 2)
		page.perfTextView.scrollTo(line, true)
		return nil
	}
	if keyPressed(event, gConfig.Keys.Breakpoint) {
		bps := page.navState.Breakpoints
		if _, ok := bps[page.navState.CurrentFile.Path][page.navState.CurrentLine()+1]; !ok {
			page.commandHandler.RunCommand(&CreateBreakpoint{
				Line: page.navState.CurrentLine() + 1, // Using 1 based indices on the backend.
				File: page.navState.CurrentFile.Path,
			})
		}
		return nil
	}
	if keyPressed(event, gConfig.Keys.ToggleBreakpoint) {
		bps := page.navState.Breakpoints
		// If breakpoint on this line, remove it.
		if len(bps[page.navState.CurrentFile.Path]) != 0 { // Using 1 based indices on the backend.
			if bp, ok := bps[page.navState.CurrentFile.Path][page.navState.CurrentLine()+1]; ok {
				if bp.Disabled {
					page.commandHandler.RunCommand(&CreateBreakpoint{
						Line: page.navState.CurrentLine() + 1, // Using 1 based indices on the backend.
						File: page.navState.CurrentFile.Path,
					})
				} else {
					page.commandHandler.RunCommand(&ClearBreakpoint{bp, true, nil})
				}
			}
		}
		return nil
	}
	if keyPressed(event, gConfig.Keys.ClearBreakpoint) {
		bps := page.navState.Breakpoints
		// If breakpoint on this line, remove it.
		if len(bps[page.navState.CurrentFile.Path]) != 0 { // Using 1 based indices on the backend.
			if bp, ok := bps[page.navState.CurrentFile.Path][page.navState.CurrentLine()+1]; ok {
				if bp.Disabled {
					page.commandHandler.RunCommand(&ClearBreakpoint{bp, false, bp})
				} else {
					page.commandHandler.RunCommand(&ClearBreakpoint{bp, false, nil})
				}
				return nil
			}
		}
	}
	return event // Propagate.
}
