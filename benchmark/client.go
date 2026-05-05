package benchmark

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"endpoint/pkg/config"
	"endpoint/pkg/kit/encode"
	tlsutil "endpoint/pkg/kit/tls"
	"endpoint/pkg/model"

	"golang.org/x/net/quic"
)

type Client struct {
	ctx    context.Context
	cancel context.CancelFunc
	cfg    *Config
}

func RunClient(cfg *Config) ([]Result, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute*5)
	defer cancel()

	addrModel := &model.NetAddr{Address: cfg.Addr, Port: uint16(cfg.Port)}
	conn, err := getClientConn(ctx, addrModel)
	if err != nil {
		return nil, fmt.Errorf("dial failed: %w", err)
	}
	defer conn.Close()

	var results []Result
	switch cfg.TestType {
	case "throughput":
		results, err = runThroughputTest(ctx, conn, cfg)
	case "latency":
		results, err = runLatencyTest(ctx, conn, cfg)
	case "concurrent":
		results, err = runConcurrentTest(ctx, conn, cfg)
	default:
		results, err = runThroughputTest(ctx, conn, cfg)
	}

	return results, err
}

func runThroughputTest(ctx context.Context, conn *quic.Conn, cfg *Config) ([]Result, error) {
	var results []Result

	for i := 0; i < cfg.Count; i++ {
		stream, err := conn.NewStream(ctx)
		if err != nil {
			return nil, fmt.Errorf("new stream failed: %w", err)
		}

		data := make([]byte, cfg.Size)
		for j := 0; j < len(data); j += 1024 {
			pattern := byte(j >> 10)
			for k := 0; k < 1024 && j+k < len(data); k++ {
				data[j+k] = pattern
			}
		}

		start := time.Now()

		_, err = stream.Write(encode.Encode(data))
		if err != nil {
			stream.Close()
			return nil, fmt.Errorf("write failed: %w", err)
		}

		reply, err := encode.Decode(stream)
		if err != nil {
			stream.Close()
			return nil, fmt.Errorf("read reply failed: %w", err)
		}

		if len(reply) != len(data) {
			stream.Close()
			return nil, fmt.Errorf("reply size mismatch")
		}

		stream.Close()

		duration := time.Since(start)
		totalBytes := cfg.Size * 2
		speed := float64(totalBytes) / duration.Seconds() / 1024 / 1024

		result := Result{
			Index:     i + 1,
			Duration:  duration,
			Bytes:     totalBytes,
			SpeedMBps: speed,
			StartTime: start,
		}
		results = append(results, result)

		fmt.Printf("Run %d: %.2f MB/s (%.2fx %.1fMB in %v)\n",
			result.Index, speed, float64(totalBytes)/float64(cfg.Size), float64(cfg.Size)/1024/1024, duration.Round(time.Millisecond))
	}

	return results, nil
}

func runLatencyTest(ctx context.Context, conn *quic.Conn, cfg *Config) ([]Result, error) {
	pkgSize := int64(64)
	if cfg.PkgSize > 0 {
		pkgSize = int64(cfg.PkgSize)
	}

	data := make([]byte, pkgSize)
	for i := range data {
		data[i] = byte(i)
	}

	var totalLatency int64
	var packets int64
	var mu sync.Mutex

	concurrency := 50
	pool := make(chan struct{}, concurrency)

	for i := 0; i < cfg.Count*100; i++ {
		pool <- struct{}{}
		go func() {
			defer func() { <-pool }()

			stream, err := conn.NewStream(ctx)
			if err != nil {
				return
			}

			start := time.Now()
			_, err = stream.Write(encode.Encode(data))
			if err != nil {
				stream.Close()
				return
			}

			_, err = encode.Decode(stream)
			stream.Close()

			latency := time.Since(start).Nanoseconds()
			mu.Lock()
			atomic.AddInt64(&totalLatency, latency)
			atomic.AddInt64(&packets, 1)
			mu.Unlock()
		}()
	}

	for i := 0; i < concurrency; i++ {
		pool <- struct{}{}
	}

	avgLatency := float64(totalLatency) / float64(packets) / 1_000_000
	result := Result{
		Index:     1,
		LatencyMs: avgLatency,
		Packets:   packets,
	}

	fmt.Printf("Average Latency: %.3f ms (%d packets)\n", avgLatency, packets)
	return []Result{result}, nil
}

func runConcurrentTest(ctx context.Context, conn *quic.Conn, cfg *Config) ([]Result, error) {
	var mu sync.Mutex
	var wg sync.WaitGroup
	var totalBytes int64
	var totalDuration time.Duration
	var firstStart time.Time
	var started bool

	streamCount := cfg.Concurrent
	pkgSize := cfg.Size / int64(streamCount)
	if pkgSize < 1024 {
		pkgSize = 1024
	}

	for i := 0; i < streamCount; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()

			stream, err := conn.NewStream(ctx)
			if err != nil {
				return
			}
			defer stream.Close()

			data := make([]byte, pkgSize)
			for j := 0; j < len(data); j++ {
				data[j] = byte(j + idx)
			}

			if !started {
				mu.Lock()
				if !started {
					firstStart = time.Now()
					started = true
				}
				mu.Unlock()
			}

			_, err = stream.Write(encode.Encode(data))
			if err != nil {
				return
			}

			reply, err := encode.Decode(stream)
			if err != nil {
				return
			}

			mu.Lock()
			atomic.AddInt64(&totalBytes, int64(len(reply))*2)
			mu.Unlock()
		}(i)
	}

	wg.Wait()
	if started {
		totalDuration = time.Since(firstStart)
	} else {
		totalDuration = time.Millisecond
	}

	speed := float64(totalBytes) / totalDuration.Seconds() / 1024 / 1024

	result := Result{
		Index:     1,
		Duration:  totalDuration,
		Bytes:     totalBytes,
		SpeedMBps: speed,
		Packets:   int64(streamCount),
		StartTime: firstStart,
	}

	fmt.Printf("Concurrent %d streams: %.2f MB/s (%.1f MB in %v)\n",
		streamCount, speed, float64(totalBytes)/1024/1024, totalDuration.Round(time.Millisecond))
	return []Result{result}, nil
}

func getClientConn(ctx context.Context, addr *model.NetAddr) (*quic.Conn, error) {
	qcfg := &quic.Config{
		MaxBidiRemoteStreams: 1000,
		MaxIdleTimeout:       time.Minute * 30,
		KeepAlivePeriod:      time.Second * 20,
		TLSConfig:            tlsutil.GetTLSConfig(config.ClientTLS, addr.Address),
	}

	ep, err := quic.Listen(config.NetworkQUIC, ":0", qcfg)
	if err != nil {
		return nil, err
	}

	return ep.Dial(ctx, config.NetworkQUIC, addr.String(), qcfg)
}
