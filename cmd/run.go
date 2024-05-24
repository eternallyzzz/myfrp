package cmd

import (
	"endpoint/cmd/configLoader"
	"github.com/spf13/cobra"
	"log"
	"os"
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
	if err := configLoader.Init(cfgFilePath); err != nil {
		return err
	}

}
