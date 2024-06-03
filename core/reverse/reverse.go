package reverse

import (
	"context"
	"encoding/json"
	"endpoint/core"
	"endpoint/core/reverse/client/ctcp"
	"endpoint/core/reverse/client/cudp"
	"endpoint/core/reverse/server/stcp"
	"endpoint/core/reverse/server/sudp"
	"endpoint/pkg/common"
	"endpoint/pkg/config"
	"endpoint/pkg/kit/net"
	"endpoint/pkg/model"
	"endpoint/pkg/zlog"
	"fmt"
	"go.uber.org/zap"
	"golang.org/x/net/quic"
	net2 "net"
	"strings"
	"sync"
	"time"
)

func DoReverseSrv(ctx context.Context, p *model.Proxy) (*model.RemoteProxy, error) {
	rProxy := &model.RemoteProxy{Type: p.Type, Transfer: &model.NetAddr{Port: net.GetFreePort()}}

	endpoint, err := common.GetEndpoint(rProxy.Transfer)
	if err != nil {
		return nil, err
	}

	ip, err := net.GetExternalIP()
	if err != nil {
		return nil, err
	}

	rProxy.Transfer.Address = ip

	var rServices []*model.RemoteService
	var listeners sync.Map
	for _, service := range p.LocalServices {
		switch service.Protocol {
		case config.NetworkUDP:
			listener, port, err := net.GetUdpListener()
			if err != nil {
				return nil, err
			}
			listeners.Store(service.Tag, listener)

			rServices = append(rServices, &model.RemoteService{
				Tag: service.Tag,
				Listen: &model.Service{
					Tag:      service.Tag,
					Listen:   ip,
					Port:     port,
					Protocol: listener.LocalAddr().Network(),
				},
			})
			break
		case config.NetworkTCP:
			listener, port, err := net.GetTcpListener()
			if err != nil {
				return nil, err
			}
			listeners.Store(service.Tag, listener)

			rServices = append(rServices, &model.RemoteService{
				Tag: service.Tag,
				Listen: &model.Service{
					Tag:      service.Tag,
					Listen:   ip,
					Port:     port,
					Protocol: listener.Addr().Network(),
				},
			})
			break
		}
	}

	rProxy.RemoteServices = rServices

	ctx, cancel := context.WithCancel(ctx)
	rpServer := &RpServer{Ctx: ctx, Cancel: cancel, ConnMap: &sync.Map{}}

	rpServer.Endpoint = endpoint
	rpServer.Listeners = &listeners

	instance := config.Ctx.Value("instance").(*core.Instance)
	err = instance.AddTask(rpServer)
	if err != nil {
		return nil, err
	}

	return rProxy, nil
}

func acceptQUICConnections(srv *RpServer) {
	for {
		accept, err := srv.Endpoint.Accept(srv.Ctx)
		if err != nil {
			return
		}

		go handleQUICConn(accept, srv)
	}
}

func handleQUICConn(conn *quic.Conn, srv *RpServer) {
	stream, err := conn.AcceptStream(srv.Ctx)
	if err != nil {
		return
	}
	defer stream.Close()

	buff := make([]byte, 64)

	timeout, cancelFunc := context.WithTimeout(srv.Ctx, time.Second*5)
	defer cancelFunc()

	stream.SetReadContext(timeout)

	n, err := stream.Read(buff)
	if err != nil {
		return
	}
	_, err = stream.Write([]byte{0x0})
	if err != nil {
		return
	}
	stream.Flush()

	var h model.Handshake
	err = json.Unmarshal(buff[:n], &h)
	if err != nil {
		return
	}

	srv.ConnMap.Store(fmt.Sprintf("%s:%s", h.Tag, h.Network), conn)
	v, ok := srv.Listeners.Load(h.Tag)

	if ok {
		instance := config.Ctx.Value("instance").(*core.Instance)
		switch h.Network {
		case config.NetworkTCP:
			listener := v.(net2.Listener)

			ctx, c := context.WithCancel(srv.Ctx)

			tsrv := &stcp.Server{
				Ctx:      ctx,
				Cancel:   c,
				Conn:     conn,
				Listener: listener,
				EventCh:  make(chan string, 10),
				Lock:     &sync.Mutex{},
			}

			err := instance.AddTask(tsrv)
			if err != nil {
				zlog.Error("rSrv failed star", zap.Error(err))
				return
			}
			break
		case config.NetworkUDP:
			listener := v.(*net2.UDPConn)

			ctx, c := context.WithCancel(srv.Ctx)

			usrv := &sudp.Server{
				Ctx:      ctx,
				Cancel:   c,
				Conn:     conn,
				Listener: listener,
				EventCh:  make(chan string, 10),
				Lock:     &sync.Mutex{},
			}

			err := instance.AddTask(usrv)
			if err != nil {
				zlog.Error("rSrv failed star", zap.Error(err))
				return
			}
			break
		}
	}
}

type RpServer struct {
	Ctx       context.Context
	Cancel    context.CancelFunc
	Endpoint  *quic.Endpoint
	Listeners *sync.Map
	ConnMap   *sync.Map
}

func (r *RpServer) Run() error {
	go acceptQUICConnections(r)
	return nil
}

func (r *RpServer) Close() error {
	var err error

	r.Cancel()

	r.ConnMap.Range(func(key, value any) bool {
		split := strings.Split(key.(string), ":")
		v, loaded := r.Listeners.LoadAndDelete(split[0])
		switch split[1] {
		case config.NetworkTCP:
			if loaded {
				err = v.(net2.Listener).Close()
			}
			break
		case config.NetworkUDP:
			if loaded {
				err = v.(*net2.UDPConn).Close()
			}
			break
		}

		err = value.(*quic.Conn).Close()
		return true
	})

	err = r.Endpoint.Close(r.Ctx)
	return err
}

func DoReverseCli(ctx context.Context, dial *quic.Conn, p *model.Proxy) (*RpClient, error) {
	stream, err := dial.NewStream(ctx)
	if err != nil {
		return nil, err
	}

	m, err := json.Marshal(p)
	if err != nil {
		return nil, err
	}

	_, err = stream.Write(m)
	if err != nil {
		return nil, err
	}
	stream.Flush()

	buff := make([]byte, 1500)
	var rProxy model.RemoteProxy

	timeout, cancel := context.WithTimeout(ctx, time.Second*5)
	defer cancel()
	stream.SetReadContext(timeout)

	n, err := stream.Read(buff)
	if err != nil {
		return nil, err
	}

	_, err = stream.Write([]byte{0x0})
	if err != nil {
		return nil, err
	}
	stream.Flush()

	err = json.Unmarshal(buff[:n], &rProxy)
	if err != nil {
		return nil, err
	}

	var pcs []*model.ProxyConfig

	ip, err := net.GetExternalIP()
	if err != nil {
		return nil, err
	}
	if rProxy.Transfer.Address == ip {
		rProxy.Transfer.Address = "127.0.0.1"
	}

	if rProxy.Type == p.Type {
		serviceMap := make(map[string]*model.Service)
		for _, service := range p.LocalServices {
			serviceMap[service.Tag] = service
		}

		for _, rService := range rProxy.RemoteServices {
			if localService, found := serviceMap[rService.Tag]; found {
				if rService.Listen.Listen == ip {
					rService.Listen.Listen = "127.0.0.1"
				}

				pcs = append(pcs, &model.ProxyConfig{
					Local: localService,
					Remote: &model.RemoteService{
						Tag:    rService.Tag,
						Listen: rService.Listen,
					},
					Transfer: rProxy.Transfer,
				})
			}
		}
	}

	withCancel, cancelFun := context.WithCancel(ctx)
	rpCli := &RpClient{
		Ctx:       withCancel,
		Cancel:    cancelFun,
		RpContext: make([]*rpTemp, 0),
	}
	for _, pc := range pcs {
		endpoint, err := common.GetEndpoint(&model.NetAddr{Port: net.GetFreePort()})
		if err != nil {
			return nil, err
		}

		rpCli.RpContext = append(rpCli.RpContext, &rpTemp{
			Endpoint:    endpoint,
			LocalProxy:  pc.Local,
			RemoteProxy: pc.Remote.Listen,
			Network:     pc.Local.Protocol,
			Transfer:    pc.Transfer,
		})
	}
	return rpCli, nil
}

type rpTemp struct {
	Endpoint    *quic.Endpoint
	LocalProxy  *model.Service
	RemoteProxy *model.Service
	Network     string
	Transfer    *model.NetAddr
}

type RpClient struct {
	Ctx       context.Context
	Cancel    context.CancelFunc
	RpContext []*rpTemp
}

func (r *RpClient) Run() error {
	instance := config.Ctx.Value("instance").(*core.Instance)

	for _, rt := range r.RpContext {
		switch rt.Network {
		case config.NetworkTCP:
			ctx, c := context.WithCancel(r.Ctx)
			tsrv := &ctcp.Server{
				Ctx:         ctx,
				Cancel:      c,
				Endpoint:    rt.Endpoint,
				LocalProxy:  rt.LocalProxy,
				RemoteProxy: rt.RemoteProxy,
				Transfer:    rt.Transfer,
			}
			err := instance.AddTask(tsrv)
			if err != nil {
				return err
			}
			break
		case config.NetworkUDP:
			ctx, c := context.WithCancel(r.Ctx)
			usrv := &cudp.Server{
				Ctx:         ctx,
				Cancel:      c,
				Endpoint:    rt.Endpoint,
				LocalProxy:  rt.LocalProxy,
				RemoteProxy: rt.RemoteProxy,
				Transfer:    rt.Transfer,
				UDPConnMaps: &sync.Map{},
			}
			err := instance.AddTask(usrv)
			if err != nil {
				return err
			}
			break
		}
	}

	return nil
}

func (r *RpClient) Close() error {
	r.Cancel()

	return nil
}
