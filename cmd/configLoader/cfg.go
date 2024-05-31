package configLoader

import (
	"encoding/json"
	_ "endpoint/core/control"
	_ "endpoint/core/reverse"
	"endpoint/pkg/config"
	"endpoint/pkg/model"
	"endpoint/pkg/zlog"
	"github.com/spf13/viper"
	"os"
	"strings"
)

func Init(path string) (*model.Config, error) {
	if c, err := loadConfig(path); err != nil {
		return nil, err
	} else {
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
