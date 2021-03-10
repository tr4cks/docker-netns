package cmd

import (
	"fmt"
	"github.com/kardianos/service"
	"github.com/spf13/cobra"
	"os"
	"strings"
)

func init() {
	serviceCmd.Flags().StringP("action", "a", "", strings.Join(service.ControlAction[:], "|"))
	serviceCmd.MarkFlagRequired("action")
	rootCmd.AddCommand(serviceCmd)
}

var serviceCmd = &cobra.Command{
	Use:   "service",
	Short: "Service management",
	Run: func(cmd *cobra.Command, args []string) {
		prg := &program{}
		s, err := service.New(prg, svcConfig)
		if err != nil {
			fmt.Fprintf(os.Stderr, "[service.New] %v\n", err)
			os.Exit(1)
		}
		err = service.Control(s, cmd.Flag("action").Value.String())
		if err != nil {
			fmt.Fprintf(os.Stderr, "[service.Control] %v\n", err)
			os.Exit(1)
		}
	},
}
