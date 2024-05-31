package cmd

import (
	"context"
	"endpoint/cmd/configLoader"
	"endpoint/core"
	"endpoint/pkg/config"
	"errors"
	"github.com/spf13/cobra"
	"log"
	"os"
	"os/signal"
	"syscall"
)

var (
	cfgFilePath string
)

func init() {
	rootCmd.Flags().StringVarP(&cfgFilePath, "config", "c", "", "path for config file")
}

var rootCmd = &cobra.Command{
	RunE: func(cmd *cobra.Command, args []string) error {
		return startEndpoint()
	},
}

func Run() {
	if err := rootCmd.Execute(); err != nil {
		log.Println(err)
		os.Exit(1)
	}
}

func startEndpoint() error {
	c, err := configLoader.Init(cfgFilePath)
	if err != nil {
		return err
	}

	instance, err := core.New(c)
	if err != nil {
		return errors.New("Failed to start:" + err.Error())
	}
	config.Ctx = context.WithValue(context.Background(), "instance", instance)

	if err = instance.Start(); err != nil {
		return errors.New("Failed to start:" + err.Error())
	}
	defer instance.Close()

	{
		osSignals := make(chan os.Signal, 1)
		signal.Notify(osSignals, os.Interrupt, syscall.SIGTERM)
		<-osSignals
	}
	return nil
}
