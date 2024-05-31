package sudp

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
	Ctx         context.Context
	Cancel      context.CancelFunc
	Conn        *quic.Conn
	Listener    *net.UDPConn
	EventCh     chan string
	Running     bool
	UDPConnMaps map[string]*WorkConnState
	Lock        *sync.Mutex
}

func (s *Server) Run() error {
	s.UDPConnMaps = make(map[string]*WorkConnState)

	sendStream, err := s.Conn.NewSendOnlyStream(s.Ctx)
	if err != nil {
		return err
	}

	go common.HandleEvent(sendStream, s.EventCh)
	go s.listenUDP()
	go s.checkInactiveStreams()

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

	s.Running = !s.Running

	return errors.Join(errs...)
}

func (s *Server) listenUDP() {
	defer func(s *Server) {
		err := s.Close()
		if err != nil {
			zlog.Error(err.Error())
		}
	}(s)

	buff := make([]byte, 1500)

	for {
		n, addr, err := s.Listener.ReadFromUDP(buff)
		if err != nil {
			return
		}

		data := make([]byte, n)
		copy(data, buff[:n])

		s.Lock.Lock()
		if v, ok := s.UDPConnMaps[addr.String()]; ok {
			select {
			case v.ReadCh <- data:
			default:
			}
			s.Lock.Unlock()
			continue
		}
		s.Lock.Unlock()

		stream, err := s.Conn.NewStream(s.Ctx)
		if err != nil {
			return
		}

		var wg sync.WaitGroup
		udpConnState := &WorkConnState{
			Ts:      time.Now(),
			Addr:    addr,
			UDPConn: s.Listener,
			Stream:  stream,
			ReadCh:  make(chan []byte, 2048),
			WriteCh: make(chan []byte, 2048),
			Wait:    &wg,
		}

		udpConnState.Wait.Add(3)
		go udpConnState.Write()
		go udpConnState.Read()
		go udpConnState.OutUDP()

		udpConnState.ReadCh <- data

		s.Lock.Lock()
		s.UDPConnMaps[addr.String()] = udpConnState
		s.Lock.Unlock()

		s.EventCh <- fmt.Sprintf("%s=%d*", addr.String(), 0)
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

func (w *WorkConnState) Write() {
	defer w.Wait.Done()
	defer close(w.WriteCh)

	for {
		select {
		case v, ok := <-w.ReadCh:
			if !ok {
				return
			}
			w.Ts = time.Now()

			_, err := w.Stream.Write(encode.Encode(v))
			if err != nil {
				return
			}
			w.Stream.Flush()
		}
	}
}

func (w *WorkConnState) Read() {
	defer w.Wait.Done()
	defer close(w.ReadCh)

	for {
		decode, err := encode.Decode(w.Stream)
		if err != nil {
			return
		}

		w.WriteCh <- decode
	}
}

func (w *WorkConnState) OutUDP() {
	defer w.Wait.Done()

	for {
		select {
		case v, ok := <-w.WriteCh:
			if !ok {
				return
			}

			_, err := w.UDPConn.WriteTo(v, w.Addr)
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

func (s *Server) checkInactiveStreams() {
	ticker := time.NewTicker(time.Millisecond * 150)

	for {
		select {
		case <-s.Ctx.Done():
			s.Lock.Lock()
			for k, state := range s.UDPConnMaps {
				delete(s.UDPConnMaps, k)
				go state.Close()
			}
			s.Lock.Unlock()
			return
		case <-ticker.C:
			s.Lock.Lock()
			for k, state := range s.UDPConnMaps {
				if time.Now().Sub(state.Ts).Seconds() > config.UDPConnTimeOut {
					delete(s.UDPConnMaps, k)
					s.EventCh <- fmt.Sprintf("%s=%d*", state.Addr.String(), 1)
					go state.Close()
				}
			}
			s.Lock.Unlock()
		}
	}
}
