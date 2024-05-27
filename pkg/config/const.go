package config

import "time"

const (
	Reverse             = "reverse"
	Forward             = "forward"
	NetworkQUIC         = "udp"
	MaxStreams          = 100
	MaxIdle             = time.Minute * 30
	KeepAlive           = time.Second * 10
	ServerTLS           = 0
	ClientTLS           = 1
	CfgBase             = "/config.yaml"
	DefaultConsoleLevel = "warn"
	DefaultFileLevel    = "error"
)
