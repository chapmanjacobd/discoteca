//go:build pyroscope

package main

import (
	"flag"
	"log"
	"os"
	"runtime"
	"runtime/pprof"
	"time"
)

// Pyroscope/profiling integration for discoteca
// Build with: go build -tags pyroscope -o disco ./cmd/disco
// Run with: ./disco --cpuprof=cpu.prof --memprof=mem.prof server [...]

var (
	cpuProfile      = flag.String("cpuprof", "", "Write CPU profile to file")
	memProfile      = flag.String("memprof", "", "Write memory profile to file")
	traceFile       = flag.String("trace", "", "Write execution trace to file")
	profileDuration = flag.Duration("profile-duration", 30*time.Second, "Duration for profiling")
)

// setupProfiling sets up profiling based on command-line flags
// Call this early in main() after flag.Parse()
func setupProfiling() func() {
	var cleanupFuncs []func()

	// CPU profiling
	if *cpuProfile != "" {
		f, err := os.Create(*cpuProfile)
		if err != nil {
			log.Fatal("could not create CPU profile: ", err)
		}

		if err := pprof.StartCPUProfile(f); err != nil {
			log.Fatal("could not start CPU profile: ", err)
		}

		log.Printf("CPU profiling started, will run for %v", *profileDuration)

		// Schedule automatic stop after duration
		go func() {
			time.Sleep(*profileDuration)
			pprof.StopCPUProfile()
			f.Close()
			log.Printf("CPU profiling completed, saved to %s", *cpuProfile)
		}()

		cleanupFuncs = append(cleanupFuncs, func() {
			pprof.StopCPUProfile()
			f.Close()
		})
	}

	// Memory profiling
	if *memProfile != "" {
		cleanupFuncs = append(cleanupFuncs, func() {
			f, err := os.Create(*memProfile)
			if err != nil {
				log.Fatal("could not create memory profile: ", err)
			}
			defer f.Close()

			// Force GC to get accurate stats
			runtime.GC()

			if err := pprof.WriteHeapProfile(f); err != nil {
				log.Fatal("could not write memory profile: ", err)
			}

			log.Printf("Memory profile saved to %s", *memProfile)
		})
	}

	// Execution tracing
	if *traceFile != "" {
		f, err := os.Create(*traceFile)
		if err != nil {
			log.Fatal("could not create trace file: ", err)
		}

		if err := runtime.StartTrace(f); err != nil {
			log.Fatal("could not start trace: ", err)
		}

		log.Printf("Execution tracing started, will run for %v", *profileDuration)

		cleanupFuncs = append(cleanupFuncs, func() {
			runtime.StopTrace()
			f.Close()
			log.Printf("Execution trace saved to %s", *traceFile)
		})
	}

	return func() {
		// Run cleanup functions in reverse order
		for i := len(cleanupFuncs) - 1; i >= 0; i-- {
			cleanupFuncs[i]()
		}
	}
}
