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
func getFileList(client *rpc2.RPCClient) chan []string {
	filesListC := make(chan []string)
	go func() {
		files, err := client.ListSources("")
		if err != nil {
			log.Fatalf("Error tracing directory: %s", err)
		}
		filesListC <- files
	}()
	return filesListC
}

var (
	port       string
	attachMode bool
)

func main() {

	getConfig()

	// Parse flags after first argument.
	exFlags := flag.NewFlagSet("", flag.ExitOnError)
	exFlags.StringVar(&port, "port", "8181", "The port dlv rpc server will listen to.")
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

	clientC := make(chan *rpc2.RPCClient)

	if attachMode {
		startDebugger(attachDebuggerCmd(target, []string{}, port))
	} else {
		targetFile, _ := filepath.Abs(target)
		startDebugger(execDebuggerCmd(targetFile, []string{}, port))
	}

	go NewClient("127.0.0.1:"+port, clientC)
	rpcClient := <-clientC
	fileList := <-getFileList(rpcClient)

	if fileList == nil || len(fileList) == 0 {
		log.Fatalf("Error: empty source list.")
	}

	// Resolve dir. For now just find by assuming it's the one prefixed by /home.
	var dir string
	for _, f := range fileList {
		if strings.HasPrefix(f, "/home/") && !strings.Contains(f, "/go/pkg") {
			dir = filepath.Dir(f)
			break
		}
	}
	log.Printf("Using dir: %s", dir)

	app := tview.NewApplication()
	nav := nav.NewNav(dir)

	nav.SourceFiles = fileList

	CreateTui(app, &nav, rpcClient)

	if err := app.Run(); err != nil {
		panic(err)
	}
}
