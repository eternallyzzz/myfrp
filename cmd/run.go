package cmd

import (
	"endpoint/cmd/configLoader"
	"endpoint/core"
	"errors"
	"github.com/spf13/cobra"
	"log"
	"os"
	"os/signal"
	"runtime"
	"runtime/debug"
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
		return startService()
	},
}

func Run() {
	if err := rootCmd.Execute(); err != nil {
		log.Println(err)
		os.Exit(1)
	}
}

func startService() error {
	c, err := configLoader.Init(cfgFilePath)
	if err != nil {
		return err
	}

	instance, err := core.New(c)
	if err != nil {
		return errors.New("Failed to start:" + err.Error())
	}

	if err = instance.Start(); err != nil {
		return errors.New("Failed to start:" + err.Error())
	}
	defer instance.Close()

	runtime.GC()
	debug.FreeOSMemory()

	{
		osSignals := make(chan os.Signal, 1)
		signal.Notify(osSignals, os.Interrupt, syscall.SIGTERM)
		<-osSignals
	}
	return nil
}
