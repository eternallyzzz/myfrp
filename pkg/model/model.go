package model

type Config struct {
	Log     *Log        `json:"log"`
	Control *Control    `json:"control"`
	Proxy   *LocalProxy `json:"proxy"`
}

type Log struct {
	ConsoleLevel string `json:"consoleLevel"`
	FileLevel    string `json:"fileLevel"`
	LogFilePath  string `json:"logFilePath"`
}

type Control struct {
	Listen *NetAddr `json:"listen"`
	Conn   *NetAddr `json:"conn"`
}

type LocalProxy struct {
	Type     string     `json:"type"`
	Services []*Service `json:"services"`
}

type NetAddr struct {
	Address string `json:"address"`
	Port    int    `json:"port"`
}

type Service struct {
	Listen   string `json:"listen"`
	Port     int    `json:"port"`
	Protocol string `json:"protocol"`
}

type RemoteProxy struct {
	Tag            string   `json:"tag"`
	RemoteService  *Service `json:"remoteService"`
	RemoteTransfer *NetAddr `json:"remoteTransfer"`
}

type RProxy struct {
	Local         *NetAddr
	Remote        *NetAddr
	Transfer      *NetAddr
	ProxyProtocol string
}
