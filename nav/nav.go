package nav

import (
	"github.com/go-delve/delve/service/api"
)

type File struct {
	Name        string
	Path        string
	Content     string
	LineCount   int
	LineIndices []int
	PackageName string // TODO
}

type DebuggerPos struct {
	File string
	Line int

}

func (nav *Nav) CurrentLine() int {
	return nav.CurrentLines[nav.CurrentFile.Path]
}

func (nav *Nav) SetLine(line int) int {
	if line >= 0 && line < nav.CurrentFile.LineCount - 1 {
		nav.CurrentLines[nav.CurrentFile.Path] = line
		return line
	}
	return nav.CurrentLines[nav.CurrentFile.Path]
}

func (nav *Nav) LineInFile(filePath string) int{
	if _, ok := nav.CurrentLines[filePath]; !ok {
		return 0
	}
	return nav.CurrentLines[filePath]
}

func (nav *Nav) ChangeCurrentFile(file *File){
	if _, ok := nav.CurrentLines[file.Path]; !ok {
		nav.CurrentLines[file.Path] = 0
	}
	if _, ok := nav.FileCache[file.Path]; !ok {
		nav.FileCache[file.Path] = file
	}
	nav.CurrentFile = file
}

// Represents state of navigation within the project directory and the debugger.
type Nav struct {

	// Project level
	SourceFiles []string
	ProjectPath string
	FileCache   map[string]*File
	Goroutines []*api.Goroutine

	// Goroutine level
	// TODO: these could all be mapped under goroutine ID

	CurrentFile *File
	CurrentLines map[string]int

	DbgState *api.DebuggerState
	Breakpoints map[string] map[int]*api.Breakpoint
	CurrentDebuggerPos DebuggerPos
	CurrentStack []api.Stackframe
	CurrentStackFrame *api.Stackframe
}

// Load saved session
func loadNav(projectRootPath string) Nav {
	return Nav{}
}

func NewNav(projectPath string) Nav {

	return Nav{
		SourceFiles: []string{},
		ProjectPath: projectPath,
		FileCache:   make(map[string]*File),
		CurrentLines: make(map[string]int),
		Breakpoints: make(map[string] map[int]*api.Breakpoint),
		Goroutines: []*api.Goroutine{},
	}
}
