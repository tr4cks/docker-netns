package cmd

import (
	"fmt"
	"github.com/spf13/cobra"
	"os"
	"os/exec"
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
		_, err := nsContext(cmd.Flag("container").Value.String(), func() (interface{}, error) {
			shell := exec.Command(os.Getenv("SHELL"))
			shell.Stdin = os.Stdin
			shell.Stdout = os.Stdout
			shell.Stderr = os.Stderr
			err := shell.Run()
			if err != nil {
				return nil, err
			}
			return nil, nil
		})
		if err != nil {
			fmt.Fprintf(os.Stderr, "[shell.Run] %v\n", err)
			os.Exit(1)
		}
	},
}
