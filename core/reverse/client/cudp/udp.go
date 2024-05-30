package cudp

import (
	"context"
	"endpoint/pkg/common"
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
	Ctx         context.Context
	Cancel      context.CancelFunc
	Endpoint    *quic.Endpoint
	Conn        *quic.Conn
	LocalProxy  *model.Service
	RemoteProxy *model.Service
	Transfer    *model.NetAddr
	Running     bool
	Users       *sync.Map
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
			var wg sync.WaitGroup

			conn, err := udpConnect(s.LocalProxy.Port)
			if err != nil {
				_ = stream.Close()
				zlog.Error(err.Error())
				continue
			}

			wg.Add(4)
			u := newUserInfo(stream, conn, &wg)
			s.Users.Store(u.ID, u)

			go u.fromLocalServer()
			go u.fromRemoteServer()
			go u.toLocal()
			go u.toRemote()
			go checkInactiveStreams(s)
		}
	}
}

type WorkConnState struct {
	Ts      time.Time
	Addr    *net.UDPAddr
	Stream  *quic.Stream
	UDPConn *net.UDPConn
	ReadCh  chan []byte
	WriteCh chan []byte
	Wait    *sync.WaitGroup
}

type UserInfo struct {
	ID      string
	Stream  *quic.Stream
	WriteCh chan []byte
	ReadCh  chan []byte
	UdpConn *net.UDPConn
	Wg      *sync.WaitGroup
	Ts      int64
}

func (u *UserInfo) fromLocalServer() {
	defer func() {
		close(u.WriteCh)
		u.Wg.Done()
	}()
	buff := make([]byte, 1500)
	for {
		n, err := u.UdpConn.Read(buff)
		if err != nil {
			return
		}
		data := make([]byte, n)
		copy(data, buff[:n])
		select {
		case u.WriteCh <- data:
		default:
		}
	}
}

func (u *UserInfo) fromRemoteServer() {
	defer func() {
		close(u.ReadCh)
		u.Wg.Done()
	}()
	for {
		data, er := encode.Decode(u.Stream)
		if er != nil {
			return
		}

		u.Ts = time.Now().Unix()
		u.ReadCh <- data
	}
}

func (u *UserInfo) toRemote() {
	defer func() {
		_ = u.Stream.Close()
		u.Wg.Done()
	}()

	for {
		select {
		case v, ok := <-u.WriteCh:
			if !ok {
				return
			}
			v = encode.Encode(v)
			_, err := u.Stream.Write(v)
			if err != nil {
				return
			}
			u.Ts = time.Now().Unix()
		}
	}
}

func (u *UserInfo) toLocal() {
	defer func() {
		_ = u.UdpConn.Close()
		u.Wg.Done()
	}()
	for {
		select {
		case v, ok := <-u.ReadCh:
			if !ok {
				return
			}
			_, err := u.UdpConn.Write(v)
			if err != nil {
				zlog.Error(err.Error())
				return
			}
		}
	}
}

func newUserInfo(stream *quic.Stream, conn *net.UDPConn, wg *sync.WaitGroup) (u *UserInfo) {
	u = &UserInfo{
		ID:      id.GetSnowflakeID().String(),
		Stream:  stream,
		WriteCh: make(chan []byte, 2048),
		ReadCh:  make(chan []byte, 2048),
		UdpConn: conn,
		Wg:      wg,
		Ts:      time.Now().Unix(),
	}
	return
}

func udpConnect(proxyPort int) (conn *net.UDPConn, err error) {
	udpAddr, err := net.ResolveUDPAddr("udp", fmt.Sprintf("127.0.0.1:%d", proxyPort))
	conn, err = net.DialUDP("udp", nil, udpAddr)
	return
}

func checkInactiveStreams(s *Server) {
	ticker := time.NewTicker(time.Millisecond * 150)
	for {
		select {
		case <-s.Ctx.Done():
			closeInactiveStreams(s.Users, false)
			return
		case <-ticker.C:
			closeInactiveStreams(s.Users, true)
		}
	}
}

func closeInactiveStreams(users *sync.Map, checkActive bool) {
	f := func(value *UserInfo, key any) {
		value.Stream.CloseRead()
		_ = value.Stream.Close()
		users.Delete(key)
	}

	users.Range(func(key, value any) bool {
		info := value.(*UserInfo)
		if checkActive {
			if time.Now().Unix()-info.Ts > 5 {
				f(info, key)
			}
		} else {
			f(info, key)
		}
		return true
	})
}
