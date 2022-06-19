package main

import (
	"bufio"
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"

	"dlvtui/nav"

	"github.com/go-delve/delve/service/rpc2"
	"github.com/rivo/tview"
)

func execDebuggerCmd(executable string, exArgs []string, port string) []string {
	log.Printf("Debugging executable at path: %s", executable)
	allArgs := []string{
		"exec",
		"--headless",
		"--accept-multiclient",
		"--api-version=2",
		"--listen=127.0.0.1:" + port,
		executable,
	}
	if exArgs != nil && len(exArgs) > 0 {
		allArgs = append(allArgs, "--")
		allArgs = append(allArgs, exArgs...)
	}
	return allArgs
}

func attachDebuggerCmd(pid string, exArgs []string, port string) []string {
	log.Printf("Debugging process with PID: %s", pid)
	allArgs := []string{
		"attach",
		"--headless",
		"--accept-multiclient",
		"--api-version=2",
		"--listen=127.0.0.1:" + port,
		pid,
	}
	if exArgs != nil && len(exArgs) > 0 {
		allArgs = append(allArgs, "--")
		allArgs = append(allArgs, exArgs...)
	}
	return allArgs
}

func startDebugger(commandArgs []string) int {
	log.Printf("Starting dlv-backend:\ndlv %s", strings.Join(commandArgs, " "))
	cmd := exec.Command(
		"dlv",
		commandArgs...,
	)
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Pdeathsig: syscall.SIGKILL,
	}
	stdout, _ := cmd.StdoutPipe()
	if err := cmd.Start(); err != nil {
		log.Printf("Error starting dlv-backend:\n%s", string(err.Error()))
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
	port       string
	dir        string
	attachMode bool
)

func main() {

	getConfig()

	// Parse flags after first argument.
	exFlags := flag.NewFlagSet("", flag.ExitOnError)
	exFlags.StringVar(&port, "port", "8181", "The port dlv rpc server will listen to.")
	exFlags.StringVar(&dir, "dir", "./", "Source code directory.")
	exFlags.BoolVar(&attachMode, "attach", false, "If enabled, attach debugger to process. Interpret first argument as PID.")

	if len(os.Args) < 2 {
		fmt.Println("No debug target provided.\n" +
			"The first argument should be an executable or a PID if the flag `attach` is set.")
		exFlags.Usage()
		os.Exit(1)
		return
	}

	exFlags.Parse(os.Args[2:])

	target := os.Args[1]
	dir, _ := filepath.Abs(dir)
	log.Printf("Using dir: %s", dir)

	app := tview.NewApplication()
	nav := nav.NewNav(dir)

	clientC := make(chan *rpc2.RPCClient)
	filesListC := make(chan []string)

	if attachMode {
		startDebugger(attachDebuggerCmd(target, []string{}, port))
	} else {
		targetFile, _ := filepath.Abs(target)
		startDebugger(execDebuggerCmd(targetFile, []string{}, port))
	}

	go NewClient("127.0.0.1:"+port, clientC)
	go getFileList(dir, filesListC)

	rpcClient := <-clientC
	nav.SourceFiles = <-filesListC

	CreateTui(app, &nav, rpcClient)

	if err := app.Run(); err != nil {
		panic(err)
	}
}
