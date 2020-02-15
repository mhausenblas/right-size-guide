package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"
	"syscall"
	"time"
)

// Findings captures idle and peak resource usage
type Findings struct {
	MemoryMaxRSS int64 `json:"memory_in_bytes"`
	CPUuser      int64 `json:"cpuuser_in_usec"`
	CPUsys       int64 `json:"cpusys_in_usec"`
}

var icmd, pcmd *exec.Cmd
var idlef, peakf chan Findings

func main() {
	// Define the CLI API:
	flag.Usage = func() {
		fmt.Printf("Usage:\n %s --target $BINARY [--api-path $HTTP_URL_PATH --api-port $HTTP_PORT --peak-delay $TIME_MS --export-findings $FILE]\n", os.Args[0])
		fmt.Println("Example usage:\n rsg --target test/test --api-path /ping --api-port 8080 2>/dev/null")
		fmt.Println("Arguments:")
		flag.PrintDefaults()
	}
	target := flag.String("target", "", "The filesystem path of the binary or script to assess")
	apipath := flag.String("api-path", "", "[OPTIONAL] The URL path component of the HTTP API to use for peak assessment")
	apiport := flag.String("api-port", "", "[OPTIONAL] The TCP port of the HTTP API to use for peak assessment")
	peakdelay := flag.Int("peak-delay", 10, "[OPTIONAL] The time in milliseconds to wait between two consecutive HTTP GET requests")
	exportfile := flag.String("export-findings", "", "[OPTIONAL] The filesystem path to export findings to; if not provided the results will be written to stdout")
	flag.Parse()
	if len(os.Args) == 0 || *target == "" {
		fmt.Printf("Need at least the target program to proceed\n\n")
		flag.Usage()
		os.Exit(1)
	}

	// Set up testing parameters
	isampletime := time.Duration(2) * time.Second
	psampletime := time.Duration(5) * time.Second
	peakhammerpause := time.Duration(*peakdelay) * time.Millisecond

	// Set up global data structures and testing parameter
	idlef = make(chan Findings, 1)
	peakf = make(chan Findings, 1)
	icmd = exec.Command(*target)
	pcmd = exec.Command(*target)
	ifs := Findings{}
	pfs := Findings{}

	// Perform idle state assessment:
	go assessidle()
	<-time.After(isampletime)
	log.Printf("Idle state assessment of %v completed\n", *target)
	if icmd.Process != nil {
		err := icmd.Process.Signal(os.Interrupt)
		if err != nil {
			log.Fatalf("Can't stop process: %v\n", err)
		}
	}
	ifs = <-idlef
	log.Printf("Found idle state resource usage. MEMORY: %vkB CPU: %vms (user)/%vms (sys)",
		ifs.MemoryMaxRSS/1000,
		ifs.CPUuser/1000,
		ifs.CPUsys/1000)

	// Perform peak state assessment:
	if *apipath != "" && *apiport != "" {
		go assesspeak(*apiport, *apipath, peakhammerpause)
		<-time.After(psampletime)
		log.Printf("Peak state assessment of %v completed\n", *target)
		if pcmd.Process != nil {
			err := pcmd.Process.Signal(os.Interrupt)
			if err != nil {
				log.Fatalf("Can't stop process: %v\n", err)
			}
		}
		pfs = <-peakf
		log.Printf("Found peak state resource usage. MEMORY: %vkB CPU: %vms (user)/%vms (sys)",
			pfs.MemoryMaxRSS/1000,
			pfs.CPUuser/1000,
			pfs.CPUsys/1000)
	}
	// Handle export of findings:
	export(ifs, pfs, *exportfile)
}

// assessidle performs the idle state resource usage assessment, that is,
// the memory and CPU usage of the process without external traffic
func assessidle() {
	log.Printf("Launching %v for idle state resource usage assessment", icmd.Path)
	log.Println("Trying to determine idle state resource usage (no external traffic)")
	icmd.Run()
	f := Findings{}
	if icmd.ProcessState != nil {
		f.MemoryMaxRSS = int64(icmd.ProcessState.SysUsage().(*syscall.Rusage).Maxrss)
		f.CPUuser = int64(icmd.ProcessState.SysUsage().(*syscall.Rusage).Utime.Usec)
		f.CPUsys = int64(icmd.ProcessState.SysUsage().(*syscall.Rusage).Stime.Usec)
	}
	idlef <- f
}

// assesspeak performs the peak state resource usage assessment, that is,
// the memory and CPU usage of the process with external traffic applied
func assesspeak(apiport, apipath string, peakhammerpause time.Duration) {
	log.Printf("Launching %v for peak state resource usage assessment", pcmd.Path)
	log.Printf("Trying to determine peak state resource usage using 127.0.0.1:%v%v", apiport, apipath)
	go stress(apiport, apipath, peakhammerpause)
	pcmd.Run()
	f := Findings{}
	if pcmd.ProcessState != nil {
		f.MemoryMaxRSS = int64(pcmd.ProcessState.SysUsage().(*syscall.Rusage).Maxrss)
		f.CPUuser = int64(pcmd.ProcessState.SysUsage().(*syscall.Rusage).Utime.Usec)
		f.CPUsys = int64(pcmd.ProcessState.SysUsage().(*syscall.Rusage).Stime.Usec)
	}
	peakf <- f
}

// stress performs an HTTP GET stress test against the port/path provided
func stress(apiport, apipath string, peakhammerpause time.Duration) {
	time.Sleep(1 * time.Second)
	ep := fmt.Sprintf("http://127.0.0.1:%v%v", apiport, apipath)
	log.Printf("Starting to hammer the endpoint %v every %v", ep, peakhammerpause)
	for {
		_, err := http.Get(ep)
		if err != nil {
			log.Println(err)
		}
		time.Sleep(peakhammerpause)
	}
}

// export writes the findings to a file or stdout, if exportfile is empty
func export(ifs, pfs Findings, exportfile string) {
	fs := map[string]Findings{
		"idle": ifs,
		"peak": pfs,
	}
	data, err := json.MarshalIndent(fs, "", " ")
	if err != nil {
		log.Printf("Can't serialize findings: %v\n", err)
	}
	switch {
	case exportfile != "": // use export file path provided
		log.Printf("Exporting findings as JSON to %v", exportfile)
		err := ioutil.WriteFile(exportfile, data, 0644)
		if err != nil {
			log.Printf("Can't export findings: %v\n", err)
		}
	default: // if no export file path set, write to stdout
		w := bufio.NewWriter(os.Stdout)
		_, err := w.Write(data)
		if err != nil {
			log.Printf("Can't export findings: %v\n", err)
		}
		w.Flush()
	}
}
