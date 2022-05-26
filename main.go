package main

import (
	"bufio"
	"flag"
	"io/ioutil"
	"log"
	"os/exec"
	"strings"

	"github.com/rivo/tview"
)

func loadFile(path string, fileChan chan File) {
	f, err := ioutil.ReadFile(path)
	if err != nil {
		log.Printf("Error loading file %s:\n%s\n", path, err)
		return
	}

	file := File{
		name:        path,
		content:     string(f),
		breakpoints: nil,
	}
	log.Printf("Loaded file %s \n", path)
	fileChan <- file
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

func main() {
	flag.Parse()

	app := tview.NewApplication()
	nav := Nav{projectPath: "."}

	view := CreateView(app, &nav)

	go getFileList(".", nav.sourceFiles)
	go view.keyEventLoop(app, executeCommand)

	if err := app.Run(); err != nil {
		panic(err)
	}
}
