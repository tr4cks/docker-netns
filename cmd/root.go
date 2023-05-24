package cmd

import (
	"bytes"
	"context"
	yamlConfig "docker-netns/config"
	"fmt"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/client"
	"github.com/kardianos/service"
	"github.com/kballard/go-shellquote"
	"github.com/spf13/cobra"
	"github.com/vishvananda/netns"
	"log"
	"os"
	"os/exec"
	"path"
	"runtime"
	"syscall"
)

func init() {
	rootCmd.Flags().StringVar(&configPath, "config", path.Join("/opt", appName, "config.yaml"), "YAML configuration file")
}

const appName = "docker-netns"

var (
	configPath string
	logger     service.Logger
	rootCmd    = &cobra.Command{
		Use:     appName,
		Short:   "Docker network namespace manager",
		Version: "1.0.1",
		Run: func(cmd *cobra.Command, args []string) {
			prg := NewProgram()
			s, err := service.New(prg, svcConfig)
			prg.service = s
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
				os.Exit(1)
			}
		},
		CompletionOptions: cobra.CompletionOptions{
			DisableDefaultCmd: true,
		},
	}
	svcConfig = &service.Config{
		Name:        "docker-netns",
		DisplayName: "Docker network namespace service",
		UserName:    "root",
		Dependencies: []string{
			"After=network.target syslog.target docker.service",
		},
		Option: map[string]interface{}{
			"Restart": "on-failure",
		},
	}
)

func nsContext(containerID string, callback func() (interface{}, error)) (interface{}, error) {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	originNamespace, err := netns.Get()
	if err != nil {
		return nil, err
	}
	defer originNamespace.Close()
	cli, err := client.NewClientWithOpts(client.WithVersion("1.40"))
	if err != nil {
		return nil, err
	}
	container, err := cli.ContainerInspect(context.Background(), containerID)
	if err != nil {
		return nil, err
	}
	ContainerNamespace, err := netns.GetFromPid(container.State.Pid)
	if err != nil {
		return nil, err
	}
	defer ContainerNamespace.Close()
	err = netns.Set(ContainerNamespace)
	if err != nil {
		return nil, err
	}
	res, err := callback()
	if err != nil {
		return res, err
	}
	err = netns.Set(originNamespace)
	if err != nil {
		return nil, err
	}
	return res, nil
}

func execCommands(containerID string, commands []string) error {
	_, err := nsContext(containerID, func() (interface{}, error) {
		for _, command := range commands {
			command, err := shellquote.Split(command)
			if err != nil {
				return nil, err
			}
			execCmd := exec.Command(command[0], command[1:]...)
			var stderr bytes.Buffer
			execCmd.Stderr = &stderr
			err = execCmd.Run()
			if err != nil {
				return nil, fmt.Errorf("%v: %v", err, stderr.String())
			}
		}
		return nil, nil
	})
	return err
}

type program struct {
	service service.Service
	ctx     context.Context
	cancel  context.CancelFunc
	exited  chan error
}

func NewProgram() *program {
	ctx, cancel := context.WithCancel(context.Background())
	exited := make(chan error)
	return &program{ctx: ctx, cancel: cancel, exited: exited}
}

func (p *program) Start(s service.Service) error {
	// Start should not block. Do the actual work async.
	config, err := yamlConfig.NewConfig(configPath)
	if err != nil {
		return err
	}
	cli, err := client.NewClientWithOpts(client.WithVersion("1.40"))
	if err != nil {
		return err
	}
	go p.run(config, cli)
	return nil
}

func (p *program) run(config *yamlConfig.Config, cli *client.Client) {
	defer close(p.exited)
	process, _ := os.FindProcess(os.Getpid())
	containers, err := cli.ContainerList(context.Background(), types.ContainerListOptions{})
	if err != nil {
		process.Signal(syscall.SIGTERM)
		p.exited <- err
		return
	}
	for containerID := range *config {
		for _, container := range containers {
			if containerID == container.ID[:len(containerID)] {
				err := execCommands(containerID, (*config)[containerID])
				if err != nil {
					process.Signal(syscall.SIGTERM)
					p.exited <- err
					return
				}
				break
			}
		}
	}

	f := filters.NewArgs()
	f.Add("event", "start")
	msgs, errs := cli.Events(p.ctx, types.EventsOptions{Filters: f})
	for {
		select {
		case <-p.ctx.Done():
			return
		case <-msgs:
			for containerID := range *config {
				err := execCommands(containerID, (*config)[containerID])
				if err != nil {
					process.Signal(syscall.SIGTERM)
					p.exited <- err
					return
				}
			}
		case err := <-errs:
			process.Signal(syscall.SIGTERM)
			p.exited <- err
			return
		}
	}
}

func (p *program) Stop(s service.Service) error {
	// Stop should not block. Return with a few seconds.
	p.cancel()
	return <-p.exited
}

func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		log.Fatal(err)
	}
}
