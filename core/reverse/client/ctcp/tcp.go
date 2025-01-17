package ctcp

import (
	"context"
	"endpoint/pkg/common"
	"endpoint/pkg/config"
	"endpoint/pkg/model"
	"endpoint/pkg/zlog"
	"errors"
	"fmt"
	"golang.org/x/net/quic"
	"net"
	"time"
)

type Server struct {
	Ctx            context.Context
	Cancel         context.CancelFunc
	Endpoint       *quic.Endpoint
	Conn           *quic.Conn
	LocalSrvConfig *model.Service
	RemoteHost     string
	Running        bool
}

func (s *Server) Run() error {
	zlog.Warn(fmt.Sprintf("local service [%s]%s ——> remote Connection Addr: [%s]%s", s.LocalSrvConfig.Protocol,
		s.LocalSrvConfig.String(), s.LocalSrvConfig.Protocol, fmt.Sprintf("%s:%d", s.RemoteHost, s.LocalSrvConfig.RemotePort)))

	dial, err := common.PreMsg(s.Ctx, s.Endpoint, &model.NetAddr{Address: s.RemoteHost, Port: s.LocalSrvConfig.NodePort}, s.LocalSrvConfig.ID, s.LocalSrvConfig.Protocol)
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
		readByte, err := stream.ReadByte()
		if err != nil {
			return
		}

		switch readByte {
		case config.MsgType:
			go common.Heartbeat(stream)
			go common.HandleSrvEvent(stream)
			break
		case config.ContentType:
			dial, err := net.Dial(s.LocalSrvConfig.Protocol, s.LocalSrvConfig.String())
			if err != nil {
				_ = stream.Close()
				zlog.Error(err.Error())
				time.Sleep(time.Second)
				continue
			}

			p := &common.Pipe{Stream: stream}

			go common.Copy(dial, p, nil, "")
			break
		}
	}
}
