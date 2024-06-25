package stcp

import (
	"context"
	"endpoint/pkg/common"
	"endpoint/pkg/config"
	"endpoint/pkg/kit/encode"
	"endpoint/pkg/zlog"
	"errors"
	"fmt"
	"golang.org/x/net/quic"
	"net"
	"sync"
	"time"
)

type Server struct {
	ID       string
	Ctx      context.Context
	Cancel   context.CancelFunc
	Conn     *quic.Conn
	Listener net.Listener
	EventCh  chan string
	Running  bool
	Conns    *sync.Map
	Lock     *sync.Mutex
}

func (s *Server) Run() error {
	s.Lock.Lock()
	defer s.Lock.Unlock()

	stream, err := s.Conn.NewStream(s.Ctx)
	if err != nil {
		return err
	}
	_, err = stream.Write([]byte{config.MsgType})
	if err != nil {
		return err
	}
	stream.Flush()

	go common.HandleEvent(stream, s.EventCh)
	go s.handleHeartbeat(stream)
	go s.listenTCP()

	s.Running = true
	return nil
}

func (s *Server) Close() error {
	s.Lock.Lock()
	defer s.Lock.Unlock()

	s.Conns.Delete(s.ID)

	if !s.Running {
		return nil
	}

	errs := make([]error, 2)

	s.Cancel()
	errs[0] = s.Conn.Close()
	errs[1] = s.Listener.Close()
	close(s.EventCh)

	s.Running = !s.Running
	zlog.Info(fmt.Sprintf("TCP Listener: %s Closed", s.Listener.Addr().String()))

	return errors.Join(errs...)
}

func (s *Server) listenTCP() {
	defer func(s *Server) {
		err := s.Close()
		if err != nil && !zlog.Ignore(err) {
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
		_, err = stream.Write([]byte{config.ContentType})
		if err != nil {
			return
		}
		stream.Flush()

		p := &common.Pipe{Stream: stream}

		s.EventCh <- fmt.Sprintf("%s connected", accept.RemoteAddr().String())

		go common.Copy(p, accept, s.EventCh, accept.RemoteAddr().String())
	}
}

func (s *Server) handleHeartbeat(stream *quic.Stream) {
	defer func(s *Server) {
		err := s.Close()
		if err != nil {
			if err != nil && !zlog.Ignore(err) {
				zlog.Error(err.Error())
			}
		}
	}(s)

	for {
		timeout, cancelFunc := context.WithTimeout(s.Ctx, time.Second*5)
		stream.SetReadContext(timeout)

		_, err := encode.Decode(stream)
		if err != nil {
			cancelFunc()
			return
		}
	}
}
