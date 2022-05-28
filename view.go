package main

import (
	"fmt"
	"strings"
	"dlvtui/ui"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

type Mode int

const (
	Normal Mode = iota
	Text
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

	fileChan       chan *File
	masterView     *tview.Flex
	textView       *ui.PerfTextView

	cmdLine         *tview.InputField
	lineSuggestions map[LineCommand][]string

	navState *Nav
}

func parseCommand(input string) (LineCommand, []string) {
	args := strings.Fields(input)
	command := args[0]
	cargs := args[1:]
	return StringToLineCommand(command), cargs
}

type CommandHandler func(LineCommand, []string, *View, *tview.Application)

type KeyHandler struct {
	app             *tview.Application
	view            *View
	commandFunction CommandHandler
	prevKey         *tcell.EventKey
}

func (keyHandler *KeyHandler) handleKeyEvent(kp KeyPress) *tcell.EventKey {
	view := keyHandler.view
	runCommand := keyHandler.commandFunction
	app := keyHandler.app
	rune := kp.event.Rune()
	pRune := ' '
	if keyHandler.prevKey != nil {
		pRune = keyHandler.prevKey.Rune()
	}
	key := kp.event.Key()
	keyHandler.prevKey = kp.event

	switch kp.mode {
	case Normal:
		if rune == ':' {
			view.toCmdMode()
			break
		}
		if rune == 'i' {
			view.toTextMode()
			break
		}
	case Text:
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
			command, args := parseCommand(linetext)
			runCommand(command, args, view, app)
			break
		}
		// In command mode return the event to allow typing.
		return kp.event
	}
	// No event propagation by default.
	return nil
}

func (view *View) newFileLoop() {
	for newFile := range view.fileChan {
		lineNumbers := ""
		view.navState.EnterNewFile(newFile)
		view.textView.SetTextP(newFile.content, newFile.lineIndices)
		view.textView.ScrollToBeginning()
		view.textView.GetGutterColumn().SetText(lineNumbers)
	}
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
	view.currentMode = Text
}

// Text navigation

func (view *View) scrollTo(line int) {
	// TODO: this is very very slow with files over ~500 lines. Most likely due to highlighting using text search.
	// Set text again as batch.
	// use max lines to limit to size of screen.
	// To scroll back, only option is setting the complete view again with former text, probably to size of screen.
	// might be more efficient to use .BetchWriter to manually scroll and load more text?
	if line < 0 || line >= view.navState.currentFile.lineCount {
		return
	}
	view.textView.Highlight("3")
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
	line := view.navState.currentFile.lineCount - 1
	view.scrollTo(line)
}

func CreateView(app *tview.Application, nav *Nav, commandHandler CommandHandler) View {

	var view = View{
		commandChan: make(chan string, 1024),
		fileChan:    make(chan *File, 1024),
		navState:    nav,
		currentMode: Normal,
	}

	view.keyHandler = KeyHandler{app: app, view: &view, commandFunction: commandHandler}

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
	view.textView.SetGutterColumn( ui.NewGutterColumn() )
	view.textView.GetGutterColumn().
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

	flex.AddItem(
		tview.NewFlex().SetDirection(tview.FlexColumn).
			AddItem(view.textView.GetGutterColumn(), 5, 1, false).
			AddItem(view.textView, 0, 1, false),
		0, 1, false)
	flex.AddItem(commandLine, 1, 1, false)

	app.SetRoot(flex, true).SetFocus(flex)

	view.masterView = flex
	view.cmdLine = commandLine

	go view.newFileLoop()

	return view
}
