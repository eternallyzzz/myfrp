package cudp

import (
	"context"
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
	LocalDial   *net.UDPConn
}

func (s *Server) Run() error {

}

func (s *Server) Close() error {

}
