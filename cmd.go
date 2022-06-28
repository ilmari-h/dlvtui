package main

import (
	"bufio"
	"dlvtui/nav"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/go-delve/delve/service/api"
	"github.com/go-delve/delve/service/rpc2"
	"github.com/rivo/tview"
	log "github.com/sirupsen/logrus"
)

// TODO: make configurable
var defaultConfig = api.LoadConfig{
	FollowPointers:     true,
	MaxVariableRecurse: 5,
	MaxStringLen:       999,
	MaxArrayValues:     999,
	MaxStructFields:    -1,
}

// Read file from disk.
func loadFile(path string, fileChan chan *nav.File) {

	var scanner *bufio.Scanner = nil
	if gConfig.SyntaxHighlighter != "" {
		commandArr := strings.Fields(gConfig.SyntaxHighlighter)
		commandArr = append(commandArr, path)
		cmd := exec.Command(commandArr[0], commandArr[1:]...)
		pipe, _ := cmd.StdoutPipe()
		cmd.Stderr = cmd.Stdout
		scanner = bufio.NewScanner(pipe)
		err := cmd.Start()
		if err != nil {
			log.Fatal(err.Error())
		}
	} else {

		f, err := os.Open(path)
		defer f.Close()
		if err != nil {
			log.Printf("Error loading file %s: %s", path, err)
			return
		}
		scanner = bufio.NewScanner(f)
	}

	buf := ""
	lineIndex := 0
	lineIndices := []int{0}
	for scanner.Scan() {
		line := tview.TranslateANSI(scanner.Text())
		buf += line + "\n"
		lineIndex += len(line) + 1
		lineIndices = append(lineIndices, lineIndex)
	}
	absPath, _ := filepath.Abs(path)
	file := nav.File{
		Name:        path,
		Path:        absPath,
		Content:     buf,
		LineCount:   len(lineIndices),
		LineIndices: lineIndices,
	}
	log.Printf("Loaded file: %s", absPath)
	fileChan <- &file
}

var AvailableCommands = []string{
	"open",
	"bs", "breakpoints",
	"stack",
	"goroutines",
	"locals",
	"code",
	"restart",
	"c", "continue",
	"n", "next",
	"s", "step",
	"so", "stepout",
	"q", "quit",
}

func StringToLineCommand(s string, args []string) LineCommand {
	log.Printf("Parsed command '%s %v'", s, args)
	switch s {
	case "open":
		return &OpenFile{
			File: args[0],
		}
	case "bs", "breakpoints":
		return &OpenPage{PageIndex: IBreakPointsPage}
	case "stack":
		return &OpenPage{PageIndex: IStackPage}
	case "goroutines":
		return &OpenPage{PageIndex: IGoroutinePage}
	case "locals":
		return &OpenPage{PageIndex: IVarsPage}
	case "code":
		return &OpenPage{PageIndex: ICodePage}
	case "restart":
		return &Restart{}
	case "c", "continue":
		return &Continue{}
	case "n", "next":
		return &Next{}
	case "s", "step":
		return &Step{}
	case "so", "stepout":
		return &StepOut{}
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

	log.Printf("Creating bp in %s at line %d", cmd.File, cmd.Line)

	res, err := client.CreateBreakpoint(&api.Breakpoint{
		File:       cmd.File,
		Line:       cmd.Line,
		Goroutine:  true,
		LoadLocals: &defaultConfig,
		LoadArgs:   &defaultConfig,
	})

	if err != nil {
		log.Printf("rpc error: %s", err.Error())
		view.showNotification(err.Error(), true)
		return
	}
	view.breakpointChan <- &nav.UiBreakpoint{false, res}
}

type OpenPage struct {
	PageIndex PageIndex
}

func (cmd *OpenPage) run(view *View, app *tview.Application, client *rpc2.RPCClient) {
	view.pageView.SwitchToPage(cmd.PageIndex)
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
	view.navState.CurrentLines[absPath] = cmd.AtLine

	// If there's a stack frame for current file at current line, select it.
	for _, sf := range view.navState.CurrentStack {
		if sf.File == absPath && sf.Line == cmd.AtLine+1 {
			view.navState.CurrentStackFrame = &sf
			break
		} else {
			view.navState.CurrentStackFrame = nil
		}
	}

	if val, ok := view.navState.FileCache[absPath]; ok {
		view.fileChan <- val
		return
	}
	go loadFile(absPath, view.fileChan)
}

type ClearBreakpoint struct {
	Breakpoint *nav.UiBreakpoint
	Disable    bool
	OfflineBp  *nav.UiBreakpoint
}

func (cmd *ClearBreakpoint) run(view *View, app *tview.Application, client *rpc2.RPCClient) {

	// If removing breakpoint that doesn't exist in the backend, don't do an rpc call.
	if cmd.OfflineBp != nil {
		cmd.OfflineBp.ID = -1 // Mark as deleted
		view.breakpointChan <- cmd.OfflineBp
		return
	}

	res, err := client.ClearBreakpoint(cmd.Breakpoint.ID)
	if err != nil {
		log.Printf("rpc error: %s", err.Error())
		view.showNotification(err.Error(), true)
		return
	}
	if !cmd.Disable {
		res.ID = -1 // Mark as deleted
	}
	view.breakpointChan <- &nav.UiBreakpoint{cmd.Disable, res}
}

type Quit struct {
}

func (cmd *Quit) run(view *View, app *tview.Application, client *rpc2.RPCClient) {
	app.Stop()
}

type Continue struct {
}

func (cmd *Continue) run(view *View, app *tview.Application, client *rpc2.RPCClient) {

	view.renderPendingContinue()
	view.SetBlocking(true)

	res := <-client.Continue()
	view.SetBlocking(false)

	debuggerMoveCommand(view, app, client, res)
}

type GetBreakpoints struct {
}

func (cmd *GetBreakpoints) run(view *View, app *tview.Application, client *rpc2.RPCClient) {
	bps, err := client.ListBreakpoints(true)
	if err != nil {
		log.Printf("rpc error: %s", err.Error())
		view.showNotification(err.Error(), true)
		return
	}
	for i := range bps {
		view.breakpointChan <- &nav.UiBreakpoint{false, bps[i]}
	}
}

type Next struct {
}

func (cmd *Next) run(view *View, app *tview.Application, client *rpc2.RPCClient) {

	nres, nerr := client.Next()

	if nerr != nil {
		log.Printf("rpc error: %s", nerr.Error())
		return
	}

	if nres.Exited {
		msg := fmt.Sprintf("Program has finished with exit status %d.", nres.ExitStatus)
		log.Print(msg)
		view.showNotification(msg, false)
		return
	}

	sres, serr := client.Stacktrace(nres.CurrentThread.GoroutineID, 5, api.StacktraceSimple, &defaultConfig)

	if serr != nil {
		log.Printf("rpc error: %s", serr.Error())
		return
	}

	// Run ListGoroutines-command when ever new Goroutines may have been started.
	lg := ListGoroutines{}
	go lg.run(view, app, client)

	view.dbgMoveChan <- &DebuggerMove{nres, sres}
}

func debuggerMoveCommand(view *View, app *tview.Application, client *rpc2.RPCClient, cmdRes *api.DebuggerState) {

	if cmdRes.Exited {
		view.notifyProgramEnded(cmdRes.ExitStatus)
		return
	}

	sres, serr := client.Stacktrace(cmdRes.CurrentThread.GoroutineID, 5, api.StacktraceSimple, &defaultConfig)

	if serr != nil {
		log.Printf("rpc error: %s", serr.Error())
		return
	}

	// If file about to move has not been loaded, load it now.
	if view.navState.FileCache[cmdRes.CurrentThread.File] == nil {
		ch := make(chan *nav.File)
		go loadFile(cmdRes.CurrentThread.File, ch)

		// Block until file loaded so it can be opened.
		file := <-ch
		view.OpenFile(file, cmdRes.CurrentThread.Line-1)
	}
	view.dbgMoveChan <- &DebuggerMove{cmdRes, sres}

}

type Step struct {
}

func (cmd *Step) run(view *View, app *tview.Application, client *rpc2.RPCClient) {
	nres, nerr := client.Step()

	if nerr != nil {
		log.Printf("rpc error: %s", nerr.Error())
		return
	}

	debuggerMoveCommand(view, app, client, nres)
}

type StepOut struct {
}

func (cmd *StepOut) run(view *View, app *tview.Application, client *rpc2.RPCClient) {
	nres, nerr := client.StepOut()

	if nerr != nil {
		log.Printf("rpc error: %s", nerr.Error())
		return
	}

	debuggerMoveCommand(view, app, client, nres)
}

type ListGoroutines struct {
}

func (cmd *ListGoroutines) run(view *View, app *tview.Application, client *rpc2.RPCClient) {
	lres, _, lerr := client.ListGoroutines(0, 99)
	if lerr != nil {
		log.Printf("rpc error: %s", lerr.Error())
		return
	}
	log.Printf("Fetched active goroutines: %v", lres)
	view.goroutineChan <- lres
}

type SwitchGoroutines struct {
	Id int
}

func (cmd *SwitchGoroutines) run(view *View, app *tview.Application, client *rpc2.RPCClient) {
	log.Printf("Switching to goroutine %d.", cmd.Id)
	res, err := client.SwitchGoroutine(cmd.Id)
	if err != nil {
		log.Printf("rpc error: %s", err.Error())
		view.showNotification(err.Error(), true)
		return
	}
	sres, serr := client.Stacktrace(res.CurrentThread.GoroutineID, 5, api.StacktraceSimple, &defaultConfig)

	if serr != nil {
		log.Printf("rpc error: %s", serr.Error())
		return
	}

	log.Printf("Switched to goroutine %d.", res.Pid)

	view.dbgMoveChan <- &DebuggerMove{res, sres}
}

type Restart struct {
}

func (cmd *Restart) run(view *View, app *tview.Application, client *rpc2.RPCClient) {
	_, err := client.Restart(false)
	if err != nil {
		log.Printf("rpc error while restarting program: %s", err.Error())
		return
	}
	view.SetBlocking(false)
}
