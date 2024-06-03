package config

import (
	"context"
	"time"
)

var (
	Ctx         context.Context
	MaxStreams  int64 = 100
	MaxIdle           = time.Minute * 30
	KeepAlive         = time.Second * 10
	ContentType byte  = 0
	MsgType     byte  = 1
)
