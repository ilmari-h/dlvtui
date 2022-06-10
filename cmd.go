package main

import (
	"bufio"
	"dlvtui/nav"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/go-delve/delve/service/api"
	"github.com/go-delve/delve/service/rpc2"
	"github.com/rivo/tview"
)

// TODO: make configurable
var defaultConfig = api.LoadConfig{
	FollowPointers:     true,
	MaxVariableRecurse: 10,
	MaxStringLen:       999,
	MaxArrayValues:     999,
	MaxStructFields:    -1,
}

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

var AvailableCommands = []string{
	"open",
	"c", "continue",
	"n", "next",
	"q", "quit",
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
	case "n", "next":
		return &Next{}
	case "q", "quit":
		return &Quit{}
	}
	return nil
}

type CommandHandler struct {
	view      *View
	app       *tview.Application
	rpcClient *rpc2.RPCClient
}

type LineCommand interface {
	run(*View, *tview.Application, *rpc2.RPCClient)
}

func NewCommandHandler(view *View, app *tview.Application, client *rpc2.RPCClient) *CommandHandler {
	return &CommandHandler{
		view:      view,
		app:       app,
		rpcClient: client,
	}
}

func (commandHandler *CommandHandler) RunCommand(cmd LineCommand) {
	go cmd.run(commandHandler.view, commandHandler.app, commandHandler.rpcClient)
}

func applyPrefix(pfx string, arr []string) []string {
	res := []string{}
	for _, v := range arr {
		res = append(res, pfx+v)
	}
	return res
}

func substractPrefix(pfx string, arr []string) []string {
	res := []string{}
	for _, v := range arr {
		if strings.HasPrefix(v, pfx) {
			res = append(res, v[len(pfx)+1:])
		}
	}
	return res
}

func filter(f string, arr []string) []string {
	res := []string{}
	for _, v := range arr {
		if strings.HasPrefix(v, f) {
			res = append(res, v)
		}
	}
	return res
}

func (commandHandler *CommandHandler) GetSuggestions(input string) []string {
	allArgs := strings.Fields(input)
	s := ""
	if len(allArgs) > 0 {
		s = allArgs[0]
	}
	switch s {
	case "open":
		opts := applyPrefix(s+" ",
			substractPrefix(
				commandHandler.view.navState.ProjectPath,
				commandHandler.view.navState.SourceFiles,
			),
		)
		return filter(input, opts)
	case "c", "continue":
		break
	}
	return filter(s, AvailableCommands)
}

type CreateBreakpoint struct {
	Line int
	File string
}

func (cmd *CreateBreakpoint) run(view *View, app *tview.Application, client *rpc2.RPCClient) {
	log.Printf("Creating bp in %s at line %d\n", cmd.File, cmd.Line)

	res, err := client.CreateBreakpoint(&api.Breakpoint{
		File:       cmd.File,
		Line:       cmd.Line,
		Goroutine:  true,
		LoadLocals: &defaultConfig,
		LoadArgs:   &defaultConfig,
	})

	if err != nil {
		log.Printf("rpc error:%s\n", err.Error())
		return
	}
	view.breakpointChan <- res
}

type OpenFile struct {
	File   string
	AtLine int
}

func (cmd *OpenFile) run(view *View, app *tview.Application, client *rpc2.RPCClient) {

	// Check cache or open new file.
	absPath := cmd.File
	if !filepath.IsAbs(cmd.File) {
		absPath = filepath.Join(view.navState.ProjectPath, cmd.File)
	}
	log.Printf("Opening file %s", absPath)
	view.navState.CurrentLines[absPath] = cmd.AtLine
	if val, ok := view.navState.FileCache[absPath]; ok {
		view.fileChan <- val
		return
	}
	go loadFile(absPath, view.fileChan)
}

type ClearBreakpoint struct {
	Breakpoint *api.Breakpoint
}

func (cmd *ClearBreakpoint) run(view *View, app *tview.Application, client *rpc2.RPCClient) {
	res, err := client.ClearBreakpoint(cmd.Breakpoint.ID)
	if err != nil {
		log.Printf("rpc error:%s\n", err.Error())
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

	// Reset debugger position for a pending continue and then re-render.
	view.navState.CurrentDebuggerPos = nav.DebuggerPos{File: "", Line: -1}

	res := <-client.Continue()
	sres, serr := client.Stacktrace(res.CurrentThread.GoroutineID, 5, api.StacktraceSimple, &defaultConfig)

	if serr != nil {
		log.Printf("rpc error:%s\n", serr.Error())
		return
	}

	if res.Exited {
		log.Printf("Program has finished with exit status %d.", res.ExitStatus)
		return
	}

	view.dbgMoveChan <- &DebuggerMove{res, nil, sres}
}

type GetBreakpoints struct {
}

func (cmd *GetBreakpoints) run(view *View, app *tview.Application, client *rpc2.RPCClient) {
	bps, err := client.ListBreakpoints(true)
	if err != nil {
		log.Printf("rpc error:%s\n", err.Error())
		return
	}
	for i := range bps {
		view.breakpointChan <- bps[i]
	}
}

type Next struct {
}

func (cmd *Next) run(view *View, app *tview.Application, client *rpc2.RPCClient) {

	nres, nerr := client.Next()
	sres, serr := client.Stacktrace(nres.CurrentThread.GoroutineID, 5, api.StacktraceSimple, &defaultConfig)

	if nerr != nil {
		log.Printf("rpc error:%s\n", nerr.Error())
		return
	}
	if serr != nil {
		log.Printf("rpc error:%s\n", serr.Error())
		return
	}
	if nres.Exited {
		log.Printf("Program has finished with exit status %d.", nres.ExitStatus)
		return
	}

	step := DebuggerStep{locals: sres[0].Locals, args: sres[0].Arguments}

	view.dbgMoveChan <- &DebuggerMove{nres, &step, sres}
}
