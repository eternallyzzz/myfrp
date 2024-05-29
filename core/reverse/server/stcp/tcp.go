package stcp

import (
	"context"
	"golang.org/x/net/quic"
	"net"
)

type Server struct {
	Ctx      context.Context
	Cancel   context.CancelFunc
	Conn     *quic.Conn
	Listener net.Listener
	EventCh  chan string
}

func (s *Server) Run() error {

}

func (s *Server) Close() error {

}

func fromTarget(l net.Listener) {
	for {
		accept, err := l.Accept()
		if err != nil {
			return
		}

	}
}

func toTarget() {

}

func fromClient() {

}

func toClient() {

}
