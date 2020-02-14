package main

import (
	"flag"
	"log"
	"os"
	"os/exec"
	"syscall"
	"time"
)

func main() {
	target := flag.String("target", "", "The filesystem path of the binary or script to assess")
	apipath := flag.String("api-path", "", "[OPTIONAL] The URL path component of the HTTP API to use for peak assessment")
	apiport := flag.String("api-port", "", "[OPTIONAL] The TCP port of the HTTP API to use for peak assessment")
	exportfile := flag.String("export-findings", "", "[OPTIONAL] The filesystem path to export findings to; if not provided the results will be written to stdout")
	flag.Parse()
	if *target == "" {
		log.Fatalln("Need at least the target program to proceed")
	}

	sampletime := 2
	done := make(chan bool)
	cmd := exec.Command(*target)

	log.Printf("Launching %v for idle state resource usage assessment", *target)
	go func() {
		log.Println("Trying to determine idle state resource usage (no external traffic)")
		cmd.Run()
		mem := cmd.ProcessState.SysUsage().(*syscall.Rusage).Maxrss
		cpuuser := cmd.ProcessState.SysUsage().(*syscall.Rusage).Utime
		cpusys := cmd.ProcessState.SysUsage().(*syscall.Rusage).Stime
		log.Printf("Found idle state resource usage. MEMORY: %vkB CPU: %vms (user)/%vms (sys)", mem/1000, cpuuser.Usec/1000, cpusys.Usec/1000)
		done <- true
	}()

	log.Printf("Launching %v for peak state resource usage assessment using 127.0.0.1:%v%v", *target, *apiport, *apipath)

	if *exportfile != "" {
		log.Printf("Exporting findings to %v", *exportfile)
	}
	for {
		select {
		case <-time.After(time.Duration(sampletime) * time.Second):
			log.Printf("Sampling completed, stopping %v\n", *target)
			err := cmd.Process.Signal(os.Interrupt)
			if err != nil {
				log.Fatalf("Can't stop process: %v\n", err)
			}
		case <-done:
			return
		}
	}
}
