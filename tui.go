package main

import (
	"fmt"
	"log"
	"strings"

	"dlvtui/nav"
	"dlvtui/ui"

	"github.com/gdamore/tcell/v2"
	"github.com/go-delve/delve/service/api"
	"github.com/go-delve/delve/service/rpc2"
	"github.com/rivo/tview"
)

type Mode int

const (
	Normal Mode = iota
	Code
	Vars
	Cmd
)

type KeyPress struct {
	mode  Mode
	event *tcell.EventKey
}

type View struct {
	commandChan chan string
	keyHandler  KeyHandler
	currentMode Mode
	fileChan   chan *nav.File

	pages      *tview.Pages
	masterView *tview.Flex
	textView   *ui.PerfTextView
	varsView   *ui.VarsView

	cmdLine         *tview.InputField
	cmdHandler      *CommandHandler
	//lineSuggestions map[LineCommand][]string

	dbgStateChan chan *api.DebuggerState
	breakpointChan chan *api.Breakpoint
	navState          *nav.Nav
}

func parseCommand(input string) LineCommand {
	args := strings.Fields(input)
	command := args[0]
	cargs := args[1:]
	return StringToLineCommand(command, cargs)
}


type KeyHandler struct {
	app             *tview.Application
	view            *View
	prevKey         *tcell.EventKey
}

func (keyHandler *KeyHandler) handleKeyEvent(kp KeyPress) *tcell.EventKey {
	view := keyHandler.view
	rune := kp.event.Rune()
	pRune := ' '
	if keyHandler.prevKey != nil {
		pRune = keyHandler.prevKey.Rune()
	}
	key := kp.event.Key()
	keyHandler.prevKey = kp.event

	switch kp.mode {
	case Vars:
		if rune == ':' {
			view.toCmdMode()
			break
		}
		if rune == 'h' {
			view.pages.SwitchToPage("code")
			view.toNormalMode()
			break
		}
		keyHandler.app.SetFocus(view.varsView.GetWidget())
		return kp.event
	case Normal:
		if rune == ':' {
			view.toCmdMode()
			break
		}
		if rune == 'i' {
			view.toTextMode()
			break
		}
		if rune == 'h' {
			view.pages.SwitchToPage("code")
			break
		}
		if rune == 'l' {
			view.pages.SwitchToPage("vars")
			view.toVarsMode()
			break
		}
	case Code:
		if rune == 'b' {
			bps := view.navState.Breakpoints

			// If breakpoint on this line, remove it.
			if len( bps[view.navState.CurrentFile.Path]) != 0 { // Using 1 based indices on the backend.
				if bp, ok := bps[view.navState.CurrentFile.Path][view.navState.CurrentLine() + 1] ; ok {
					view.cmdHandler.RunCommand(&ClearBreakpoint{ bp })
					break;
				}
			}

			view.cmdHandler.RunCommand(&CreateBreakpoint{
				Line: view.navState.CurrentLine() + 1, // Using 1 based indices on the backend.
				File: view.navState.CurrentFile.Path, // This should have the absolute path TODO
			})
			break
		}
		if rune == ':' {
			view.toCmdMode()
			break
		}
		if rune == 'j' {
			view.scrollDown()
			break
		}
		if rune == 'k' {
			view.scrollUp()
			break
		}
		if rune == 'g' && pRune == 'g' {
			view.scrollToTop()
			break
		}
		if rune == 'G' && pRune == 'G' {
			view.scrollToBottom()
			break
		}
		if key == tcell.KeyEscape {
			view.toNormalMode()
			break
		}
	case Cmd:
		if key == tcell.KeyEscape {
			view.toNormalMode()
			break
		}
		if key == tcell.KeyEnter {
			linetext := view.cmdLine.GetText()
			view.toNormalMode()
			command := parseCommand(linetext)
			view.cmdHandler.RunCommand(command)
			break
		}

		// In command mode return the event to propagate it. Allows typing.
		return kp.event
	}
	// No event propagation by default.
	return nil
}

func (view *View) newFileLoop() {
	for newFile := range view.fileChan {
		view.navState.EnterNewFile(newFile)
		view.textView.SetTextP(newFile.Content, newFile.LineIndices)
		view.scrollToTop()
	}
}

func (view *View)setProgramAsExited(exitCode int) {
	view.navState.CurrentBreakpoint = nil
}

func (view *View) dbgStateLoop() {
	for newState := range view.dbgStateChan {
		line := newState.CurrentThread.Line
		file := newState.CurrentThread.File
		log.Printf("Hit breakpoint in %s on line %d!",file,line)

		view.navState.DbgState = newState

		// Update breakpoint that was hit
		view.navState.Breakpoints[file][line] = newState.CurrentThread.Breakpoint
		view.navState.CurrentBreakpoint = newState.CurrentThread.Breakpoint

		// Navigate to file at breakpoint.
		view.navState.ChangeCurrentFile(file)
		view.scrollTo(line - 1) // Internally use zero based indices.

		// Update local variables
		view.varsView.RenderBreakpointHit(newState.CurrentThread.BreakpointInfo)
	}
}

func (view *View) breakpointLoop() {
	for newBp := range view.breakpointChan {

		log.Printf("Got breakpoint in %s on line %d!",newBp.File, newBp.Line)
		// ID -1 signifies deleted breakpoint
		if newBp.ID == -1 {
			delete(view.navState.Breakpoints[newBp.File], newBp.Line)
			continue
		}

		if len(view.navState.Breakpoints[newBp.File]) == 0 {
			view.navState.Breakpoints[newBp.File] = make(map[int]*api.Breakpoint)
		}
		view.navState.Breakpoints[newBp.File][newBp.Line] = newBp
	}
}

func (view *View) toVarsMode() {
	view.cmdLine.SetLabel("")
	view.keyHandler.app.SetFocus(view.masterView)
	view.currentMode = Vars
}

func (view *View) toNormalMode() {
	view.cmdLine.SetLabel("")
	view.cmdLine.SetText("")
	view.keyHandler.app.SetFocus(view.masterView)
	view.currentMode = Normal
}

func (view *View) toCmdMode() {
	view.cmdLine.SetLabel(":")
	view.keyHandler.app.SetFocus(view.cmdLine)
	view.currentMode = Cmd
}

func (view *View) toTextMode() {
	view.cmdLine.SetLabel("")
	view.cmdLine.SetText("")
	view.keyHandler.app.SetFocus(view.textView)
	view.currentMode = Code
}

// Text navigation

func (view *View) scrollTo(line int) {
	if line < 0 || line >= view.navState.CurrentFile.LineCount {
		return
	}
	view.textView.ScrollTo(line)
	view.navState.SetLine(line)
}

func (view *View) scrollDown() {
	line := view.navState.CurrentLine() + 1
	view.scrollTo(line)
}

func (view *View) scrollUp() {
	line := view.navState.CurrentLine() - 1
	view.scrollTo(line)
}

func (view *View) scrollToTop() {
	view.scrollTo(0)
}

func (view *View) scrollToBottom() {
	line := view.navState.CurrentFile.LineCount - 1
	view.scrollTo(line)
}

func CreateTui(app *tview.Application, navState *nav.Nav, rpcClient *rpc2.RPCClient) View {

	var view = View{
		commandChan:       make(chan string, 1024),
		fileChan:          make(chan *nav.File, 1024),
		dbgStateChan: make(chan *api.DebuggerState, 1024),
		breakpointChan: make(chan *api.Breakpoint, 1024),
		navState:          navState,
		currentMode:       Normal,
		pages: tview.NewPages(),
	}

	view.keyHandler = KeyHandler{app: app, view: &view}
	view.cmdHandler= NewCommandHandler(&view, app, rpcClient)

	flex := tview.NewFlex().SetDirection(tview.FlexRow)
	flex.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		return view.keyHandler.handleKeyEvent(KeyPress{event: event, mode: view.currentMode})
	})

	view.textView = ui.NewPerfTextView()
	view.textView.
		SetRegions(true).
		SetDynamicColors(true).
		SetChangedFunc(func() {
			app.Draw()
		})
	lineColumn := NewLineColumn(5,navState)
	view.textView.SetGutterColumn(lineColumn)
	lineColumn.textView.
		SetRegions(true).
		SetDynamicColors(true).
		SetChangedFunc(func() {
			app.Draw()
		})

	fmt.Fprintf(view.textView, "%s ", "")
	view.textView.SetBackgroundColor(tcell.ColorDefault)
	view.textView.SetWrap(false)

	commandLine := tview.NewInputField().
		SetLabel("").
		SetFieldWidth(0).
		SetDoneFunc(func(key tcell.Key) {
			event := tcell.NewEventKey(key, 0, tcell.ModNone)
			view.keyHandler.handleKeyEvent(KeyPress{event: event, mode: Cmd})
		})
	commandLine.SetBackgroundColor(tcell.ColorDefault)
	commandLine.SetFieldBackgroundColor(tcell.ColorDefault)

	view.pages.AddPage("code", tview.NewFlex().SetDirection(tview.FlexColumn).
			AddItem(lineColumn.textView, lineColumn.width, 1, false).
			AddItem(view.textView, 0, 1, false),
		true, true)

	view.varsView = ui.NewVarsView()
	view.pages.AddPage("vars", view.varsView.GetWidget(), true, false)

	flex.AddItem( view.pages, 0, 1, false )
	flex.AddItem(commandLine, 1, 1, false)

	app.SetRoot(flex, true).SetFocus(flex)

	view.masterView = flex
	view.cmdLine = commandLine

	go view.newFileLoop()
	go view.breakpointLoop()
	go view.dbgStateLoop()

	go view.cmdHandler.RunCommand(&GetBreakpoints{})

	return view
}
