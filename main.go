package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"syscall"
	"time"
)

// Findings captures idle and peak resource usage
type Findings struct {
	MemoryMaxRSS int64 `json:"memory_in_bytes"`
	CPUuser      int64 `json:"cpuuser_in_usec"`
	CPUsys       int64 `json:"cpusys_in_usec"`
}

var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)
var icmd, pcmd *exec.Cmd
var idlef, peakf chan Findings

func main() {
	// Define the CLI API:
	flag.Usage = func() {
		fmt.Printf("Usage:\n %s --target $BINARY \n [--api-path $HTTP_URL_PATH --api-port $HTTP_PORT --peak-delay $TIME_MS --sampletime-idle $TIME_SEC --sampletime-peak $TIME_SEC --export-findings $FILE --output json|openmetrics]\n", os.Args[0])
		fmt.Println("Example usage:\n rsg --target test/test --api-path /ping --api-port 8080 2>/dev/null")
		fmt.Println("Arguments:")
		flag.PrintDefaults()
	}
	target := flag.String("target", "", "The filesystem path of the binary or script to assess")
	idlest := flag.Int("sampletime-idle", 2, "[OPTIONAL] The time in seconds to perform idle resource usage assessment")
	peakst := flag.Int("sampletime-peak", 10, "[OPTIONAL] The time in seconds to perform peak resource usage assessment")
	apibaseurl := flag.String("api-baseurl", "http://127.0.0.1", "[OPTIONAL] The base URL component of the HTTP API to use for peak resource usage assessment")
	apipath := flag.String("api-path", "", "[OPTIONAL] The URL path component of the HTTP API to use for peak resource usage assessment")
	apiport := flag.String("api-port", "", "[OPTIONAL] The TCP port of the HTTP API to use for peak resource usage assessment")
	peakdelay := flag.Int("delay-peak", 10, "[OPTIONAL] The time in milliseconds to wait between two consecutive HTTP GET requests for peak resource usage assessment")
	exportfile := flag.String("export-findings", "", "[OPTIONAL] The filesystem path to export findings to; if not provided the results will be written to stdout")
	outputformat := flag.String("output", "json", "[OPTIONAL] The output format, valid values are 'json' and 'openmetrics'")
	showversion := flag.Bool("version", false, "Print the version of rsg and exit")
	flag.Parse()

	if *showversion {
		fmt.Printf("%v, commit %v, built at %v\n", version, commit, date)
		os.Exit(0)
	}

	if len(os.Args) == 0 || *target == "" {
		fmt.Printf("Need at least the target program to proceed\n\n")
		flag.Usage()
		os.Exit(1)
	}

	// Set up testing parameters
	isampletime := time.Duration(*idlest) * time.Second
	psampletime := time.Duration(*peakst) * time.Second
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
		go assesspeak(*apibaseurl, *apiport, *apipath, peakhammerpause)
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
	export(ifs, pfs, *exportfile, *outputformat, *target)
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
func assesspeak(apibaseurl, apiport, apipath string, peakhammerpause time.Duration) {
	log.Printf("Launching %v for peak state resource usage assessment", pcmd.Path)
	log.Printf("Trying to determine peak state resource usage using %v:%v%v", apibaseurl, apiport, apipath)
	go stress(apibaseurl, apiport, apipath, peakhammerpause)
	pcmd.Run()
	f := Findings{}
	if pcmd.ProcessState != nil {
		f.MemoryMaxRSS = int64(pcmd.ProcessState.SysUsage().(*syscall.Rusage).Maxrss)
		f.CPUuser = int64(pcmd.ProcessState.SysUsage().(*syscall.Rusage).Utime.Usec)
		f.CPUsys = int64(pcmd.ProcessState.SysUsage().(*syscall.Rusage).Stime.Usec)
	}
	peakf <- f
}

// stress performs an HTTP GET stress test against the base URL/port/path provided
func stress(apibaseurl, apiport, apipath string, peakhammerpause time.Duration) {
	time.Sleep(1 * time.Second)
	ep := fmt.Sprintf("%v:%v%v", apibaseurl, apiport, apipath)
	log.Printf("Starting to hammer %v every %v", ep, peakhammerpause)
	for {
		_, err := http.Get(ep)
		if err != nil {
			log.Println(err)
		}
		time.Sleep(peakhammerpause)
	}
}

// export writes the findings to a file or stdout, if exportfile is empty
func export(ifs, pfs Findings, exportfile, outputformat, target string) {
	fs := map[string]Findings{
		"idle": ifs,
		"peak": pfs,
	}
	data, err := json.MarshalIndent(fs, "", " ")
	if err != nil {
		log.Printf("Can't serialize findings: %v\n", err)
	}

	outputformat = strings.ToLower(outputformat)
	switch outputformat {
	case "json":
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
	case "openmetrics":
		var buffer bytes.Buffer
		buffer.WriteString(emito("idle_memory",
			"gauge",
			"The idle state memory consumption",
			fmt.Sprintf("%d", ifs.MemoryMaxRSS),
			map[string]string{"target": target, "unit": "kB"}))
		buffer.WriteString(emito("idle_cpu_user",
			"gauge",
			"The idle state CPU consumption in user land",
			fmt.Sprintf("%d", ifs.CPUuser),
			map[string]string{"target": target, "unit": "microsec"}))
		buffer.WriteString(emito("idle_cpu_sys",
			"gauge",
			"The idle state CPU consumption in the kernel",
			fmt.Sprintf("%d", ifs.CPUsys),
			map[string]string{"target": target, "unit": "microsec"}))
		if pfs.MemoryMaxRSS != 0 {
			buffer.WriteString(emito("peak_memory",
				"gauge",
				"The peak state memory consumption",
				fmt.Sprintf("%d", pfs.MemoryMaxRSS),
				map[string]string{"target": target, "unit": "kB"}))
			buffer.WriteString(emito("peak_cpu_user",
				"gauge",
				"The peak state CPU consumption in user land",
				fmt.Sprintf("%d", pfs.CPUuser),
				map[string]string{"target": target, "unit": "microsec"}))
			buffer.WriteString(emito("peak_cpu_sys",
				"gauge",
				"The peak state CPU consumption in the kernel",
				fmt.Sprintf("%d", pfs.CPUsys),
				map[string]string{"target": target, "unit": "microsec"}))
		}
		switch {
		case exportfile != "": // use export file path provided
			log.Printf("Exporting findings in OpenMetrics format to %v", exportfile)
			err := ioutil.WriteFile(exportfile, buffer.Bytes(), 0644)
			if err != nil {
				log.Printf("Can't export findings: %v\n", err)
			}
		default: // if no export file path set, write to stdout
			w := bufio.NewWriter(os.Stdout)
			_, err := w.Write(buffer.Bytes())
			if err != nil {
				log.Printf("Can't export findings: %v\n", err)
			}
			w.Flush()
		}
	default:
		log.Printf("Can't export findings, unknown output format selected, please use json or openmetrics")
	}
}

// emito creates an OpenMetrics compliant line, for example:
// # HELP pod_count_all Number of pods in any state (running, terminating, etc.)
// # TYPE pod_count_all gauge
// pod_count_all{namespace="krs"} 4 1538675211
func emito(metric, metrictype, metricdesc, value string, labels map[string]string) (line string) {
	line = fmt.Sprintf("# HELP %v %v\n", metric, metricdesc)
	line += fmt.Sprintf("# TYPE %v %v\n", metric, metrictype)
	// add labels:
	line += fmt.Sprintf("%v{", metric)
	for k, v := range labels {
		line += fmt.Sprintf("%v=\"%v\"", k, v)
		line += ","
	}
	// make sure that we get rid of trialing comma:
	line = strings.TrimSuffix(line, ",")
	// now add value and we're done:
	line += fmt.Sprintf("} %v\n", value)
	return
}
