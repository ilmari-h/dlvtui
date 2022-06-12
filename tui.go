package main

import (
	"log"
	"strings"

	"dlvtui/nav"

	"github.com/gdamore/tcell/v2"
	"github.com/go-delve/delve/service/api"
	"github.com/go-delve/delve/service/rpc2"
	"github.com/rivo/tview"
)

type Mode int

const (
	Normal Mode = iota
	Cmd
)

type KeyPress struct {
	mode  Mode
	event *tcell.EventKey
}

type DebuggerStep struct {
	locals  []api.Variable
	args    []api.Variable
	globals []api.Variable
}

type DebuggerMove struct {
	DbgState *api.DebuggerState
	Stack    []api.Stackframe
}

type View struct {
	commandChan chan string
	keyHandler  KeyHandler
	currentMode Mode
	fileChan    chan *nav.File

	pageView    *PageView
	masterView  *tview.Flex
	currentPage int

	cmdLine    *tview.InputField
	cmdHandler *CommandHandler

	dbgMoveChan    chan *DebuggerMove
	breakpointChan chan *api.Breakpoint
	navState       *nav.Nav
}

func parseCommand(input string) LineCommand {
	if len(input) == 0 {
		return nil
	}
	args := strings.Fields(input)
	command := args[0]
	cargs := args[1:]
	return StringToLineCommand(command, cargs)
}

type KeyHandler struct {
	app     *tview.Application
	view    *View
	prevKey *tcell.EventKey
}

func (keyHandler *KeyHandler) handleKeyEvent(kp KeyPress) *tcell.EventKey {
	view := keyHandler.view
	rune := kp.event.Rune()
	key := kp.event.Key()
	keyHandler.prevKey = kp.event

	if rune == ':' {
		view.toCmdMode()
		return nil
	}
	if key == tcell.KeyEscape { // This is only relevant when typing commands
		view.toNormalMode()
		return nil
	}

	// Parse and run command from line input
	if key == tcell.KeyEnter && view.cmdLine.HasFocus() {
		linetext := view.cmdLine.GetText()
		view.toNormalMode()
		command := parseCommand(linetext)
		if command != nil {
			view.cmdHandler.RunCommand(command)
		}
		return nil
	}

	// Delegate to page view, which either changes page or delegates to current page.
	return view.pageView.HandleKeyEvent(kp.event)
}

/**
 * Open a file.
 * If file has been opened previously, resume on that line. Otherwise open at
 * the first line.
 */
func (view *View) OpenFile(file *nav.File, atLine int) {
	view.navState.ChangeCurrentFile(file)
	view.navState.SetLine(atLine)
	view.pageView.LoadFile(file, atLine)

	// Render current stack frame if one is selected.
	if view.navState.CurrentStackFrame != nil {
		view.pageView.RenderStack(view.navState.CurrentStack, view.navState.CurrentStackFrame)
	}
}

func (view *View) fileLoop() {
	for newFile := range view.fileChan {
		view.OpenFile(
			newFile,
			view.navState.LineInFile(newFile.Path),
		)
	}
}

/**
 * Render a debugger move.
 * A debugger move is any operation where the current line or file changes, and
 * the current stack frame may also have new variables or function calls.
 */
func (view *View) dbgMoveLoop() {
	for dbgMove := range view.dbgMoveChan {
		newState := dbgMove.DbgState
		line := newState.CurrentThread.Line
		file := newState.CurrentThread.File
		view.navState.DbgState = newState
		view.navState.CurrentDebuggerPos = nav.DebuggerPos{File: file, Line: line}

		// Navigate to file and update call stack.
		log.Printf("Debugger move inside file %s on line %d.", file, line-1)
		view.OpenFile(view.navState.FileCache[file], line-1)

		view.navState.CurrentStack = dbgMove.Stack
		view.navState.CurrentStackFrame = &dbgMove.Stack[0]

		// If hit breakpoint.
		if newState.CurrentThread.BreakpointInfo != nil {

			log.Printf("Hit breakpoint in %s on line %d.", file, line)

			// Update breakpoint that was hit
			view.navState.Breakpoints[file][line] = newState.CurrentThread.Breakpoint
		}

		// Update pages.
		view.pageView.RenderBreakpointHit(dbgMove.DbgState.CurrentThread.BreakpointInfo)
		view.pageView.RenderStack(view.navState.CurrentStack, view.navState.CurrentStackFrame)
		view.pageView.RenderJumpToLine(line - 1)
	}
}

func (view *View) breakpointLoop() {
	for newBp := range view.breakpointChan {

		log.Printf("Got breakpoint in %s on line %d!", newBp.File, newBp.Line)

		// ID -1 signifies deleted breakpoint.
		if newBp.ID == -1 {
			delete(view.navState.Breakpoints[newBp.File], newBp.Line)
			view.pageView.RefreshLineColumn()
			continue
		}

		if len(view.navState.Breakpoints[newBp.File]) == 0 {
			view.navState.Breakpoints[newBp.File] = make(map[int]*api.Breakpoint)
		}
		view.navState.Breakpoints[newBp.File][newBp.Line] = newBp
		view.pageView.RefreshLineColumn()
	}
}

func (view *View) toNormalMode() {
	view.cmdLine.SetAutocompleteFunc(func(currentText string) (entries []string) {
		return []string{}
	})
	view.cmdLine.SetLabel("")
	view.cmdLine.SetText("")
	view.keyHandler.app.SetFocus(view.masterView)
	view.currentMode = Normal
}

func (view *View) toCmdMode() {
	view.cmdLine.SetAutocompleteFunc(view.cmdHandler.GetSuggestions)
	view.cmdLine.SetLabel(":")
	view.keyHandler.app.SetFocus(view.cmdLine)
	view.currentMode = Cmd
}

func CreateTui(app *tview.Application, navState *nav.Nav, rpcClient *rpc2.RPCClient) View {

	var view = View{
		commandChan:    make(chan string, 1024),
		fileChan:       make(chan *nav.File, 1024),
		dbgMoveChan:    make(chan *DebuggerMove, 1024),
		breakpointChan: make(chan *api.Breakpoint, 1024),
		navState:       navState,
		currentMode:    Normal,
		pageView:       nil,
	}

	view.cmdHandler = NewCommandHandler(&view, app, rpcClient)
	view.pageView = NewPageView(view.cmdHandler, navState, app)
	view.keyHandler = KeyHandler{app: app, view: &view}

	flex := tview.NewFlex().SetDirection(tview.FlexRow)
	flex.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		return view.keyHandler.handleKeyEvent(KeyPress{event: event, mode: view.currentMode})
	})

	commandLine := tview.NewInputField().
		SetLabel("").
		SetFieldWidth(0).
		SetDoneFunc(func(key tcell.Key) {
			event := tcell.NewEventKey(key, 0, tcell.ModNone)
			view.keyHandler.handleKeyEvent(KeyPress{event: event, mode: Cmd})
		})
	commandLine.SetBackgroundColor(tcell.ColorDefault)
	commandLine.SetFieldBackgroundColor(tcell.ColorDefault)

	flex.AddItem(view.pageView.GetWidget(), 0, 1, false)
	flex.AddItem(commandLine, 1, 1, false)

	app.SetRoot(flex, true).SetFocus(flex)

	view.masterView = flex
	view.cmdLine = commandLine

	go view.fileLoop()
	go view.breakpointLoop()
	go view.dbgMoveLoop()

	go view.cmdHandler.RunCommand(&GetBreakpoints{})

	return view
}
