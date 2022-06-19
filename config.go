package main

import (
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/gdamore/tcell/v2"
	"github.com/spf13/viper"
)

type Config struct {
	useTabNavigation bool
	prevTab          string
	nextTab          string

	breakpoint string
	pageTop    string
	pageEnd    string
	lineUp     string
	lineDown   string

	prevSection string
	nextSection string

	selectItem string

	toggleBreakpoint string
	clearBreakpoint  string
}

var gConfig Config

// Override tree view input using custom keybindings.
func listInputCaptureC(event *tcell.EventKey) *tcell.EventKey {
	if keyPressed(event, gConfig.lineDown) {
		return tcell.NewEventKey(256, 'j', tcell.ModNone)
	}
	if keyPressed(event, gConfig.lineUp) {
		return tcell.NewEventKey(256, 'k', tcell.ModNone)
	}
	if keyPressed(event, gConfig.selectItem) {
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

	viper.SetDefault("useTabNavigation", true)

	viper.SetDefault("keys.prevTab", "h")
	viper.SetDefault("keys.nextTab", "l")

	viper.SetDefault("keys.breakpoint", "b")
	viper.SetDefault("keys.pageTop", "g")
	viper.SetDefault("keys.pageEnd", "G")
	viper.SetDefault("keys.lineUp", "k")
	viper.SetDefault("keys.lineDown", "j")

	viper.SetDefault("keys.toggleBreakpoint", "d")
	viper.SetDefault("keys.clearBreakpoint", "D")

	viper.SetDefault("keys.prevSection", "Backtab")
	viper.SetDefault("keys.nextSection", "Tab")

	viper.SetDefault("keys.selectItem", "Enter")

	conf := Config{
		useTabNavigation: viper.GetBool("useTabNavigation"),
		prevTab:          viper.GetString("keys.prevTab"),
		nextTab:          viper.GetString("keys.nextTab"),
		breakpoint:       viper.GetString("keys.breakpoint"),
		pageTop:          viper.GetString("keys.pageTop"),
		pageEnd:          viper.GetString("keys.pageEnd"),
		lineUp:           viper.GetString("keys.lineUp"),
		lineDown:         viper.GetString("keys.lineDown"),
		prevSection:      viper.GetString("keys.prevSection"),
		nextSection:      viper.GetString("keys.nextSection"),
		selectItem:       viper.GetString("keys.selectItem"),

		toggleBreakpoint: viper.GetString("keys.toggleBreakpoint"),
		clearBreakpoint:  viper.GetString("keys.clearBreakpoint"),
	}
	gConfig = conf
}
