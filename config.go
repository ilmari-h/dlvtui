package main

import (
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/gdamore/tcell/v2"
	"github.com/spf13/viper"
)

var strColors = []string{
	"black",   // 0
	"maroon",  // 1
	"green",   // 2
	"olive",   // 3
	"navy",    // 4
	"purple",  // 5
	"teal",    // 6
	"silver",  // 7
	"gray",    // 8
	"red",     // 9
	"lime",    // 10
	"yellow",  // 11
	"blue",    // 12
	"fuchsia", // 13
	"aqua",    // 14
	"white",   // 15
	"aliceblue",
	"antiquewhite",
	"aquamarine",
	"azure",
	"beige",
	"bisque",
	"blanchedalmond",
	"blueviolet",
	"brown",
	"burlywood",
	"cadetblue",
	"chartreuse",
	"chocolate",
	"coral",
	"cornflowerblue",
	"cornsilk",
	"crimson",
	"darkblue",
	"darkcyan",
	"darkgoldenrod",
	"darkgray",
	"darkgreen",
	"darkkhaki",
	"darkmagenta",
	"darkolivegreen",
	"darkorange",
	"darkorchid",
	"darkred",
	"darksalmon",
	"darkseagreen",
	"darkslateblue",
	"darkslategray",
	"darkturquoise",
	"darkviolet",
	"deeppink",
	"deepskyblue",
	"dimgray",
	"dodgerblue",
	"firebrick",
	"floralwhite",
	"forestgreen",
	"gainsboro",
	"ghostwhite",
	"gold",
	"goldenrod",
	"greenyellow",
	"honeydew",
	"hotpink",
	"indianred",
	"indigo",
	"ivory",
	"khaki",
	"lavender",
	"lavenderblush",
	"lawngreen",
	"lemonchiffon",
	"lightblue",
	"lightcoral",
	"lightcyan",
	"lightgoldenrodyellow",
	"lightgray",
	"lightgreen",
	"lightpink",
	"lightsalmon",
	"lightseagreen",
	"lightskyblue",
	"lightslategray",
	"lightsteelblue",
	"lightyellow",
	"limegreen",
	"linen",
	"mediumaquamarine",
	"mediumblue",
	"mediumorchid",
	"mediumpurple",
	"mediumseagreen",
	"mediumslateblue",
	"mediumspringgreen",
	"mediumturquoise",
	"mediumvioletred",
	"midnightblue",
	"mintcream",
	"mistyrose",
	"moccasin",
	"navajowhite",
	"oldlace",
	"olivedrab",
	"orange",
	"orangered",
	"orchid",
	"palegoldenrod",
	"palegreen",
	"paleturquoise",
	"palevioletred",
	"papayawhip",
	"peachpuff",
	"peru",
	"pink",
	"plum",
	"powderblue",
	"rebeccapurple",
	"rosybrown",
	"royalblue",
	"saddlebrown",
	"salmon",
	"sandybrown",
	"seagreen",
	"seashell",
	"sienna",
	"skyblue",
	"slateblue",
	"slategray",
	"snow",
	"springgreen",
	"steelblue",
	"tan",
	"thistle",
	"tomato",
	"turquoise",
	"violet",
	"wheat",
	"whitesmoke",
	"yellowgreen",
	"grey",
	"dimgrey",
	"darkgrey",
	"darkslategrey",
	"lightgrey",
	"lightslategrey",
	"slategrey",
}

type Keys struct {
	Breakpoint string
	PageTop    string
	PageEnd    string
	LineUp     string
	LineDown   string
	PrevTab    string
	NextTab    string

	PrevSection string
	NextSection string

	SelectItem string

	ToggleBreakpoint string
	ClearBreakpoint  string
}

type Colors struct {
	BpFg           int
	BpActiveFg     int
	LineFg         int
	LineSelectedFg int
	LineSelectedBg int
	LineActiveFg   int
	LineActiveBg   int

	VarTypeFg  int
	VarNameFg  int
	VarValueFg int
	VarAddrFg  int

	ListHeaderFg   int
	ListExpand     int
	ListSelectedBg int
	HeaderFg       int
	CodeHeaderFg   int

	NotifErrorFg  int
	NotifPromptFg int
	NotifMsgFg    int

	MenuBg         int
	MenuFg         int
	MenuSelectedBg int
	MenuSelectedFg int
}

type Icons struct {
	Bp         string
	BpDisabled string
	BpActive   string

	IndRunning     string
	IndStopped     string
	IndExitSuccess string
	IndExitError   string
}

type Config struct {
	UseTabNavigation bool
	Keys             Keys
	Colors           Colors
	Icons            Icons
}

var gConfig Config

func NewConfig() Config {
	keyconf := Keys{
		Breakpoint:       "b",
		PageTop:          "g",
		PageEnd:          "G",
		LineUp:           "k",
		LineDown:         "j",
		PrevTab:          "h",
		NextTab:          "l",
		PrevSection:      "Backtab",
		NextSection:      "Tab",
		SelectItem:       "Enter",
		ToggleBreakpoint: "d",
		ClearBreakpoint:  "D",
	}
	colorconf := Colors{
		BpFg:           9,
		BpActiveFg:     1,
		LineFg:         15,
		LineSelectedFg: 0,
		LineSelectedBg: 15,
		LineActiveFg:   0,
		LineActiveBg:   6,

		VarTypeFg:  6,
		VarNameFg:  2,
		VarValueFg: 15,
		VarAddrFg:  8,

		ListHeaderFg:   5,
		ListExpand:     12,
		ListSelectedBg: 0,
		HeaderFg:       15,
		CodeHeaderFg:   12,

		NotifErrorFg:  1,
		NotifPromptFg: 2,
		NotifMsgFg:    15,

		MenuBg:         7,
		MenuFg:         0,
		MenuSelectedBg: 15,
		MenuSelectedFg: 0,
	}
	iconconf := Icons{
		Bp:         "●",
		BpDisabled: "○",
		BpActive:   "◎",

		IndRunning:     "▶",
		IndStopped:     "◼",
		IndExitSuccess: "⚑",
		IndExitError:   "⚐",
	}
	return Config{
		UseTabNavigation: true,
		Keys:             keyconf,
		Colors:           colorconf,
		Icons:            iconconf,
	}
}

func iToColorS(c int) string {
	return strColors[c]
}

func iToColorTcell(c int) tcell.Color {
	return tcell.ColorNames[strColors[c]]
}

// Override tree view input using custom keybindings.
func listInputCaptureC(event *tcell.EventKey) *tcell.EventKey {
	if keyPressed(event, gConfig.Keys.LineDown) {
		return tcell.NewEventKey(256, 'j', tcell.ModNone)
	}
	if keyPressed(event, gConfig.Keys.LineUp) {
		return tcell.NewEventKey(256, 'k', tcell.ModNone)
	}
	if keyPressed(event, gConfig.Keys.SelectItem) {
		return tcell.NewEventKey(tcell.KeyEnter, rune(tcell.KeyEnter), tcell.ModNone)
	}
	return nil
}

func keyPressed(event *tcell.EventKey, binding string) bool {
	if tBindingName, ok := tcell.KeyNames[event.Key()]; ok {
		return strings.ToLower(tBindingName) == strings.ToLower(binding)
	}
	return binding == string(event.Rune())
}

func getConfig() {
	gConfig = NewConfig() // Initialize with default values.

	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath("$XDG_CONFIG_HOME/dlvtui")
	viper.AddConfigPath(".") // Optionally use config in working directory.

	// Ignore error wher config file was not found, print error and exit on any other errors.
	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			// Config file was found but another error was produced
			fmt.Println("Error parsing config file: %w", err)
			os.Exit(1)
		}
		log.Println("No config file found, using defaults.")
	}

	conf_err := viper.Unmarshal(&gConfig)
	if conf_err != nil {
		log.Fatalf("Error reading configuration: %v", conf_err)
	}
	conf_keys_err := viper.UnmarshalKey("keys", &gConfig.Keys)
	if conf_keys_err != nil {
		log.Fatalf("Error reading key configuration: %v", conf_keys_err)
	}
	conf_icons_err := viper.UnmarshalKey("icons", &gConfig.Keys)
	if conf_keys_err != nil {
		log.Fatalf("Error reading icon configuration: %v", conf_icons_err)
	}
}
