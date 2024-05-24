package configLoader

import (
	"endpoint/pkg/zlog"
	"github.com/spf13/viper"
	"os"
)

func Init(path string) error {
	if err := loadConfig(path); err != nil {
		return nil
	}
	return zlog.Init()
}

func loadConfig(path string) error {
	if path == "" {
		path, _ = os.Getwd()
	}

	viper.SetConfigFile(path)

	if err := viper.ReadInConfig(); err != nil {
		return err
	}
	return nil
}
