package control

import (
	"context"
	"encoding/json"
	"endpoint/core/reverse"
	"endpoint/pkg/config"
	"endpoint/pkg/kit/common"
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
		rProxy, err = reverse.DoReverseSrv(&proxy)
		break
	case config.Forward:
		// TODO
		break
	}

	if err != nil {
		zlog.Error(err.Error())
		_, _ = stream.Write([]byte(config.BadService))
		stream.Flush()
	} else {
		m, er := json.Marshal(rProxy)
		if er != nil {
			zlog.Error(err.Error())
			_, _ = stream.Write([]byte(config.BadService))
			stream.Flush()
		} else {
			_, _ = stream.Write(m)
		}
	}

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

	rpClients, err := handshake(ctx, dial, connConfig.Proxy)

	return rpClients, err
}

func handshake(ctx context.Context, dial *quic.Conn, p *model.Proxy) ([]*reverse.RpClient, error) {
	stream, err := dial.NewStream(ctx)
	if err != nil {
		return nil, err
	}

	m, err := json.Marshal(p)
	if err != nil {
		return nil, err
	}

	_, err = stream.Write(m)
	if err != nil {
		return nil, err
	}

	buff := make([]byte, 1500)
	var rProxy model.RemoteProxy

	timeout, cancel := context.WithTimeout(ctx, time.Second*5)
	defer cancel()
	stream.SetReadContext(timeout)

	n, err := stream.Read(buff)
	if err != nil {
		return nil, err
	}

	err = json.Unmarshal(buff[:n], &rProxy)
	if err != nil {
		return nil, err
	}

	var pcs []*model.ProxyConfig

	if rProxy.Type == p.Type {
		serviceMap := make(map[string]*model.Service)
		for _, service := range p.LocalServices {
			serviceMap[service.Tag] = service
		}

		for _, rService := range rProxy.RemoteServices {
			if localService, found := serviceMap[rService.Tag]; found {
				pcs = append(pcs, &model.ProxyConfig{
					Local: localService,
					Remote: &model.RemoteService{
						Tag:    rService.Tag,
						Listen: rService.Listen,
					},
					Transfer: rProxy.Transfer,
				})
			}
		}
	}

	var rpClis []*reverse.RpClient
	for _, pc := range pcs {
		endpoint, err := common.GetEndpoint(&model.NetAddr{Port: net.GetFreePort()})
		if err != nil {
			return nil, err
		}

		pointDial, err := common.GetEndPointDial(ctx, endpoint, pc.Transfer)
		if err != nil {
			return nil, err
		}

		rpClis = append(rpClis, &reverse.RpClient{
			Ctx:         ctx,
			Endpoint:    endpoint,
			Dial:        pointDial,
			LocalProxy:  pc.Local,
			RemoteProxy: pc.Remote.Listen,
		})
	}
	return rpClis, nil
}

func init() {
	lc := reflect.TypeOf(&model.ListenControl{})
	common.ServerContext[lc] = ListenCreator
	cc := reflect.TypeOf(&model.Proxy{})
	common.ServerContext[cc] = ConnCreator
}
