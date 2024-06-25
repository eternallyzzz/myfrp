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
	l, err := quic.Listen(config.NetworkQUIC, addr.String(), getSrvCfg())
	if err != nil {
		return nil, err
	}
	return l, nil
}

func GetEndPointDial(ctx context.Context, endpoint *quic.Endpoint, addr *model.NetAddr) (*quic.Conn, error) {
	dial, err := endpoint.Dial(ctx, config.NetworkQUIC, addr.String(), getCliCfg(addr.Address))
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

func PreMsg(ctx context.Context, endpoint *quic.Endpoint, addr *model.NetAddr, id, network string) (*quic.Conn, error) {
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
		ID:      id,
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

func getSrvCfg() *quic.Config {
	q := quic.Config{
		TLSConfig: tls.GetTLSConfig(config.ServerTLS, ""),
	}
	convertToQUIC(&q)

	if config.QUICCfg.TLS != nil {
		q.TLSConfig = tls.GetTLSConfigWithCustom(config.ServerTLS, "", config.QUICCfg.TLS.Crt, config.QUICCfg.TLS.Key)
	}

	return &q
}

func getCliCfg(addr string) *quic.Config {
	q := quic.Config{
		TLSConfig: tls.GetTLSConfig(config.ClientTLS, addr),
	}
	convertToQUIC(&q)
	return &q
}

func convertToQUIC(q *quic.Config) {
	if config.QUICCfg != nil {
		q.MaxStreamReadBufferSize = config.QUICCfg.MaxStreamReadBufferSize
		q.MaxStreamWriteBufferSize = config.QUICCfg.MaxStreamWriteBufferSize
		q.MaxConnReadBufferSize = config.QUICCfg.MaxConnReadBufferSize
		q.MaxBidiRemoteStreams = config.QUICCfg.MaxBidiRemoteStreams
		q.MaxUniRemoteStreams = config.QUICCfg.MaxUniRemoteStreams
		q.HandshakeTimeout = config.QUICCfg.HandshakeTimeout
		q.MaxIdleTimeout = config.QUICCfg.MaxIdleTimeout
		q.KeepAlivePeriod = config.QUICCfg.KeepAlivePeriod
		q.RequireAddressValidation = config.QUICCfg.RequireAddressValidation
	}

	if config.QUICCfg.MaxBidiRemoteStreams == 0 {
		q.MaxBidiRemoteStreams = config.MaxStreams
	}

	if config.QUICCfg.MaxIdleTimeout == 0 {
		q.MaxIdleTimeout = config.MaxIdle
	}

	if config.QUICCfg.KeepAlivePeriod == 0 {
		q.KeepAlivePeriod = config.KeepAlive
	}
}
