package model

import (
	"fmt"
)

type Config struct {
	Log    *Log     `json:"log"`
	Quic   *QUIC    `json:"quic"`
	Role   []string `json:"role"`
	Server *Server  `json:"server"`
	Client *Client  `json:"client"`
}

type Client struct {
	Conn  *NetAddr `json:"conn"`
	Proxy *Proxy   `json:"proxy"`
}

type Server struct {
	Listen *NetAddr `json:"listen"`
}

type Log struct {
	ConsoleLevel string `json:"consoleLevel"`
	FileLevel    string `json:"fileLevel"`
	LogFilePath  string `json:"logFilePath"`
}

type QUIC struct {
	MaxIncomeStreams int64 `json:"maxIncomeStreams"`
	MaxIdle          int64 `json:"maxIdle"`
	Keepalive        int   `json:"keepalive"`
}

type Proxy struct {
	Type     string     `json:"type"`
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
	Type           string           `json:"type"`
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
