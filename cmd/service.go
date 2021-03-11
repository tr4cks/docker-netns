package cmd

import (
	"fmt"
	"github.com/kardianos/service"
	"github.com/spf13/cobra"
	"io/ioutil"
	"os"
	"path"
)

func init() {
	installCmd.Flags().String("config", "./config.yaml", "YAML configuration file")
	serviceCmd.AddCommand(startCmd, stopCmd, restartCmd, installCmd, uninstallCmd)
	rootCmd.AddCommand(serviceCmd)
}

func serviceControl(action string) func(*cobra.Command, []string) {
	return func(cmd *cobra.Command, args []string) {
		prg := NewProgram()
		s, err := service.New(prg, svcConfig)
		prg.service = s
		if err != nil {
			fmt.Fprintf(os.Stderr, "[service.New] %v\n", err)
			os.Exit(1)
		}
		err = service.Control(s, action)
		if err != nil {
			fmt.Fprintf(os.Stderr, "[service.Control] %v\n", err)
			os.Exit(1)
		}
	}
}

var (
	serviceCmd = &cobra.Command{
		Use:   "service",
		Short: "Service management",
	}
	startCmd = &cobra.Command{
		Use:   "start",
		Short: "Start service",
		Run:   serviceControl("start"),
	}
	stopCmd = &cobra.Command{
		Use:   "stop",
		Short: "Stop service",
		Run:   serviceControl("stop"),
	}
	restartCmd = &cobra.Command{
		Use:   "restart",
		Short: "Restart service",
		Run:   serviceControl("restart"),
	}
	installCmd = &cobra.Command{
		Use:   "install",
		Short: "Install service",
		Run: func(cmd *cobra.Command, args []string) {
			// TODO: transaction
			err := os.Mkdir(path.Join("/opt", appName), 0755)
			if err != nil {
				fmt.Fprintf(os.Stderr, "[os.Mkdir] %v\n", err)
				os.Exit(1)
			}
			bytes, err := ioutil.ReadFile(cmd.Flag("config").Value.String())
			if err != nil {
				fmt.Fprintf(os.Stderr, "[ioutil.ReadFile] %v\n", err)
				os.Exit(1)
			}
			err = ioutil.WriteFile(path.Join("/opt", appName, "config.yaml"), bytes, 0644)
			if err != nil {
				fmt.Fprintf(os.Stderr, "[ioutil.WriteFile] %v\n", err)
				os.Exit(1)
			}
			serviceControl("install")(cmd, args)
		},
	}
	uninstallCmd = &cobra.Command{
		Use:   "uninstall",
		Short: "Uninstall service",
		Run: func(cmd *cobra.Command, args []string) {
			// TODO: transaction
			err := os.RemoveAll(path.Join("/opt", appName))
			if err != nil {
				fmt.Fprintf(os.Stderr, "[ioutil.RemoveAll] %v\n", err)
				os.Exit(1)
			}
			serviceControl("uninstall")(cmd, args)
		},
	}
)
