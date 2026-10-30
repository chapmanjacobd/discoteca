//go:build pyroscope

package main

import (
	"log"
	"os"
	"runtime"
	"runtime/pprof"
	"strings"
	"time"
)

// Pyroscope/profiling integration for discoteca
// Build with: go build -tags pyroscope -o disco ./cmd/disco
// Run with: ./disco --cpuprof=cpu.prof --memprof=mem.prof server [...]

const (
	profileCPUFlag      = "--cpuprof"
	profileMemFlag      = "--memprof"
	profileTraceFlag    = "--trace"
	profileDurationFlag = "--profile-duration"
)

// setupProfiling extracts profiling flags before Kong parses the command line,
// then configures the requested profilers.
func setupProfiling() func() {
	values, remaining := extractProfilingArgs(os.Args[1:])
	os.Args = append([]string{os.Args[0]}, remaining...)

	profileDuration := 30 * time.Second
	if value := values[profileDurationFlag]; value != "" {
		if parsed, err := time.ParseDuration(value); err == nil {
			profileDuration = parsed
		} else {
			log.Printf("invalid %s value %q: %v", profileDurationFlag, value, err)
		}
	}

	var cleanupFuncs []func()

	if cpuProfile := values[profileCPUFlag]; cpuProfile != "" {
		f, err := os.Create(cpuProfile)
		if err != nil {
			log.Fatal("could not create CPU profile: ", err)
		}

		if err := pprof.StartCPUProfile(f); err != nil {
			log.Fatal("could not start CPU profile: ", err)
		}

		log.Printf("CPU profiling started, will run for %v", profileDuration)

		go func() {
			time.Sleep(profileDuration)
			pprof.StopCPUProfile()
			f.Close()
			log.Printf("CPU profiling completed, saved to %s", cpuProfile)
		}()

		cleanupFuncs = append(cleanupFuncs, func() {
			pprof.StopCPUProfile()
			f.Close()
		})
	}

	if memProfile := values[profileMemFlag]; memProfile != "" {
		cleanupFuncs = append(cleanupFuncs, func() {
			f, err := os.Create(memProfile)
			if err != nil {
				log.Fatal("could not create memory profile: ", err)
			}
			defer f.Close()

			runtime.GC()

			if err := pprof.WriteHeapProfile(f); err != nil {
				log.Fatal("could not write memory profile: ", err)
			}

			log.Printf("Memory profile saved to %s", memProfile)
		})
	}

	if traceFile := values[profileTraceFlag]; traceFile != "" {
		f, err := os.Create(traceFile)
		if err != nil {
			log.Fatal("could not create trace file: ", err)
		}

		if err := runtime.StartTrace(); err != nil {
			log.Fatal("could not start trace: ", err)
		}

		log.Printf("Execution tracing started, will run for %v", profileDuration)

		cleanupFuncs = append(cleanupFuncs, func() {
			runtime.StopTrace()
			f.Close()
			log.Printf("Execution trace saved to %s", traceFile)
		})
	}

	return func() {
		for i := len(cleanupFuncs) - 1; i >= 0; i-- {
			cleanupFuncs[i]()
		}
	}
}

func extractProfilingArgs(args []string) (map[string]string, []string) {
	values := make(map[string]string)
	remaining := make([]string, 0, len(args))
	known := map[string]bool{
		profileCPUFlag:      true,
		profileMemFlag:      true,
		profileTraceFlag:    true,
		profileDurationFlag: true,
	}

	for i := 0; i < len(args); i++ {
		arg := args[i]

		matched := false
		for flag := range known {
			if arg == flag {
				if i+1 < len(args) {
					values[flag] = args[i+1]
					i++
				}
				matched = true
				break
			}
			if value, ok := strings.CutPrefix(arg, flag+"="); ok {
				values[flag] = value
				matched = true
				break
			}
		}
		if !matched {
			remaining = append(remaining, arg)
		}
	}

	return values, remaining
}
