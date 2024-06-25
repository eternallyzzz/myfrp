package control

import (
	"context"
	"endpoint/core/reverse"
	"endpoint/pkg/common"
	"endpoint/pkg/model"
	"errors"
	"reflect"
	"sync"
)

type proxyServer struct {
	Ctx         context.Context
	ProxyConfig *model.Proxy
}

func (p *proxyServer) Run() error {
	remoteInfo := make(map[string]*model.RemoteInfo)

	for _, info := range p.ProxyConfig.Remote {
		remoteInfo[info.Tag] = info
	}

	hm := make(map[string][]*model.Service)
	for _, service := range p.ProxyConfig.Local {
		service.NodePort = 0
		hm[service.RemoteTag] = append(hm[service.RemoteTag], service)
	}

	var errs []error
	var wg sync.WaitGroup
	for tag, services := range hm {
		info := remoteInfo[tag]
		wg.Add(1)
		go reverse.DoReverseCli(p.Ctx, info, services, errs, &wg)
	}
	wg.Wait()

	return errors.Join(errs...)
}

func (p *proxyServer) Close() error {
	return nil
}

func proxyServerCreator(ctx context.Context, v any) (any, error) {
	proxy := v.(*model.Proxy)
	return &proxyServer{Ctx: ctx, ProxyConfig: proxy}, nil
}

func init() {
	pc := reflect.TypeOf(&model.Proxy{})
	common.ServerContext[pc] = proxyServerCreator
}
