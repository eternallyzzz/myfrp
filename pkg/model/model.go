package model

import (
	"fmt"
	"time"
)

type Config struct {
	Log      *Log      `json:"log"`
	Transfer *Transfer `json:"transfer"`
	Endpoint *Endpoint `json:"endpoint"`
	Proxy    *Proxy    `json:"proxy"`
}

type Proxy struct {
	Remote []*RemoteInfo `json:"remote"`
	Local  []*Service    `json:"local"`
}

type RemoteInfo struct {
	Tag string `json:"tag"`
	*NetAddr
}

type Client struct {
	Conn  *NetAddr `json:"conn"`
	Proxy *Proxy   `json:"proxy"`
}

type Endpoint struct {
	RandPort string `json:"randPort"`
	*NetAddr
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

type NetAddr struct {
	Address string `json:"address"`
	Port    uint16 `json:"port"`
}

func (n *NetAddr) String() string {
	return fmt.Sprintf("%s:%d", n.Address, n.Port)
}

type Service struct {
	ID         string `json:"id"`
	Address    string `json:"address"`
	Port       uint16 `json:"port"`
	Protocol   string `json:"protocol"`
	RemoteTag  string `json:"remoteTag"`
	RemotePort uint16 `json:"remotePort"`
	NodePort   uint16 `json:"nodePort"`
}

func (s *Service) String() string {
	return fmt.Sprintf("%s:%d", s.Address, s.Port)
}

type Handshake struct {
	ID      string `json:"id"`
	Network string `json:"network"`
}
