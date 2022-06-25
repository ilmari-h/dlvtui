package main

import (
	"fmt"
	"math"
	"strings"

	"dlvtui/nav"

	"github.com/gdamore/tcell/v2"
	"github.com/go-delve/delve/service/api"
	"github.com/go-delve/delve/service/rpc2"
	"github.com/rivo/tview"
	log "github.com/sirupsen/logrus"
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
	nwBlocking bool

	commandChan chan string
	keyHandler  KeyHandler
	currentMode Mode
	fileChan    chan *nav.File

	pageView    *PageView
	masterView  *tview.Flex
	currentPage int

	cmdLine       *tview.InputField
	indicatorText *tview.TextView
	cmdHandler    *CommandHandler

	notificationLine *tview.TextView

	dbgMoveChan    chan *DebuggerMove
	breakpointChan chan *nav.UiBreakpoint
	navState       *nav.Nav

	goroutineChan chan []*api.Goroutine
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

	// Block events if there's a notification prompt.
	if view.notificationLine.GetText(true) != "" {
		if key == tcell.KeyEnter {
			view.clearNotification()
		}
		return nil
	}

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

	if view.currentMode != Cmd {
		// Delegate to page view, which either changes page or delegates to current page.
		return view.pageView.HandleKeyEvent(kp.event)
	}
	return kp.event
}

func (view *View) SetBlocking(blocking bool) {
	view.nwBlocking = blocking
	if blocking {
		view.indicatorText.SetText(fmt.Sprintf("%s ", gConfig.Icons.IndRunning))
	} else {
		view.indicatorText.SetText(fmt.Sprintf("%s ", gConfig.Icons.IndStopped))
	}
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
		view.pageView.RenderStack(
			view.navState.CurrentStack,
			view.navState.CurrentStackFrame,
			view.navState.DbgState.CurrentThread.ReturnValues)
	}
}

/**
 * Listen to debugger state related messages from channels.
 * These usually result in re-rendering of some ui elements.
 */
func (view *View) uiEventLoop() {
	for {
		select {
		case dbgMove := <-view.dbgMoveChan:
			view.onDebuggerMove(dbgMove)
		case newFile := <-view.fileChan:
			view.onNewFile(newFile)
		case activeGoroutines := <-view.goroutineChan:
			view.onNewGoroutines(activeGoroutines)
		case newBp := <-view.breakpointChan:
			view.onNewBreakpoint(newBp)
		}
	}
}

// Render that's done before a continue command, essentially just refresh current pages.
func (view *View) renderPendingContinue() {

	view.navState.CurrentDebuggerPos = nav.DebuggerPos{File: "", Line: -1}
	view.pageView.RenderBreakpoints(view.navState.GetAllBreakpoints())
	view.pageView.RefreshCodePage()
}

/**
 * Render a debugger move.
 * A debugger move is any operation where the current line or file changes, and
 * the current stack frame may also have new variables or function calls.
 */
func (view *View) onDebuggerMove(dbgMove *DebuggerMove) {
	newState := dbgMove.DbgState
	line := newState.CurrentThread.Line
	file := newState.CurrentThread.File
	view.navState.DbgState = newState
	view.navState.CurrentDebuggerPos = nav.DebuggerPos{File: file, Line: line}

	// Navigate to file and update call stack.
	log.Printf("Debugger move inside file %s on line %d.", file, line-1)
	view.OpenFile(view.navState.FileCache[file], line-1)

	if len(dbgMove.Stack) > 0 {
		view.navState.CurrentStack = dbgMove.Stack
		view.navState.CurrentStackFrame = &dbgMove.Stack[0]
	}

	// If hit breakpoint.
	if newState.CurrentThread.BreakpointInfo != nil {

		log.Printf("Hit breakpoint in %s on line %d.", file, line)

		// Update breakpoint that was hit
		view.navState.Breakpoints[file][line] = &nav.UiBreakpoint{false, newState.CurrentThread.Breakpoint}
		view.pageView.RenderBreakpointHit(dbgMove.DbgState.CurrentThread.BreakpointInfo)
	}

	// Update pages.
	view.pageView.RenderBreakpoints(view.navState.GetAllBreakpoints())
	view.pageView.RenderStack(
		view.navState.CurrentStack,
		view.navState.CurrentStackFrame,
		view.navState.DbgState.CurrentThread.ReturnValues)
	view.pageView.RenderJumpToLine(line - 1)
}

func (view *View) onNewFile(newFile *nav.File) {
	view.OpenFile(
		newFile,
		view.navState.LineInFile(newFile.Path),
	)
}

func (view *View) onNewBreakpoint(newBp *nav.UiBreakpoint) {

	log.Printf("Got breakpoint in %s on line %d!", newBp.File, newBp.Line)

	// ID -1 signifies deleted breakpoint.
	if newBp.ID == -1 {
		delete(view.navState.Breakpoints[newBp.File], newBp.Line)
		view.pageView.RefreshCodePage()
		return
	}

	if len(view.navState.Breakpoints[newBp.File]) == 0 {
		view.navState.Breakpoints[newBp.File] = make(map[int]*nav.UiBreakpoint)
	}

	view.navState.Breakpoints[newBp.File][newBp.Line] = newBp
	view.pageView.RenderBreakpoints(view.navState.GetAllBreakpoints())
	view.pageView.RefreshCodePage()
}

func (view *View) onNewGoroutines(activeGoroutines []*api.Goroutine) {
	view.navState.Goroutines = activeGoroutines
	view.pageView.goroutinePage.RenderGoroutines(
		activeGoroutines,
		view.navState.DbgState.CurrentThread.GoroutineID,
	)
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

func (view *View) clearNotification() {
	view.notificationLine.SetText("")
	view.masterView.ResizeItem(view.notificationLine, 0, 0)
	view.masterView.ResizeItem(view.pageView.GetWidget(), 0, 1)
	log.Print("Clearing notif")
	view.pageView.RefreshCodePage()
}

func (view *View) notifyProgramEnded(exitCode int) {
	msg := fmt.Sprintf("Program has finished with exit status %d.", exitCode)
	log.Print(msg)
	view.showNotification(msg, false)
	if exitCode == 0 {
		view.indicatorText.SetText(fmt.Sprintf("%s ", gConfig.Icons.IndExitSuccess))
	} else {
		view.indicatorText.SetText(fmt.Sprintf("%s %d ", gConfig.Icons.IndExitError, exitCode))
	}
}

func (view *View) showNotification(msg string, error bool) {
	msgLen := len(msg)
	msg += fmt.Sprintf("\n[%s::b]Press Enter to continue", iToColorS(gConfig.Colors.NotifPromptFg))
	if error {
		msgLen += 7
		view.notificationLine.SetText(fmt.Sprintf("[%s::b]Error[%s:-:-]: %s",
			iToColorS(gConfig.Colors.NotifErrorFg),
			iToColorS(gConfig.Colors.NotifMsgFg),
			msg,
		))
	} else {
		view.notificationLine.SetText(fmt.Sprintf("[%s]%s", iToColorS(gConfig.Colors.NotifMsgFg), msg))
	}
	_, _, boxWidth, _ := view.notificationLine.GetRect()
	lines := int(math.Ceil(float64(msgLen) / float64(boxWidth)))
	_, _, _, linesText := view.pageView.pagesView.GetInnerRect()
	view.masterView.ResizeItem(view.notificationLine, lines+1, 1)
	view.masterView.ResizeItem(view.pageView.GetWidget(), linesText-lines+1, 1)
	log.Print("Adding notif")
	view.pageView.RefreshCodePage()
}

func CreateTui(app *tview.Application, navState *nav.Nav, rpcClient *rpc2.RPCClient) View {

	var view = View{
		nwBlocking:     false,
		commandChan:    make(chan string, 1024),
		fileChan:       make(chan *nav.File, 1024),
		dbgMoveChan:    make(chan *DebuggerMove, 1024),
		goroutineChan:  make(chan []*api.Goroutine, 1024),
		breakpointChan: make(chan *nav.UiBreakpoint, 1024),
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
	notificationLine := tview.NewTextView().
		SetDynamicColors(true).
		SetText("")
	notificationLine.SetBackgroundColor(tcell.ColorDefault)
	notificationLine.SetWordWrap(true)

	commandLine.SetBackgroundColor(tcell.ColorDefault)
	commandLine.SetFieldBackgroundColor(tcell.ColorDefault)

	suggestionStyle := tcell.StyleDefault.
		Foreground(iToColorTcell(gConfig.Colors.MenuFg)).
		Background(iToColorTcell(gConfig.Colors.MenuBg))

	suggestionStyleSelected := tcell.StyleDefault.
		Foreground(iToColorTcell(gConfig.Colors.MenuSelectedFg)).
		Background(iToColorTcell(gConfig.Colors.MenuSelectedBg)).
		Attributes(tcell.AttrBold)

	commandLine.SetAutocompleteStyles(iToColorTcell(gConfig.Colors.MenuBg), suggestionStyle, suggestionStyleSelected)

	indicatorText := tview.NewTextView()
	indicatorText.SetBackgroundColor(tcell.ColorDefault)
	indicatorText.SetTextAlign(tview.AlignRight)
	indicatorText.SetText(fmt.Sprintf("%s ", gConfig.Icons.IndStopped))

	bottomRow := tview.NewFlex().
		AddItem(commandLine, 0, 1, false).
		AddItem(indicatorText, 5, 1, false)

	flex.AddItem(view.pageView.GetWidget(), 0, 1, false)
	flex.AddItem(notificationLine, 0, 0, false)
	flex.AddItem(bottomRow, 1, 1, false)

	app.SetRoot(flex, true).SetFocus(flex)

	view.masterView = flex
	view.cmdLine = commandLine
	view.indicatorText = indicatorText
	view.notificationLine = notificationLine

	go view.uiEventLoop()
	go func() {
		rpcClient.SetReturnValuesLoadConfig(&defaultConfig)
	}()
	go view.cmdHandler.RunCommand(&GetBreakpoints{})

	return view
}
