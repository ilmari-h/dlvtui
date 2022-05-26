package main

import (
	"bufio"
	"os/exec"
	"strings"

	"github.com/rivo/tview"
)

func openFile(path string, filechan chan File) File {
	return File{}
}

func executeCommand(command LineCommand, args []string, view *View, app *tview.Application) {
	switch command {
	case Quit:
		app.Stop()
	case OpenFile:

		// Check cache or open new file.
		if val, ok := view.navState.fileCache[args[0]]; ok {
			view.fileChan <- val
			break
		}
		go openFile(args[0], view.fileChan)
	}
}

// Used for autosuggestions for now, a browser in the future.
func getFileList(projectRoot string, filesListChan chan []string) {
	out, err := exec.Command("find", projectRoot, "-name", "*.go").Output()
	if err != nil {
		panic(err)
	}
	scanner := bufio.NewScanner(strings.NewReader(string(out)))
	a := make([]string, 1)
	for scanner.Scan() {
		a = append(a, scanner.Text())
	}
	filesListChan <- a
}

func main() {
	app := tview.NewApplication()
	nav := Nav{projectPath: "."}

	view := CreateView(app,&nav)
	view.sourceFilesChan = make(chan []string)

	go getFileList(".", view.sourceFilesChan)
	go view.keyEventLoop(app, executeCommand)

	if err := app.Run(); err != nil {
		panic(err)
	}
}
