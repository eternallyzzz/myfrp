package model

import (
	"fmt"
	"time"
)

type Config struct {
	Log    *Log      `json:"log"`
	Quic   *Transfer `json:"transfer"`
	Role   []string  `json:"role"`
	Server *Server   `json:"server"`
	Client *Client   `json:"client"`
}

type Client struct {
	Conn  *NetAddr `json:"conn"`
	Proxy *Proxy   `json:"proxy"`
}

type Server struct {
	RandomPort string   `json:"randomPort"`
	Listen     *NetAddr `json:"listen"`
}

type Log struct {
	ConsoleLevel string `json:"consoleLevel"`
	FileLevel    string `json:"fileLevel"`
	LogFilePath  string `json:"logFilePath"`
}

type Transfer struct {
	TLS *Tls `json:"tls"`
	*QUICCfg
}

type QUICCfg struct {
	MaxBidiRemoteStreams     int64         `json:"maxBidiRemoteStreams"`
	MaxUniRemoteStreams      int64         `json:"maxUniRemoteStreams"`
	MaxStreamReadBufferSize  int64         `json:"maxStreamReadBufferSize"`
	MaxStreamWriteBufferSize int64         `json:"maxStreamWriteBufferSize"`
	MaxConnReadBufferSize    int64         `json:"maxConnReadBufferSize"`
	RequireAddressValidation bool          `json:"requireAddressValidation"`
	HandshakeTimeout         time.Duration `json:"handshakeTimeout"`
	MaxIdleTimeout           time.Duration `json:"maxIdleTimeout"`
	KeepAlivePeriod          time.Duration `json:"keepAlivePeriod"`
}

type Tls struct {
	Crt string `json:"crt"`
	Key string `json:"key"`
}

type Proxy struct {
	Services []*Service `json:"services"`
}

type NetAddr struct {
	Address string `json:"address"`
	Port    int    `json:"port"`
}

func (n *NetAddr) String() string {
	return fmt.Sprintf("%s:%d", n.Address, n.Port)
}

type Service struct {
	Tag      string `json:"tag"`
	Listen   string `json:"listen"`
	Port     int    `json:"port"`
	Protocol string `json:"protocol"`
}

func (s *Service) String() string {
	return fmt.Sprintf("%s:%d", s.Listen, s.Port)
}

type RemoteProxy struct {
	RemoteServices []*RemoteService `json:"remoteServices"`
	Transfer       *NetAddr         `json:"transfer"`
}

type RemoteService struct {
	Tag    string   `json:"tag"`
	Listen *Service `json:"listen"`
}

type ProxyConfig struct {
	Local    *Service
	Remote   *RemoteService
	Transfer *NetAddr `json:"transfer"`
}

type Handshake struct {
	Tag     string
	Network string
}
