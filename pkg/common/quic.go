package common

import (
	"context"
	"endpoint/pkg/config"
	"endpoint/pkg/kit/tls"
	"endpoint/pkg/model"
	"golang.org/x/net/quic"
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
