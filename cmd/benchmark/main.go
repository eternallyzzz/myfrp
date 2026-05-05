package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"endpoint/benchmark"
)

var (
	mode       = flag.String("mode", "client", "run mode: server or client")
	addr       = flag.String("addr", "127.0.0.1:12306", "server address")
	testType   = flag.String("test", "throughput", "test type: throughput, latency, concurrent")
	size       = flag.Int64("size", 1048576, "data size in bytes")
	concurrent = flag.Int("concurrent", 1, "number of concurrent streams")
	count      = flag.Int("count", 3, "number of test runs")
	pkgSize    = flag.Int("pkgSize", 32768, "packet size in bytes")
	port       = flag.Int("port", 12306, "server port")
)

func main() {
	flag.Parse()

	cfg := &benchmark.Config{
		Addr:       *addr,
		TestType:   *testType,
		Size:       *size,
		Concurrent: *concurrent,
		Count:      *count,
		PkgSize:    *pkgSize,
		Port:       *port,
	}

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	switch *mode {
	case "server":
		fmt.Println("=== myfrp Benchmark Server ===")
		fmt.Printf("Listening on port %d...\n", *port)
		if err := benchmark.RunServer(cfg); err != nil {
			fmt.Fprintf(os.Stderr, "Server error: %v\n", err)
			os.Exit(1)
		}
	case "client":
		fmt.Println("=== myfrp Benchmark Client ===")
		fmt.Printf("Connecting to %s...\n", *addr)
		results, err := benchmark.RunClient(cfg)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Client error: %v\n", err)
			os.Exit(1)
		}
		benchmark.PrintResults(results, cfg)
	default:
		fmt.Fprintf(os.Stderr, "Unknown mode: %s\n", *mode)
		os.Exit(1)
	}
}
