package common

import (
	"context"
	"encoding/json"
	"endpoint/pkg/config"
	"endpoint/pkg/kit/tls"
	"endpoint/pkg/model"
	"endpoint/pkg/zlog"
	"fmt"
	"golang.org/x/net/quic"
	"strings"
	"time"
)

func GetEndpoint(addr *model.NetAddr) (*quic.Endpoint, error) {
	if addr == nil {
		return nil, nil
	}
	l, err := quic.Listen(config.NetworkQUIC, addr.String(), &quic.Config{
		TLSConfig:            tls.GetTLSConfig(config.ServerTLS, ""),
		MaxIdleTimeout:       config.MaxIdle,
		KeepAlivePeriod:      config.KeepAlive,
		MaxBidiRemoteStreams: config.MaxStreams,
	})
	if err != nil {
		return nil, err
	}
	return l, nil
}

func GetEndPointDial(ctx context.Context, endpoint *quic.Endpoint, addr *model.NetAddr) (*quic.Conn, error) {
	dial, err := endpoint.Dial(ctx, config.NetworkQUIC, addr.String(), &quic.Config{
		TLSConfig:       tls.GetTLSConfig(config.ClientTLS, addr.Address),
		MaxIdleTimeout:  config.MaxIdle,
		KeepAlivePeriod: config.KeepAlive,
	})
	return dial, err
}

func HandleEvent(stream *quic.Stream, eventCh chan string) {
	defer stream.Close()

	for {
		select {
		case v, ok := <-eventCh:
			if !ok {
				return
			}

			_, err := stream.Write([]byte(v))
			if err != nil {
				return
			}
		}
	}
}

func HandleSrvEvent(stream *quic.Stream) {
	defer stream.Close()

	buff := make([]byte, 64)

	for {
		n, err := stream.Read(buff)
		if err != nil {
			return
		}
		fetchData(buff[:n])
	}
}

func fetchData(data []byte) {
	split := strings.Split(string(data), "*")
	for _, sp := range split {
		i := strings.Split(sp, "=")
		if len(i) > 1 {
			switch i[1] {
			case "0":
				zlog.Info(fmt.Sprintf("%s connected", i[0]))
				break
			case "1":
				zlog.Info(fmt.Sprintf("%s disconnected", i[0]))
				break
			}
		}
	}
}

func PreMsg(ctx context.Context, endpoint *quic.Endpoint, addr *model.NetAddr, tag, network string) (*quic.Conn, error) {
	pointDial, err := GetEndPointDial(ctx, endpoint, addr)
	if err != nil {
		return nil, err
	}
	defer pointDial.Close()

	newStream, err := pointDial.NewStream(ctx)
	if err != nil {
		return nil, err
	}
	defer newStream.Close()

	h := model.Handshake{
		Tag:     tag,
		Network: network,
	}
	m, err := json.Marshal(&h)
	if err != nil {
		return nil, err
	}

	_, err = newStream.Write(m)
	if err != nil {
		return nil, err
	}
	newStream.Flush()

	time.Sleep(time.Second * 3)
	dial, err := GetEndPointDial(ctx, endpoint, addr)
	if err != nil {
		return nil, err
	}

	return dial, err
}
