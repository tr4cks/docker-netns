package cmd

import (
	"context"
	"fmt"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/client"
	"github.com/kardianos/service"
	"github.com/spf13/cobra"
	"log"
	"os"
)

var (
	logger service.Logger
	rootCmd = &cobra.Command{
		Use:   "docker-netns",
		Short: "Docker network namespace manager",
		Version: "1.0.0",
		Run: func(cmd *cobra.Command, args []string) {
			prg := &program{}
			s, err := service.New(prg, svcConfig)
			if err != nil {
				fmt.Fprintf(os.Stderr, "[service.New] %v\n", err)
				os.Exit(1)
			}
			logger, err = s.Logger(nil)
			if err != nil {
				fmt.Fprintf(os.Stderr, "[s.Logger] %v\n", err)
				os.Exit(1)
			}
			err = s.Run()
			if err != nil {
				logger.Error(err)
			}
		},
	}
	svcConfig = &service.Config{
		Name: "docker-netns",
		DisplayName: "Docker network namespace service",
		UserName: "root",
		Dependencies: []string{
			"After=network.target syslog.target docker.service",
		},
		Option: map[string]interface{}{
			"Restart": "on-failure",
		},
	}
)

type program struct {}

func (p *program) Start(s service.Service) error {
	// Start should not block. Do the actual work async.
	cli, err := client.NewClientWithOpts(client.WithVersion("1.40"))
	if err != nil {
		return err
	}
	go p.run(cli)
	return nil
}

func (p *program) run(cli *client.Client) {
	ctx := context.Background()
	f := filters.NewArgs()
	f.Add("event", "start")
	msgs, errs := cli.Events(ctx, types.EventsOptions{Filters: f})
	for {
		select {
		case msg := <-msgs:
			_ = msg
		case err := <-errs:
			_ = err
			return
		}
	}
}

func (p *program) Stop(s service.Service) error {
	// Stop should not block. Return with a few seconds.
	return nil
}

func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		log.Fatal(err)
	}
}
