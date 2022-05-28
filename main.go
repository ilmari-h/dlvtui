package main

import (
	"bufio"
	"flag"
	"log"
	"os"
	"os/exec"
	"strconv"
	"strings"

	"github.com/rivo/tview"
)

func killProcess(pid int) {
	_,err := exec.Command(
		"kill",
		strconv.Itoa(pid),
	).Output()
	if err != nil  {
		log.Printf("Error terminating backend process at pid %d",pid)
	}
}

func startDebugger(executable string, exArgs []string, port string) int {
	allArgs := []string {
		"debug",
		"--headless",
		"--api-version=2",
		"--listen=127.0.0.1:" + port,
		"--accept-multiclient",
		executable,
	}
	if exArgs != nil && len(exArgs) > 0 {
		allArgs = append(allArgs, "--")
		allArgs = append(allArgs, exArgs...)
	}
	log.Printf("Starting backend:\ndlv %s",strings.Join(allArgs, " "))
	cmd := exec.Command(
		"dlv",
		allArgs...
		)
	stdout, _ := cmd.StdoutPipe()
	if err := cmd.Start(); err != nil  {
		log.Printf("Error starting backend:\n%s",string(err.Error()))
		panic(err)
	}

	go func() {
		in := bufio.NewScanner(stdout)
		for in.Scan() {
			log.Printf("Backend:\n%s",in.Text())
		}
		if err := in.Err(); err != nil {
			log.Printf("Error:\n%s", err)
		}
	}()

	return cmd.Process.Pid
}

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

func executeCommand(command LineCommand, args []string, view *View, app *tview.Application) {

	if !CheckArgBounds(command, args) {
		app.Stop()
		return
	}

	switch command {
	case Quit:
		app.Stop()
	case OpenFile:

		// Check cache or open new file.
		if val, ok := view.navState.fileCache[args[0]]; ok {
			view.fileChan <- val
			break
		}
		app.SetFocus(view.textView)
		go loadFile(args[0], view.fileChan)
	}
}

// Used for autosuggestions for now, a browser window in the future.
func getFileList(projectRoot string, filesList []string) {
	out, err := exec.Command("find", projectRoot, "-name", "*.go").Output()
	if err != nil {
		panic(err)
	}
	scanner := bufio.NewScanner(strings.NewReader(string(out)))
	a := make([]string, 1)
	for scanner.Scan() {
		a = append(a, scanner.Text())
	}
	filesList = a
}

var (
	flagVal string
	)

func main() {
	flag.StringVar(&flagVal, "port", "8181", "The port dlv grpc server will listen to.")
	flag.Parse()

	app := tview.NewApplication()
	nav := NewNav(".")

	defer killProcess(startDebugger("dlvtui",[]string{},"8181"))

	CreateView(app, &nav, executeCommand)

	go getFileList(".", nav.sourceFiles)

	if err := app.Run(); err != nil {
		panic(err)
	}
}
