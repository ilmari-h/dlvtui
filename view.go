package main

import (
	"fmt"
	"strings"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

type Mode int

const (
	Normal Mode = iota
	Insert
	Cmd
)

type ModeCommand struct {
	mode Mode
	key  tcell.Key
	rune rune
}

type View struct {
	commandChan  chan string
	keyEventChan chan ModeCommand
	fileChan     chan File
	masterView   *tview.Flex
	textView     *tview.TextView

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

func (view *View) keyEventLoop(app *tview.Application, runCommand CommandHandler) {
	for modeCommand := range view.keyEventChan {
		switch modeCommand.mode {
		case Normal:
			if modeCommand.rune == 58 {
				view.cmdLine.SetLabel(":")
				app.SetFocus(view.cmdLine)
				break
			}
			if modeCommand.rune == 105 {
				app.SetFocus(view.textView)
			}
		case Insert:
			if modeCommand.key == tcell.KeyEscape {
				app.SetFocus(view.masterView)
			}
		case Cmd:
			if modeCommand.key == tcell.KeyEscape {
				app.SetFocus(view.masterView)
			}
			if modeCommand.key == tcell.KeyEnter {
				linetext := view.cmdLine.GetText()
				view.cmdLine.SetLabel("")
				view.cmdLine.SetText("")
				app.SetFocus(view.masterView)
				command, args := parseCommand(linetext)
				runCommand(command, args, view, app)
			}
		}
	}
}

func (view *View) newFileLoop() {
	view.navState.fileCache = make(map[string]File)
	for newFile := range view.fileChan {

		// Update cache
		view.navState.fileCache[newFile.name] = newFile
		view.textView.SetText(newFile.content)
	}
}

func CreateView(app *tview.Application, nav *Nav) View {

	var view = View{
		commandChan:  make(chan string, 1024),
		keyEventChan: make(chan ModeCommand, 1024),
		fileChan:     make(chan File, 1024),
		navState:     nav,
	}

	flex := tview.NewFlex().SetDirection(tview.FlexRow)
	flex.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		view.keyEventChan <- ModeCommand{key: event.Key(), rune: event.Rune(), mode: Normal}
		return event // TODO: this propagates down and causes issues
		// Can't disable propagation either
		// Should just direct keyboard events altogether elsewhere when in normal mode
	})

	textView := tview.
		NewTextView().
		SetRegions(true).
		SetDynamicColors(true).
		SetChangedFunc(func() {
			app.Draw()
		})
	textView.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		view.keyEventChan <- ModeCommand{key: event.Key(), rune: event.Rune(), mode: Insert}
		return event
	})
	fmt.Fprintf(textView, "%s ", "")
	textView.SetBackgroundColor(tcell.ColorDefault)

	commandLine := tview.NewInputField().
		SetLabel("").
		SetFieldWidth(0).
		SetDoneFunc(func(key tcell.Key) {
			view.keyEventChan <- ModeCommand{key: key, mode: Cmd}
		})
	commandLine.SetBackgroundColor(tcell.ColorDefault)
	commandLine.SetFieldBackgroundColor(tcell.ColorDefault)

	flex.AddItem(
		textView, 0, 1, false)
	flex.AddItem(commandLine, 1, 1, false)

	app.SetRoot(flex, true).SetFocus(flex)

	view.masterView = flex
	view.textView = textView
	view.cmdLine = commandLine

	go view.newFileLoop()

	return view
}
