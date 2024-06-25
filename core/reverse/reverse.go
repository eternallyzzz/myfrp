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
	"endpoint/pkg/kit/id"
	"endpoint/pkg/kit/net"
	"endpoint/pkg/model"
	"endpoint/pkg/zlog"
	"golang.org/x/net/quic"
	"sync"
	"time"
)

func DoReverseSrv(ctx context.Context, services []*model.Service) error {
	nd := model.NetAddr{Port: uint16(net.GetFreePort())}

	data := make(map[string]*model.Service)
	for _, service := range services {
		service.NodePort = nd.Port
		if service.RemotePort == 0 {
			service.RemotePort = uint16(net.GetFreePort())
		}
		service.ID = id.GetSnowflakeID().String()
		data[service.ID] = service
	}

	endpoint, err := common.GetEndpoint(&nd)
	if err != nil {
		return err
	}

	server := nodeServer{Ctx: ctx, Endpoint: endpoint, Services: data, Conns: &sync.Map{}}

	instance := config.Ctx.Value("instance").(*core.Instance)
	err = instance.AddTask(&server)
	if err != nil {
		return err
	}

	return nil
}

func DoReverseCli(ctx context.Context, info *model.RemoteInfo, services []*model.Service, errs []error, wg *sync.WaitGroup) {
	defer wg.Done()

	endpoint, err := common.GetEndpoint(&model.NetAddr{Port: uint16(net.GetFreePort())})
	if err != nil {
		errs = append(errs, err)
		return
	}
	defer endpoint.Close(ctx)

	dial, err := common.GetEndPointDial(ctx, endpoint, info.NetAddr)
	if err != nil {
		errs = append(errs, err)
		return
	}
	defer dial.Close()

	newStream, err := dial.NewStream(ctx)
	if err != nil {
		errs = append(errs, err)
		return
	}
	defer newStream.Close()

	m, err := json.Marshal(services)
	if err != nil {
		errs = append(errs, err)
		return
	}

	_, err = newStream.Write(m)
	if err != nil {
		errs = append(errs, err)
		return
	}
	newStream.Flush()

	var data [1500]byte
	n, err := newStream.Read(data[:])
	if err != nil {
		errs = append(errs, err)
		return
	}
	err = json.Unmarshal(data[:n], &services)
	if err != nil {
		errs = append(errs, err)
		return
	}

	client := RpClient{
		Ctx:        ctx,
		Services:   services,
		RemoteHost: info.Address,
	}
	instance := config.Ctx.Value("instance").(*core.Instance)
	err = instance.AddTask(&client)
	if err != nil {
		errs = append(errs, err)
		return
	}
}

type RpClient struct {
	Ctx        context.Context
	Services   []*model.Service
	RemoteHost string
}

func (r *RpClient) Run() error {
	instance := config.Ctx.Value("instance").(*core.Instance)
	endpoint, err := common.GetEndpoint(&model.NetAddr{Port: uint16(net.GetFreePort())})
	if err != nil {
		return err
	}

	for _, service := range r.Services {
		switch service.Protocol {
		case config.NetworkTCP:
			ctx, c := context.WithCancel(r.Ctx)
			tsrv := &ctcp.Server{
				Ctx:            ctx,
				Cancel:         c,
				Endpoint:       endpoint,
				LocalSrvConfig: service,
				RemoteHost:     r.RemoteHost,
			}
			err := instance.AddTask(tsrv)
			if err != nil {
				return err
			}
			break
		case config.NetworkUDP:
			ctx, c := context.WithCancel(r.Ctx)
			usrv := &cudp.Server{
				Ctx:            ctx,
				Cancel:         c,
				Endpoint:       endpoint,
				LocalSrvConfig: service,
				RemoteHost:     r.RemoteHost,
				UDPConnMaps:    &sync.Map{},
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
	return nil
}

type nodeServer struct {
	Ctx      context.Context
	Endpoint *quic.Endpoint
	Services map[string]*model.Service
	Conns    *sync.Map
}

func (n *nodeServer) Run() error {
	go func() {
		for {
			accept, err := n.Endpoint.Accept(n.Ctx)
			if err != nil {
				zlog.Error(err.Error())
				return
			}

			go n.handleData(accept)
		}
	}()

	go func() {
		for {
			time.Sleep(time.Second * 30)
			var count int
			n.Conns.Range(func(key, value any) bool {
				count++
				return true
			})
			if count == 0 {
				n.Close()
			}
		}
	}()

	return nil
}

func (n *nodeServer) Close() error {
	return n.Endpoint.Close(n.Ctx)
}

func (n *nodeServer) handleData(conn *quic.Conn) {
	stream, err := conn.AcceptStream(n.Ctx)
	if err != nil {
		zlog.Error(err.Error())
		return
	}
	defer stream.Close()

	var buf [1500]byte
	read, err := stream.Read(buf[:])
	if err != nil {
		zlog.Error(err.Error())
		return
	}

	_, err = stream.Write([]byte("ok"))
	if err != nil {
		zlog.Error(err.Error())
		return
	}
	stream.Flush()

	var hk model.Handshake
	err = json.Unmarshal(buf[:read], &hk)
	if err != nil {
		zlog.Error(err.Error())
		return
	}
	s := n.Services[hk.ID]

	instance := config.Ctx.Value("instance").(*core.Instance)
	switch hk.Network {
	case config.NetworkTCP:
		listener, err := net.GetTcpListener(s.RemotePort)
		if err != nil {
			zlog.Error(err.Error())
			return
		}
		ctx, cancelFunc := context.WithCancel(n.Ctx)

		server := stcp.Server{
			ID:       hk.ID,
			Ctx:      ctx,
			Cancel:   cancelFunc,
			Conn:     conn,
			Listener: listener,
			EventCh:  make(chan string, 10),
			Lock:     &sync.Mutex{},
			Conns:    n.Conns,
		}
		err = instance.AddTask(&server)
		if err != nil {
			zlog.Error(err.Error())
			return
		}
		break
	case config.NetworkUDP:
		listener, err := net.GetUdpListener(s.RemotePort)
		if err != nil {
			zlog.Error(err.Error())
			return
		}
		ctx, cancelFunc := context.WithCancel(n.Ctx)

		server := sudp.Server{
			ID:       hk.ID,
			Ctx:      ctx,
			Cancel:   cancelFunc,
			Listener: listener,
			Conn:     conn,
			EventCh:  make(chan string, 10),
			Lock:     &sync.Mutex{},
			Conns:    n.Conns,
		}
		err = instance.AddTask(&server)
		if err != nil {
			zlog.Error(err.Error())
			return
		}
		break
	}

	n.Conns.Store(hk.ID, conn)
}
