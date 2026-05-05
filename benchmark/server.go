package benchmark

import (
	"context"
	"fmt"
	"io"
	"time"

	"endpoint/pkg/config"
	"endpoint/pkg/kit/encode"
	tlsutil "endpoint/pkg/kit/tls"

	"golang.org/x/net/quic"
)

func RunServer(cfg *Config) error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	qcfg := &quic.Config{
		MaxBidiRemoteStreams:    1000,
		MaxIdleTimeout:       time.Minute * 30,
		KeepAlivePeriod:      time.Second * 20,
		TLSConfig:            tlsutil.GetTLSConfig(config.ServerTLS, ""),
	}

	l, err := quic.Listen(config.NetworkQUIC, fmt.Sprintf(":%d", cfg.Port), qcfg)
	if err != nil {
		return fmt.Errorf("listen failed: %w", err)
	}
	defer l.Close(ctx)

	fmt.Printf("Server listening on :%d\n", cfg.Port)

	for {
		conn, err := l.Accept(ctx)
		if err != nil {
			select {
			case <-ctx.Done():
				return nil
			default:
				return fmt.Errorf("accept failed: %w", err)
			}
		}

		go handleConn(ctx, conn)
	}
}

func handleConn(ctx context.Context, conn *quic.Conn) {
	defer conn.Close()

	for {
		stream, err := conn.AcceptStream(ctx)
		if err != nil {
			return
		}

		go handleStream(ctx, stream)
	}
}

func handleStream(ctx context.Context, stream *quic.Stream) {
	defer stream.Close()

	data, err := encode.Decode(stream)
	if err != nil {
		if err != io.EOF {
		}
		return
	}

	_, err = stream.Write(encode.Encode(data))
	if err != nil {
		return
	}
}
