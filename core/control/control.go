package control

import (
	"context"
	"encoding/json"
	"endpoint/core/reverse"
	"endpoint/pkg/common"
	"endpoint/pkg/config"
	"endpoint/pkg/kit/net"
	"endpoint/pkg/model"
	"endpoint/pkg/zlog"
	"errors"
	"go.uber.org/zap"
	"golang.org/x/net/quic"
	"reflect"
	"time"
)

type Listen struct {
	Ctx      context.Context
	Endpoint *quic.Endpoint
}

func (l *Listen) Run() error {
	go func() {
		for {
			conn, err := l.Endpoint.Accept(l.Ctx)
			if err != nil {
				select {
				case <-l.Ctx.Done():
					return
				default:
					zlog.Error("failed accept quic", zap.Error(err))
					time.Sleep(time.Second)
					continue
				}
			}
			go handleConn(l.Ctx, conn)
		}
	}()
	return nil
}

func (l *Listen) Close() error {
	err := l.Endpoint.Close(l.Ctx)
	if err != nil {
		return err
	}

	return nil
}

func handleConn(ctx context.Context, conn *quic.Conn) {
	defer conn.Close()

	stream, err := conn.AcceptStream(ctx)
	if err != nil {
		select {
		case <-ctx.Done():
			return
		default:
			zlog.Error("failed accept stream", zap.Error(err))
		}
	}
	defer stream.Close()

	buff := make([]byte, 1500)
	var proxy model.Proxy

	timeout, cancelFunc := context.WithTimeout(ctx, time.Second*5)
	defer cancelFunc()
	stream.SetReadContext(timeout)

	n, err := stream.Read(buff)
	if err != nil {
		zlog.Error(err.Error())
		return
	}

	err = json.Unmarshal(buff[:n], &proxy)
	if err != nil {
		zlog.Error(err.Error())
		return
	}

	var rProxy *model.RemoteProxy

	switch proxy.Type {
	case config.Reverse:
		rProxy, err = reverse.DoReverseSrv(ctx, &proxy)
		break
	case config.Forward:
		// TODO
		break
	}

	if err != nil {
		zlog.Error(err.Error())
		_, _ = stream.Write([]byte(config.BadService))
	} else {
		m, er := json.Marshal(rProxy)
		if er != nil {
			zlog.Error(err.Error())
			_, _ = stream.Write([]byte(config.BadService))
		} else {
			_, _ = stream.Write(m)
		}
	}
	stream.Flush()

	time.Sleep(time.Second * 5)
}

func ListenCreator(ctx context.Context, v any) (any, error) {
	listenConfig, ok := v.(*model.ListenControl)
	if !ok {
		return nil, errors.New("invalid config type")
	}

	endpoint, err := common.GetEndpoint(listenConfig.NetAddr)
	if err != nil {
		return nil, err
	}

	return &Listen{ctx, endpoint}, nil
}

func ConnCreator(ctx context.Context, v any) (any, error) {
	connConfig, ok := v.(*model.ConnControl)
	if !ok {
		return nil, errors.New("invalid config type")
	}

	endpoint, err := common.GetEndpoint(connConfig.NetAddr)
	if err != nil {
		return nil, nil
	}
	defer endpoint.Close(ctx)

	dial, err := common.GetEndPointDial(ctx, endpoint, &model.NetAddr{Port: net.GetFreePort()})
	if err != nil {
		return nil, nil
	}
	defer dial.Close()

	rpClient, err := reverse.DoReverseCli(ctx, dial, connConfig.Proxy)

	return rpClient, err
}

func init() {
	lc := reflect.TypeOf(&model.ListenControl{})
	common.ServerContext[lc] = ListenCreator
	cc := reflect.TypeOf(&model.Proxy{})
	common.ServerContext[cc] = ConnCreator
}
