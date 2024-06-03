package configLoader

import (
	"encoding/json"
	_ "endpoint/core/control"
	_ "endpoint/core/reverse"
	"endpoint/pkg/config"
	"endpoint/pkg/model"
	"endpoint/pkg/zlog"
	"fmt"
	"github.com/spf13/viper"
	"os"
	"strings"
	"time"
)

func Init(path string) (*model.Config, error) {
	if c, err := loadConfig(path); err != nil {
		return nil, err
	} else {
		if c.Quic != nil {
			config.MaxStreams = c.Quic.MaxIncomeStreams
			config.MaxIdle = time.Duration(c.Quic.MaxIdle) * time.Second
			config.KeepAlive = time.Duration(c.Quic.Keepalive) * time.Second
		}

		return c, zlog.Init(c.Log)
	}
}

func loadConfig(path string) (*model.Config, error) {
	if path == "" {
		wd, _ := os.Getwd()
		path = wd + config.CfgBase
		if file, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0777); err != nil {
			return nil, err
		} else {
			_ = file.Close()
		}
	}

	fmt.Println(time.Now().Format(time.RFC3339), "	INFO", "	Use config: "+path)

	viper.SetConfigFile(path)

	if err := viper.ReadInConfig(); err != nil {
		return nil, err
	}

	var cfg model.Config

	m, err := json.Marshal(viper.AllSettings())
	if err != nil {
		return nil, err
	}
	err = json.Unmarshal(m, &cfg)
	if err != nil {
		return nil, err
	}

	if &cfg != nil && cfg.Control != nil && cfg.Control.Conn != nil && cfg.Control.Conn.Proxy != nil && cfg.Control.Conn.Proxy.LocalServices != nil {
		for _, service := range cfg.Control.Conn.Proxy.LocalServices {
			service.Protocol = strings.ToLower(service.Protocol)
		}
	}

	return &cfg, nil
}
