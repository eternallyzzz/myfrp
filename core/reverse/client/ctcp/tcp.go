package ctcp

import (
	"context"
	"encoding/json"
	"endpoint/pkg/common"
	"endpoint/pkg/model"
	"golang.org/x/net/quic"
	"net"
)

type Server struct {
	Ctx         context.Context
	Cancel      context.CancelFunc
	Endpoint    *quic.Endpoint
	LocalProxy  *model.Service
	RemoteProxy *model.Service
	Transfer    *model.NetAddr
	LocalDial   net.Listener
}

func (s *Server) Run() error {

}

func (s *Server) Close() error {

}

func s() {
	withCancel, cancelFun := context.WithCancel(ctx)

	pointDial, err := common.GetEndPointDial(withCancel, endpoint, pc.Transfer)
	if err != nil {
		cancelFun()
		return nil, err
	}

	newStream, err := pointDial.NewStream(withCancel)
	if err != nil {
		cancelFun()
		return nil, err
	}

	h := model.Handshake{
		Tag:     pc.Remote.Tag,
		Network: pc.Remote.Listen.Protocol,
	}
	m1, err := json.Marshal(&h)
	if err != nil {
		cancelFun()
		return nil, err
	}

	_, err = newStream.Write(m1)
	if err != nil {
		cancelFun()
		return nil, err
	}

	b := make([]byte, 10)
	for {
		_, err := newStream.Read(b)
		if err != nil {
			break
		}
	}
}
