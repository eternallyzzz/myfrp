package config

import (
	"context"
	"endpoint/pkg/model"
	"time"
)

var (
	Ctx         context.Context
	MaxStreams  int64 = 100
	MaxIdle           = time.Minute * 30
	KeepAlive         = time.Second * 20
	ContentType byte  = 0
	MsgType     byte  = 1
	QUICCfg     *model.Transfer
)
