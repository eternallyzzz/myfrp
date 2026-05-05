package sudp

import (
	"context"
	"endpoint/pkg/common"
	"endpoint/pkg/config"
	"endpoint/pkg/kit/encode"
	"endpoint/pkg/zlog"
	"errors"
	"fmt"
	"net"
	"sync"
	"time"

	"golang.org/x/net/quic"
)

var udpReadBufferPool = sync.Pool{
	New: func() interface{} {
		return make([]byte, 1500)
	},
}

type Server struct {
	ID          string
	Ctx         context.Context
	Cancel      context.CancelFunc
	Conn        *quic.Conn
	Listener    *net.UDPConn
	EventCh     chan string
	Running     bool
	Conns       *sync.Map
	UDPConnMaps map[string]*WorkConnState
	Lock        sync.RWMutex // 使用读写锁，优化读多写少场景
}

func (s *Server) Run() error {
	s.Lock.Lock()
	defer s.Lock.Unlock()

	if s.Running {
		return nil
	}

	s.UDPConnMaps = make(map[string]*WorkConnState)

	stream, err := s.Conn.NewStream(s.Ctx)
	if err != nil {
		return err
	}
	_, err = stream.Write([]byte{config.MsgType})
	if err != nil {
		return err
	}
	stream.Flush()

	go s.handleHeartbeat(stream)
	go common.HandleEvent(stream, s.EventCh)
	go s.listenUDP()
	go s.checkInactiveStreams()

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

	s.Running = false

	errs := make([]error, 2)

	s.Cancel()
	errs[0] = s.Conn.Close()
	errs[1] = s.Listener.Close()

	zlog.Info(fmt.Sprintf("UDP Listener: %s Closed", s.Listener.LocalAddr().String()))

	return errors.Join(errs...)
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

func (s *Server) listenUDP() {
	defer func(s *Server) {
		err := s.Close()
		if err != nil {
			zlog.Error(err.Error())
		}
	}(s)

	buff := udpReadBufferPool.Get().([]byte)
	defer udpReadBufferPool.Put(buff)

	for {
		n, addr, err := s.Listener.ReadFromUDP(buff)
		if err != nil {
			return
		}

		data := make([]byte, n)
		copy(data, buff[:n])

		s.Lock.RLock()
		if v, ok := s.UDPConnMaps[addr.String()]; ok {
			s.Lock.RUnlock()
			select {
			case v.ReadCh <- data:
			default:
			}
			continue
		}
		s.Lock.RUnlock()

		stream, err := s.Conn.NewStream(s.Ctx)
		if err != nil {
			return
		}
		_, err = stream.Write([]byte{config.ContentType})
		if err != nil {
			return
		}
		stream.Flush()

		var wg sync.WaitGroup
		udpConnState := &WorkConnState{
			Ts:      time.Now(),
			Addr:    addr,
			UDPConn: s.Listener,
			Stream:  stream,
			ReadCh:  make(chan []byte, 256),
			WriteCh: make(chan []byte, 256),
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

		s.EventCh <- fmt.Sprintf("%s connected", addr.String())
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
	ticker := time.NewTicker(time.Second)

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
					s.EventCh <- fmt.Sprintf("%s disconnected", state.Addr.String())
					go state.Close()
				}
			}
			s.Lock.Unlock()
		}
	}
}
