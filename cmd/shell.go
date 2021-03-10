package cmd

import (
	"context"
	"fmt"
	"github.com/docker/docker/client"
	"github.com/spf13/cobra"
	"github.com/vishvananda/netns"
	"os"
	"os/exec"
	"runtime"
)

func init() {
	shellCmd.Flags().StringP("container", "c", "", "container ID")
	shellCmd.MarkFlagRequired("container")
	rootCmd.AddCommand(shellCmd)
}

var shellCmd = &cobra.Command{
	Use:   "shell",
	Short: "Start a shell in a specific network namespace",
	Run: func(cmd *cobra.Command, args []string) {
		runtime.LockOSThread()
		defer runtime.UnlockOSThread()
		originNamespace, err := netns.Get()
		if err != nil {
			fmt.Fprintf(os.Stderr, "[netns.Get] %v\n", err)
			os.Exit(1)
		}
		defer originNamespace.Close()
		cli, err := client.NewClientWithOpts(client.WithVersion("1.40"))
		if err != nil {
			fmt.Fprintf(os.Stderr, "[client.NewClientWithOpts] %v\n", err)
			os.Exit(1)
		}
		container, err := cli.ContainerInspect(context.Background(), cmd.Flag("container").Value.String())
		if err != nil {
			fmt.Fprintf(os.Stderr, "[cli.ContainerInspect] %v\n", err)
			os.Exit(1)
		}
		println(container.State.Pid)
		ContainerNamespace, err := netns.GetFromPid(container.State.Pid)
		if err != nil {
			fmt.Fprintf(os.Stderr, "[netns.GetFromPid] %v\n", err)
			os.Exit(1)
		}
		defer ContainerNamespace.Close()
		err = netns.Set(ContainerNamespace)
		if err != nil {
			fmt.Fprintf(os.Stderr, "[netns.Set] %v\n", err)
			os.Exit(1)
		}
		shell := exec.Command(os.Getenv("SHELL"))
		shell.Stdin = os.Stdin
		shell.Stdout = os.Stdout
		shell.Stderr = os.Stderr
		err = shell.Run()
		if err != nil {
			fmt.Fprintf(os.Stderr, "[shell.Run] %v\n", err)
			os.Exit(1)
		}
		err = netns.Set(originNamespace)
		if err != nil {
			fmt.Fprintf(os.Stderr, "[netns.Set] %v\n", err)
			os.Exit(1)
		}
	},
}
