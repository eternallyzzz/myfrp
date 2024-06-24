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
	"runtime"
	"runtime/debug"
	"syscall"
)

const (
	cfgFilePath = "config"
)

func init() {
	rootCmd.Flags().StringP(cfgFilePath, "c", "config.yaml", "config file path")
}

var rootCmd = &cobra.Command{
	RunE: func(cmd *cobra.Command, args []string) error {
		cPath, err := cmd.Flags().GetString(cfgFilePath)
		if err != nil {
			return err
		}
		return startEndpoint(cPath)
	},
}

func Run() {
	if err := rootCmd.Execute(); err != nil {
		log.Println(err)
		os.Exit(1)
	}
}

func startEndpoint(cPath string) error {
	c, err := configLoader.Init(cPath)
	if err != nil {
		return err
	}

	instance, err := core.New(c)
	if err != nil {
		return errors.New("Failed to start: " + err.Error())
	}
	config.Ctx = context.WithValue(context.Background(), "instance", instance)

	if err = instance.Start(); err != nil {
		return errors.New("Failed to start: " + err.Error())
	}
	defer func(instance *core.Instance) {
		err := instance.Close()
		if err != nil {
			return
		}
	}(instance)

	runtime.GC()
	debug.FreeOSMemory()

	{
		osSignals := make(chan os.Signal, 1)
		signal.Notify(osSignals, os.Interrupt, syscall.SIGTERM)
		<-osSignals
	}
	return nil
}
