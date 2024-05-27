package core

import (
	"context"
	"encoding/json"
	"endpoint/pkg/config"
	"endpoint/pkg/inf"
	"endpoint/pkg/kit/net"
	"endpoint/pkg/kit/tls"
	"endpoint/pkg/model"
	"endpoint/pkg/zlog"
	"errors"
	"fmt"
	"golang.org/x/net/quic"
	"reflect"
	"strings"
	"sync"
	"time"
)

func New(c *model.Config) (*Instance, error) {
	instance := &Instance{Ctx: context.Background(), LocalProxy: c.Proxy}

	endpoint, err := endPoint(c.Control.Listen)
	if err != nil {
		return nil, err
	}
	instance.EndPoint = endpoint

	rProxy, err := initProxyInfo(c)
	if err != nil {
		return nil, err
	}

	createServer(c.Proxy, rProxy)

	instance.RemoteProxy = rProxy

	return instance, nil
}

func endPoint(addr *model.NetAddr) (*quic.Endpoint, error) {
	if addr == nil {
		return nil, nil
	}
	endpoint, err := quic.Listen(config.NetworkQUIC, fmt.Sprintf("%s:%d", addr.Address, addr.Port), &quic.Config{
		TLSConfig:            tls.GetTLSConfig(config.ServerTLS, ""),
		MaxIdleTimeout:       config.MaxIdle,
		KeepAlivePeriod:      config.KeepAlive,
		MaxBidiRemoteStreams: config.MaxStreams,
	})
	if err != nil {
		return nil, err
	}
	return endpoint, nil
}

func initProxyInfo(c *model.Config) ([]*model.RemoteProxy, error) {
	if c.Control.Conn == nil {
		return nil, nil
	}

	buff := make([]byte, 1500)

	endpoint, err := endPoint(&model.NetAddr{Address: "", Port: net.GetFreePort()})
	if err != nil {
		return nil, err
	}
	defer endpoint.Close(context.Background())

	dial, err := endpoint.Dial(context.Background(), config.NetworkQUIC, fmt.Sprintf("%s:%d", c.Control.Conn.Address, c.Control.Conn.Port), &quic.Config{
		TLSConfig:       tls.GetTLSConfig(config.ClientTLS, c.Control.Conn.Address),
		MaxIdleTimeout:  config.MaxIdle,
		KeepAlivePeriod: config.KeepAlive,
	})
	if err != nil {
		return nil, err
	}

	m, err := json.Marshal(c.Proxy)
	if err != nil {
		return nil, err
	}

	stream, err := dial.NewStream(context.Background())
	if err != nil {
		return nil, err
	}
	defer stream.Close()

	_, err = stream.Write(m)
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	stream.SetReadContext(ctx)

	n, err := stream.Read(buff)
	if err != nil {
		return nil, err
	}

	var rProxy []*model.RemoteProxy
	err = json.Unmarshal(buff[:n], &rProxy)
	if err != nil {
		return nil, err
	}
	return rProxy, nil
}

type Instance struct {
	Lock        sync.Mutex
	Ctx         context.Context
	EndPoint    *quic.Endpoint
	Futures     []inf.Task
	RemoteProxy []*model.RemoteProxy
	LocalProxy  *model.LocalProxy
}

func (i *Instance) Start() error {
	i.Lock.Lock()
	defer i.Lock.Unlock()

	for _, task := range i.Futures {
		if err := task.Run(); err != nil {
			return err
		}
	}

	zlog.Info("Endpoint started")

	return nil
}

func (i *Instance) Close() error {
	i.Lock.Lock()
	defer i.Lock.Unlock()

	var errMsg string
	for _, task := range i.Futures {
		if err := task.Close(); err != nil {
			errMsg += " " + err.Error()
		}
	}

	if errMsg != "" {
		return errors.New(errMsg)
	}

	return nil
}

func createServer(lp *model.LocalProxy, rp []*model.RemoteProxy) {

	value := reflect.ValueOf(strings.ToLower(strings.TrimSpace(lp.Type)))
	value.MethodByName("Create")
}
