package ctcp

import (
	"context"
	"endpoint/pkg/common"
	"endpoint/pkg/model"
	"endpoint/pkg/zlog"
	"errors"
	"golang.org/x/net/quic"
	"net"
)

type Server struct {
	Ctx         context.Context
	Cancel      context.CancelFunc
	Endpoint    *quic.Endpoint
	Conn        *quic.Conn
	LocalProxy  *model.Service
	RemoteProxy *model.Service
	Transfer    *model.NetAddr
	Running     bool
}

func (s *Server) Run() error {
	dial, err := common.PreMsg(s.Ctx, s.Endpoint, s.Transfer, s.RemoteProxy.Tag, s.RemoteProxy.Protocol)
	if err != nil {
		return err
	}
	s.Conn = dial

	go s.listenQUIC()

	return nil
}

func (s *Server) Close() error {
	if !s.Running {
		return nil
	}

	errs := make([]error, 2)

	errs[0] = s.Conn.Close()
	errs[1] = s.Endpoint.Close(s.Ctx)
	s.Cancel()

	s.Running = !s.Running

	return errors.Join(errs...)
}

func (s *Server) listenQUIC() {
	defer func(s *Server) {
		err := s.Close()
		if err != nil {
			zlog.Error(err.Error())
		}
	}(s)

	for {
		stream, err := s.Conn.AcceptStream(s.Ctx)
		if err != nil {
			return
		}
		if stream.IsReadOnly() {
			go common.HandleSrvEvent(stream)
		} else {
			dial, err := net.Dial(s.LocalProxy.Protocol, s.LocalProxy.String())
			if err != nil {
				return
			}

			go common.Copy(dial, stream, nil, "")
		}
	}
}
