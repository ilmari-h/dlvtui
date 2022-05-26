package main

type File struct {
	name        string
	content     string
	breakpoints []uint
}

type BreakPoint struct {
	filename string
	line int
}

// Represents state of navigation within the project directory and the debugger.
type Nav struct {

	// directory
	projectPath string
	currentFile File
	currentLine map[string]int
	fileCache   map[string]File

	// debugger
	locals      []string
	breakpoints map[string][]int
	currentBreakpoint BreakPoint
	goroutines  []string
	stack []string
}

// Load saved session
func loadNav(projectRootPath string) Nav {
	return Nav{}
}
