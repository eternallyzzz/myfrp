package model

import "fmt"

type Config struct {
	Log     *Log     `json:"log"`
	Control *Control `json:"control"`
}

type Log struct {
	ConsoleLevel string `json:"consoleLevel"`
	FileLevel    string `json:"fileLevel"`
	LogFilePath  string `json:"logFilePath"`
}

type Control struct {
	Role   []string       `json:"role"`
	Listen *ListenControl `json:"listen"`
	Conn   *ConnControl   `json:"conn"`
}

type ListenControl struct {
	*NetAddr
}

type ConnControl struct {
	*NetAddr
	Proxy *Proxy `json:"proxy"`
}

type Proxy struct {
	Type          string     `json:"type"`
	LocalServices []*Service `json:"LocalServices"`
}

func (l *Proxy) GenTag() {
	for _, service := range l.LocalServices {
		service.Tag = fmt.Sprintf("%s-[%s]%s:%d", l.Type, service.Protocol, service.Listen, service.Port)
	}
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
