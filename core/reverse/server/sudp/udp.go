package sudp

import (
	"context"
	"golang.org/x/net/quic"
	"net"
)

type Server struct {
	Ctx      context.Context
	Cancel   context.CancelFunc
	Conn     *quic.Conn
	Listener *net.UDPConn
	EventCh  chan string
}

func (s *Server) Run() error {

}

func (s *Server) Close() error {

}
