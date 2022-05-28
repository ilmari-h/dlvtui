package main

import (
	"log"
	"os"
	"bufio"

	"github.com/go-delve/delve/service/api"
	"github.com/go-delve/delve/service/rpc2"
	"github.com/rivo/tview"
)

// Read file from disk.
func loadFile(path string, fileChan chan *File) {
	f, err := os.Open(path)
	defer f.Close()
	if err != nil {
		log.Printf("Error loading file %s:\n%s\n", path, err)
		return
	}
	scanner := bufio.NewScanner(f)
	buf := ""
	lineIndex := 0
	lineIndices := []int{0}
	for scanner.Scan() {
		line := scanner.Text()
		buf += line + "\n"
		lineIndex += len(line) + 1
		lineIndices = append(lineIndices, lineIndex)
	}
	file := File{
		name:        path,
		content:     buf,
		breakpoints: nil,
		lineCount:   len(lineIndices),
		lineIndices: lineIndices,
	}
	log.Printf("Loaded file %s \n", path)
	fileChan <- &file
}


func StringToLineCommand(s string, args []string) LineCommand {
	log.Printf("Running command '%s %v'", s, args)
	switch s {
	case "open":
		return &OpenFile{
			File: args[0],
		}
	case "q", "quit":
		return &Quit{
		}
	}
	return nil
}

type CommandHandler struct {
	view *View
	app *tview.Application
	grpcClient *rpc2.RPCClient
}

type LineCommand interface {
	run( *View, *tview.Application, *rpc2.RPCClient )
}

func NewCommandHandler(view *View, app *tview.Application, client *rpc2.RPCClient) *CommandHandler {
	return &CommandHandler{
		view: view,
		app: app,
		grpcClient: client,
	}
}

func (commandHandler *CommandHandler) RunCommand( cmd LineCommand ) {
	cmd.run( commandHandler.view, commandHandler.app, commandHandler.grpcClient )
}

type NewBreakpoint struct {
	Line int
	File string
}


func (cmd *NewBreakpoint) run(view *View, app *tview.Application, client *rpc2.RPCClient) {
	client.CreateBreakpoint(&api.Breakpoint{
		ID: 1,
		File: cmd.File,
		Line: cmd.Line,
	})
}

type OpenFile struct {
	File string
}

func (cmd *OpenFile) run(view *View, app *tview.Application, client *rpc2.RPCClient) {
		// Check cache or open new file.
		if val, ok := view.navState.fileCache[cmd.File]; ok {
			view.fileChan <- val
			return
		}
		app.SetFocus(view.textView)
		go loadFile(cmd.File, view.fileChan)
}

type Quit struct {
}

func (cmd *Quit) run(view *View, app *tview.Application, client *rpc2.RPCClient) {
	app.Stop()
}
