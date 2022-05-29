package main

import (
	"bufio"
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strconv"
	"strings"

	"dlvtui/dlvrpc"
	"dlvtui/nav"

	"github.com/go-delve/delve/service/rpc2"
	"github.com/rivo/tview"
)

func killProcess(pid int) {
	_, err := exec.Command(
		"kill",
		strconv.Itoa(pid),
	).Output()
	if err != nil {
		log.Printf("Error terminating dlv-backend process at pid %d", pid)
	} else {
		log.Printf("Terminated dlv-backend process at pid %d", pid)
	}
}

func startDebugger(executable string, exArgs []string, port string) int {
	allArgs := []string{
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
	log.Printf("Starting dlv-backend:\ndlv %s", strings.Join(allArgs, " "))
	cmd := exec.Command(
		"dlv",
		allArgs...,
	)
	stdout, _ := cmd.StdoutPipe()
	if err := cmd.Start(); err != nil {
		log.Printf("Error starting backend:\n%s", string(err.Error()))
		panic(err)
	}

	log.Printf("dlv-backend running with pid %d", cmd.Process.Pid)

	go func() {
		in := bufio.NewScanner(stdout)
		for in.Scan() {
			log.Printf("dlv-backend:\n%s", in.Text())
		}
		if err := in.Err(); err != nil {
			log.Printf("Error:\n%s", err)
		}
	}()

	return cmd.Process.Pid
}

// Used for autosuggestions for now, a browser window in the future.
func getFileList(projectRoot string, filesList chan []string) {
	out, err := exec.Command("find", projectRoot, "-name", "*.go").Output()
	if err != nil {
		panic(err)
	}
	scanner := bufio.NewScanner(strings.NewReader(string(out)))
	a := make([]string, 1)
	for scanner.Scan() {
		a = append(a, scanner.Text())
	}
	filesList <- a
}

var (
	port string
)

func main() {

	// TODO: parse os args[1] absolute path
	if len( os.Args ) < 2 {
		fmt.Println("No executable or PID provided.")
		os.Exit(1)
	}

	flag.StringVar(&port, "port", "8181", "The port dlv rpc server will listen to.")
	flag.Parse()

	app := tview.NewApplication()
	nav := nav.NewNav(".")

	clientC := make(chan *rpc2.RPCClient)
	filesListC := make(chan []string)

	defer killProcess(startDebugger(os.Args[1], []string{}, "8181"))
	go dlvrpc.NewClient("127.0.0.1:"+port, clientC)
	go getFileList(".", filesListC)

	rpcClient := <-clientC
	nav.SourceFiles = <-filesListC

	CreateTui(app, &nav, rpcClient)

	if err := app.Run(); err != nil {
		panic(err)
	}
}
