package id

import (
	"endpoint/pkg/zlog"
	"github.com/bwmarrin/snowflake"
	"math/rand"
	"sync"
)

var (
	once sync.Once
	node *snowflake.Node
)

func GetSnowflakeID() snowflake.ID {
	once.Do(func() {
		nod, err := snowflake.NewNode(rand.Int63n(1023))
		if err != nil {
			zlog.Error(err.Error())
			return
		}
		node = nod
	})
	return node.Generate()
}
