package benchmark

import (
	"fmt"
	"sort"
	"time"
)

func PrintResults(results []Result, cfg *Config) {
	if len(results) == 0 {
		return
	}

	fmt.Println()
	fmt.Println("=== Benchmark Results ===")
	fmt.Printf("Test Type: %s | Size: %s | Concurrent: %d\n",
		cfg.TestType, formatBytes(cfg.Size), cfg.Concurrent)
	fmt.Println()

	switch cfg.TestType {
	case "throughput", "concurrent":
		printThroughputResults(results, cfg)
	case "latency":
		printLatencyResults(results)
	}
}

func printThroughputResults(results []Result, cfg *Config) {
	var speeds []float64
	var totalDuration time.Duration
	var totalBytes int64

	for _, r := range results {
		speeds = append(speeds, r.SpeedMBps)
		totalDuration += r.Duration
		totalBytes += r.Bytes
	}

	sort.Float64s(speeds)

	avgSpeed := float64(totalBytes) / totalDuration.Seconds() / 1024 / 1024
	minSpeed := speeds[0]
	maxSpeed := speeds[len(speeds)-1]

	fmt.Println("Individual Runs:")
	for _, r := range results {
		fmt.Printf("  Run %d: %.2f MB/s (%v)\n", r.Index, r.SpeedMBps, r.Duration.Round(time.Millisecond))
	}

	fmt.Println()
	fmt.Printf("Average: %.2f MB/s\n", avgSpeed)
	fmt.Printf("Min:     %.2f MB/s\n", minSpeed)
	fmt.Printf("Max:     %.2f MB/s\n", maxSpeed)

	if len(speeds) >= 2 {
		var sumSqDev float64
		for _, s := range speeds {
			dev := s - avgSpeed
			sumSqDev += dev * dev
		}
		stdDev := 0.0
		if len(speeds) > 1 {
			stdDev = sumSqDev / float64(len(speeds)-1)
		}
		fmt.Printf("StdDev:  %.2f MB/s\n", stdDev)
	}
}

func printLatencyResults(results []Result) {
	for _, r := range results {
		fmt.Printf("Average Latency: %.3f ms\n", r.LatencyMs)
		fmt.Printf("Total Packets:   %d\n", r.Packets)
	}
}

func formatBytes(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}
