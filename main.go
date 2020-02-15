package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"syscall"
	"time"
)

// Findings captures idle and peak resource usage
type Findings struct {
	MemoryMaxRSS int64
	CPUuser      syscall.Timeval
	CPUsys       syscall.Timeval
}

var icmd, pcmd *exec.Cmd
var idlef, peakf chan Findings

func main() {
	// Define the CLI API:
	target := flag.String("target", "", "The filesystem path of the binary or script to assess")
	apipath := flag.String("api-path", "", "[OPTIONAL] The URL path component of the HTTP API to use for peak assessment")
	apiport := flag.String("api-port", "", "[OPTIONAL] The TCP port of the HTTP API to use for peak assessment")
	exportfile := flag.String("export-findings", "", "[OPTIONAL] The filesystem path to export findings to; if not provided the results will only be written to stdout")
	flag.Parse()
	if *target == "" {
		log.Fatalln("Need at least the target program to proceed")
	}

	// Set up global data structures:
	isampletime := time.Duration(1) * time.Second
	psampletime := time.Duration(5) * time.Second
	idlef = make(chan Findings, 1)
	peakf = make(chan Findings, 1)
	icmd = exec.Command(*target)
	pcmd = exec.Command(*target)

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
	ifs := <-idlef
	log.Printf("Found idle state resource usage. MEMORY: %vkB CPU: %vms (user)/%vms (sys)",
		ifs.MemoryMaxRSS/1000,
		ifs.CPUuser.Usec/1000,
		ifs.CPUsys.Usec/1000)

	// Perform peak state assessment:
	go assesspeak(*apiport, *apipath)
	<-time.After(psampletime)
	log.Printf("Peak state assessment of %v completed\n", *target)
	if pcmd.Process != nil {
		err := pcmd.Process.Signal(os.Interrupt)
		if err != nil {
			log.Fatalf("Can't stop process: %v\n", err)
		}
	}
	pfs := <-peakf
	log.Printf("Found peak state resource usage. MEMORY: %vkB CPU: %vms (user)/%vms (sys)",
		pfs.MemoryMaxRSS/1000,
		pfs.CPUuser.Usec/1000,
		pfs.CPUsys.Usec/1000)

	// Handle export of findings if instructed to do so
	if *exportfile != "" {
		export(ifs, pfs, *exportfile)
	}
}

// assessidle performs the idle state resource usage assessment, that is,
// the memory and CPU usage of the process without external traffic
func assessidle() {
	log.Printf("Launching %v for idle state resource usage assessment", icmd.Path)
	log.Println("Trying to determine idle state resource usage (no external traffic)")
	icmd.Run()
	f := Findings{}
	if icmd.ProcessState != nil {
		f.MemoryMaxRSS = icmd.ProcessState.SysUsage().(*syscall.Rusage).Maxrss
		f.CPUuser = icmd.ProcessState.SysUsage().(*syscall.Rusage).Utime
		f.CPUsys = icmd.ProcessState.SysUsage().(*syscall.Rusage).Stime
	}
	idlef <- f
}

// assesspeak performs the peak state resource usage assessment, that is,
// the memory and CPU usage of the process with external traffic applied
func assesspeak(apiport, apipath string) {
	log.Printf("Launching %v for peak state resource usage assessment", pcmd.Path)
	log.Printf("Trying to determine peak state resource usage using 127.0.0.1:%v%v", apiport, apipath)
	go stress(apiport, apipath)
	pcmd.Run()
	f := Findings{}
	if pcmd.ProcessState != nil {
		f := Findings{}
		f.MemoryMaxRSS = pcmd.ProcessState.SysUsage().(*syscall.Rusage).Maxrss
		f.CPUuser = pcmd.ProcessState.SysUsage().(*syscall.Rusage).Utime
		f.CPUsys = pcmd.ProcessState.SysUsage().(*syscall.Rusage).Stime
	}
	peakf <- f
}

func stress(apiport, apipath string) {
	time.Sleep(1 * time.Second)
	ep := fmt.Sprintf("http://127.0.0.1:%v%v", apiport, apipath)
	log.Printf("Starting to hammer the endpoint %v", ep)
	for {
		_, err := http.Get(ep)
		if err != nil {
			log.Println(err)
		}
		time.Sleep(1 * time.Second)
	}
}

func export(ifs, pfs Findings, exportfile string) {
	log.Printf("Exporting findings to %v", exportfile)
	fmt.Printf("%v %v", ifs, pfs)
}
