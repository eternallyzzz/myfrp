package benchmark

import "time"

type Config struct {
	Addr       string
	TestType   string
	Size       int64
	Concurrent int
	Count      int
	PkgSize    int
	Port       int
}

type Result struct {
	Index     int
	Duration  time.Duration
	Bytes     int64
	SpeedMBps float64
	LatencyMs float64
	Packets   int64
	StartTime time.Time
}

type TestType string

const (
	TestThroughput TestType = "throughput"
	TestLatency    TestType = "latency"
	TestConcurrent TestType = "concurrent"
)

const (
	CmdStartTest byte = 0x01
	CmdStopTest  byte = 0x02
	CmdResult    byte = 0x03
)
