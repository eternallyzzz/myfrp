package reverse

import (
	"context"
	"endpoint/pkg/config"
	"endpoint/pkg/kit/common"
	"endpoint/pkg/kit/net"
	"endpoint/pkg/model"
	"golang.org/x/net/quic"
)

func DoReverseSrv(p *model.Proxy) (*model.RemoteProxy, error) {
	rProxy := &model.RemoteProxy{Type: p.Type, Transfer: &model.NetAddr{Port: net.GetFreePort()}}

	endpoint, err := common.GetEndpoint(rProxy.Transfer)
	if err != nil {
		return nil, err
	}

	var rServicces []*model.RemoteService
	for _, service := range p.LocalServices {
		switch service.Protocol {
		case config.NetworkUDP:
			break
		case config.NetworkTCP:

			break
		}
	}
	go acceptQUICConnections(endpoint)

}

func acceptQUICConnections(endpoint *quic.Endpoint) {

}

type RpClient struct {
	Ctx         context.Context
	Endpoint    *quic.Endpoint
	Dial        *quic.Conn
	LocalProxy  *model.Service
	RemoteProxy *model.Service
}

func (r *RpClient) Run() error {

}

func (r *RpClient) Close() error {
}
