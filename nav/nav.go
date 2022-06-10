package nav

import (
	"github.com/go-delve/delve/service/api"
)

type File struct {
	Name        string
	Path        string
	Content     string
	Breakpoints []uint
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

func (nav *Nav) EnterNewFile(file *File) int {
	if _, ok := nav.CurrentLines[file.Path]; !ok {
		nav.CurrentLines[file.Path] = 0
	}
	nav.FileCache[file.Path] = file
	nav.CurrentFile = file
	return nav.CurrentLine()
}

func (nav *Nav) ChangeCurrentFile(filePath string){
	nav.CurrentFile = nav.FileCache[filePath]
}

// Represents state of navigation within the project directory and the debugger.
type Nav struct {

	// directory
	SourceFiles []string
	ProjectPath string
	CurrentFile *File
	CurrentLines map[string]int
	FileCache   map[string]*File

	// debugger
	DbgState *api.DebuggerState
	Breakpoints map[string] map[int]*api.Breakpoint
	CurrentDebuggerPos DebuggerPos
	CurrentStack []api.Stackframe
}

// Load saved session
func loadNav(projectRootPath string) Nav {
	return Nav{}
}

func NewNav(projectPath string) Nav {

	return Nav{
		ProjectPath: projectPath,
		FileCache:   make(map[string]*File),
		CurrentLines: make(map[string]int),
		Breakpoints: make(map[string] map[int]*api.Breakpoint),
	}
}
