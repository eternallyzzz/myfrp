package config

import "time"

const (
	Reverse                   = "reverse"
	Forward                   = "forward"
	NetworkQUIC               = "udp"
	MaxStreams                = 100
	MaxIdle                   = time.Minute * 30
	KeepAlive                 = time.Second * 30
	ServerTLS                 = 0
	ClientTLS                 = 1
	CfgBase                   = "/config.yaml"
	DefaultConsoleLevel       = "warn"
	DefaultFileLevel          = "error"
	RoleSrv                   = "server"
	RoleCli                   = "client"
	BadService                = "bad service"
	NetworkUDP                = "udp"
	NetworkTCP                = "tcp"
	HeartbeatType       int64 = 0
	EventType           int64 = 1
	ContentType         int64 = 2
)
