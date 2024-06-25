package cudp

import (
	"context"
	"endpoint/pkg/common"
	"endpoint/pkg/config"
	"endpoint/pkg/kit/encode"
	"endpoint/pkg/kit/id"
	"endpoint/pkg/model"
	"endpoint/pkg/zlog"
	"errors"
	"fmt"
	"golang.org/x/net/quic"
	"net"
	"sync"
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
	UDPConnMaps    *sync.Map
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
	go s.checkInactiveStreams()

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
			var wg sync.WaitGroup

			conn, err := udpConnect(s.LocalSrvConfig.String())
			if err != nil {
				_ = stream.Close()
				zlog.Error(err.Error())
				time.Sleep(time.Second)
				continue
			}

			udpConnState := &WorkConnState{
				ID:      id.GetSnowflakeID().String(),
				Ts:      time.Now(),
				Stream:  stream,
				UDPConn: conn,
				Wait:    &wg,
				ReadCh:  make(chan []byte, 2048),
				WriteCh: make(chan []byte, 2048),
			}

			wg.Add(4)
			s.UDPConnMaps.Store(udpConnState.ID, udpConnState)

			go udpConnState.Read()
			go udpConnState.Write()
			go udpConnState.InUDP()
			go udpConnState.OutUDP()
			break
		}
	}
}

type WorkConnState struct {
	ID      string
	Ts      time.Time
	Stream  *quic.Stream
	UDPConn *net.UDPConn
	ReadCh  chan []byte
	WriteCh chan []byte
	Wait    *sync.WaitGroup
}

func (w *WorkConnState) Read() {
	defer w.Wait.Done()
	defer w.UDPConn.Close()

	for {
		decode, err := encode.Decode(w.Stream)
		if err != nil {
			return
		}

		w.ReadCh <- decode
	}
}

func (w *WorkConnState) Write() {
	defer w.Wait.Done()
	defer close(w.ReadCh)

	for {
		select {
		case v, ok := <-w.WriteCh:
			if !ok {
				return
			}

			_, err := w.Stream.Write(encode.Encode(v))
			if err != nil {
				return
			}
			w.Stream.Flush()
		}
	}
}

func (w *WorkConnState) InUDP() {
	defer w.Wait.Done()
	defer close(w.WriteCh)

	buff := make([]byte, 1500)
	for {
		n, _, err := w.UDPConn.ReadFromUDP(buff)
		if err != nil {
			return
		}

		data := make([]byte, n)
		copy(data, buff[:n])

		select {
		case w.WriteCh <- data:
		default:
		}
	}
}

func (w *WorkConnState) OutUDP() {
	defer w.Wait.Done()

	for {
		select {
		case v, ok := <-w.ReadCh:
			if !ok {
				return
			}
			_, err := w.UDPConn.Write(v)
			if err != nil {
				return
			}
		}
	}
}

func (w *WorkConnState) Close() {
	_ = w.Stream.Close()
	w.Wait.Wait()
}

func udpConnect(addr string) (conn *net.UDPConn, err error) {
	udpAddr, err := net.ResolveUDPAddr("udp", addr)
	conn, err = net.DialUDP("udp", nil, udpAddr)
	return
}

func (s *Server) checkInactiveStreams() {
	ticker := time.NewTicker(time.Millisecond * 150)
	for {
		select {
		case <-s.Ctx.Done():
			closeInactiveStreams(s.UDPConnMaps, false)
			return
		case <-ticker.C:
			closeInactiveStreams(s.UDPConnMaps, true)
		}
	}
}

func closeInactiveStreams(users *sync.Map, checkActive bool) {
	f := func(value *WorkConnState, key any) {
		value.Stream.CloseRead()
		_ = value.Stream.Close()
		users.Delete(key)
	}

	users.Range(func(key, value any) bool {
		info := value.(*WorkConnState)
		if checkActive {
			if time.Now().Sub(info.Ts).Seconds() > config.UDPTimeOutCli {
				f(info, key)
			}
		} else {
			f(info, key)
		}
		return true
	})
}
