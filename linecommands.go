package main

var maxArgsCount = map[LineCommand]int{
	OpenFile: 1,
	Quit:     0,

	DContinue: 0,
	DNext:     0,
	DStep:     0,
}

var minArgsCount = map[LineCommand]int{
	OpenFile: 1,
}

var stringToLineCommand = map[string]LineCommand{
	"q":    Quit,
	"quit": Quit,
	"open": OpenFile,

	"c": DContinue,
	"n": DNext,
	"s": DStep,
}

type LineCommand int

const (
	OpenFile LineCommand = iota
	Quit

	DContinue
	DNext
	DStep
)

func StringToLineCommand(s string) LineCommand {
	return stringToLineCommand[s]
}

func CheckArgBounds(cmd LineCommand, args []string) bool {
	if maxArgLen, ok := maxArgsCount[cmd]; ok {
		if len(args) > maxArgLen {
			return false
		}
	}

	if minArgsCount, ok := maxArgsCount[cmd]; ok {
		if len(args) < minArgsCount {
			return false
		}
	}
	return true
}

func getSuggestions(cmd string) []string {
	var strArray [1]string
	strArray[0] = "asdf"
	return strArray[:]
}
