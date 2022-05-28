package main

import (
	"github.com/go-delve/delve/service/api"
)

type File struct {
	name        string
	content     string
	breakpoints []uint
	lineCount   int
	lineIndices []int
	packageName string // TODO
}

type BreakPoint struct {
	filename string
	line     int
}

func (nav *Nav) CurrentLine() int {
	return nav.currentLine[nav.currentFile.name]
}

func (nav *Nav) SetLine(line int) {
	nav.currentLine[nav.currentFile.name] = line
}

func (nav *Nav) EnterNewFile(file *File) {
	if _, ok := nav.currentLine[file.name]; !ok {
		nav.currentLine[file.name] = 0
	}
	nav.currentLine[file.name] = 0
	nav.fileCache[file.name] = file
	nav.currentFile = file
}

// Represents state of navigation within the project directory and the debugger.
type Nav struct {

	// directory
	sourceFiles []string
	projectPath string
	currentFile *File
	currentLine map[string]int
	fileCache   map[string]*File

	// Debugger
	dbgState *api.DebuggerState
	breakpoints map[string] []*api.Breakpoint
}

// Load saved session
func loadNav(projectRootPath string) Nav {
	return Nav{}
}

func NewNav(projectPath string) Nav {

	return Nav{
		projectPath: projectPath,
		fileCache:   make(map[string]*File),
		currentLine: make(map[string]int),
	}
}
