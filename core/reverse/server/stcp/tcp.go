package stcp

import (
	"context"
	"endpoint/pkg/common"
	"endpoint/pkg/zlog"
	"errors"
	"fmt"
	"golang.org/x/net/quic"
	"net"
)

type Server struct {
	Ctx      context.Context
	Cancel   context.CancelFunc
	Conn     *quic.Conn
	Listener net.Listener
	EventCh  chan string
	Running  bool
}

func (s *Server) Run() error {
	sendStream, err := s.Conn.NewSendOnlyStream(s.Ctx)
	if err != nil {
		return err
	}

	go common.HandleEvent(sendStream, s.EventCh)
	go s.listenTCP()

	s.Running = true
	return nil
}

func (s *Server) Close() error {
	if !s.Running {
		return nil
	}

	errs := make([]error, 2)

	s.Cancel()
	errs[0] = s.Conn.Close()
	errs[1] = s.Listener.Close()
	close(s.EventCh)

	s.Running = !s.Running

	return errors.Join(errs...)
}

func (s *Server) listenTCP() {
	defer func(s *Server) {
		err := s.Close()
		if err != nil {
			zlog.Error(err.Error())
		}
	}(s)

	for {
		accept, err := s.Listener.Accept()
		if err != nil {
			return
		}

		stream, err := s.Conn.NewStream(s.Ctx)
		if err != nil {
			return
		}

		s.EventCh <- fmt.Sprintf("%s=%d*", accept.RemoteAddr().String(), 0)

		go common.Copy(stream, accept, s.EventCh, accept.RemoteAddr().String())
	}
}
