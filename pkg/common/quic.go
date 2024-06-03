package common

import (
	"context"
	"encoding/json"
	"endpoint/pkg/config"
	"endpoint/pkg/kit/encode"
	"endpoint/pkg/kit/tls"
	"endpoint/pkg/model"
	"endpoint/pkg/zlog"
	"golang.org/x/net/quic"
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

			_, err := stream.Write(encode.Encode([]byte(v)))
			if err != nil {
				return
			}
			stream.Flush()
		}
	}
}

func HandleSrvEvent(stream *quic.Stream) {
	defer stream.Close()

	for {
		decode, err := encode.Decode(stream)
		if err != nil {
			return
		}
		zlog.Info(string(decode))
	}
}

func PreMsg(ctx context.Context, endpoint *quic.Endpoint, addr *model.NetAddr, tag, network string) (*quic.Conn, error) {
	pointDial, err := GetEndPointDial(ctx, endpoint, addr)
	if err != nil {
		return nil, err
	}

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

	buff := make([]byte, 2)
	_, err = newStream.Read(buff)
	if err != nil {
		return nil, err
	}

	return pointDial, err
}

func Heartbeat(stream *quic.Stream) {
	defer stream.Close()

	for {
		_, err := stream.Write(encode.Encode([]byte("PING")))
		if err != nil {
			return
		}
		stream.Flush()

		time.Sleep(time.Second * 3)
	}
}
