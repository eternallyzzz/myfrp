package reverse

import (
	"endpoint/pkg/config"
	"endpoint/pkg/model"
)

type Server struct {
	Cfg *model.RProxy
}

func (s *Server) Run() error {

}

func (s *Server) Close() error {

}

func (s *Server) Set(cfg any) any {
	s.Cfg = cfg.(*model.RProxy)
	return s
}

func init() {
	config.ServerContext[config.Reverse] = &Server{}
}
