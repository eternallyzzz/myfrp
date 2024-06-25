package control

import (
	"context"
	"encoding/json"
	"endpoint/core/reverse"
	"endpoint/pkg/common"
	"endpoint/pkg/model"
	"endpoint/pkg/zlog"
	"golang.org/x/net/quic"
	"reflect"
	"time"
)

type endpointServer struct {
	Ctx            context.Context
	EndpointConfig *model.Endpoint
	Endpoint       *quic.Endpoint
}

func (e *endpointServer) Run() error {
	endpoint, err := common.GetEndpoint(e.EndpointConfig.NetAddr)
	if err != nil {
		return err
	}
	e.Endpoint = endpoint

	zlog.Warn("listening UDP on " + e.EndpointConfig.NetAddr.String())

	go acceptConn(e.Ctx, endpoint)

	return nil
}

func (e *endpointServer) Close() error {
	return e.Endpoint.Close(e.Ctx)
}

func acceptConn(ctx context.Context, endpoint *quic.Endpoint) {
	for {
		accept, err := endpoint.Accept(ctx)
		if err != nil {
			zlog.Error(err.Error())
			return
		}

		go handleConn(ctx, accept)
	}
}

func handleConn(ctx context.Context, conn *quic.Conn) {
	defer conn.Close()

	for {
		stream, err := conn.AcceptStream(ctx)
		if err != nil {
			zlog.Error(err.Error())
			return
		}

		var buf [1500]byte
		n, err := stream.Read(buf[:])
		if err != nil {
			zlog.Error(err.Error())
			return
		}

		var data []*model.Service
		err = json.Unmarshal(buf[:n], &data)
		if err != nil {
			stream.Close()
			zlog.Error(err.Error())
			return
		}

		err = reverse.DoReverseSrv(ctx, data)
		if err != nil {
			stream.Close()
			zlog.Error(err.Error())
			return
		}

		m, err := json.Marshal(&data)
		if err != nil {
			stream.Close()
			zlog.Error(err.Error())
			return
		}

		_, err = stream.Write(m)
		if err != nil {
			zlog.Error(err.Error())
			return
		}
		stream.Flush()

		time.Sleep(time.Second * 5)
		stream.Close()
		break
	}

}

func endpointServerCreator(ctx context.Context, v any) (any, error) {
	endpoint := v.(*model.Endpoint)
	return &endpointServer{Ctx: ctx, EndpointConfig: endpoint}, nil
}

func init() {
	ec := reflect.TypeOf(&model.Endpoint{})
	common.ServerContext[ec] = endpointServerCreator
}
