package main

import (
	"log"
	"os"
	"bufio"
	"path/filepath"
	"dlvtui/nav"

	"github.com/go-delve/delve/service/api"
	"github.com/go-delve/delve/service/rpc2"
	"github.com/rivo/tview"
)

// Read file from disk.
func loadFile(path string, fileChan chan *nav.File) {
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
	absPath, _ := filepath.Abs(path)
	file := nav.File{
		Name:        path,
		Path:        absPath,
		Content:     buf,
		Breakpoints: nil,
		LineCount:   len(lineIndices),
		LineIndices: lineIndices,
	}
	log.Printf("Loaded file %s \n", absPath)
	fileChan <- &file
}


func StringToLineCommand(s string, args []string) LineCommand {
	log.Printf("Parsed command '%s %v'", s, args)
	switch s {
	case "open":
		return &OpenFile{
			File: args[0],
		}
	case "c", "continue":
		return &Continue{}
	case "q", "quit":
		return &Quit{}
	}
	return nil
}

type CommandHandler struct {
	view *View
	app *tview.Application
	rpcClient *rpc2.RPCClient
}

type LineCommand interface {
	run( *View, *tview.Application, *rpc2.RPCClient )
}

func NewCommandHandler(view *View, app *tview.Application, client *rpc2.RPCClient) *CommandHandler {
	return &CommandHandler{
		view: view,
		app: app,
		rpcClient: client,
	}
}

func (commandHandler *CommandHandler) RunCommand( cmd LineCommand ) {
	go cmd.run( commandHandler.view, commandHandler.app, commandHandler.rpcClient )
}

type CreateBreakpoint struct {
	Line int
	File string
}

func (cmd *CreateBreakpoint) run(view *View, app *tview.Application, client *rpc2.RPCClient) {
	// TODO: make configurable
	config := api.LoadConfig{
		FollowPointers: true,
		MaxVariableRecurse: 10,
		MaxStringLen: 999,
		MaxArrayValues: 999,
		MaxStructFields: -1,
	}
	log.Printf("Creating bp in %s at line %d\n",cmd.File, cmd.Line)
	res, err := client.CreateBreakpoint(&api.Breakpoint{
		File: cmd.File,
		Line: cmd.Line,
		Goroutine: true,
		LoadLocals: &config,
		LoadArgs: &config,
	})

	if err != nil {
		log.Printf("rpc error:%s\n",err.Error())
		return
	}
	view.breakpointChan <- res
}

type OpenFile struct {
	File string
}

func (cmd *OpenFile) run(view *View, app *tview.Application, client *rpc2.RPCClient) {

	// Check cache or open new file.
	absPath, _ := filepath.Abs(cmd.File)
	if val, ok := view.navState.FileCache[absPath]; ok {
		view.fileChan <- val
		return
	}
	app.SetFocus(view.textView)
	go loadFile(cmd.File, view.fileChan)
}


type ClearBreakpoint struct {
	Breakpoint *api.Breakpoint
}

func (cmd *ClearBreakpoint) run(view *View, app *tview.Application, client *rpc2.RPCClient) {
	res, err := client.ClearBreakpoint(cmd.Breakpoint.ID)
	if err != nil {
		log.Printf("rpc error:%s\n",err.Error())
		return
	}
	res.ID = -1 // Deleted
	view.breakpointChan <- res
}

type Quit struct {
}

func (cmd *Quit) run(view *View, app *tview.Application, client *rpc2.RPCClient) {
	app.Stop()
}

type Continue struct {
}

func (cmd *Continue) run(view *View, app *tview.Application, client *rpc2.RPCClient) {
	res := <- client.Continue()
	view.dbgStateChan <- res
}

type GetBreakpoints struct {
}

func (cmd *GetBreakpoints) run(view *View, app *tview.Application, client *rpc2.RPCClient) {
	bps, err := client.ListBreakpoints(true)
	if err != nil {
		log.Printf("rpc error:%s\n",err.Error())
		return
	}
	for i := range bps {
		view.breakpointChan <- bps[i]
	}
}
