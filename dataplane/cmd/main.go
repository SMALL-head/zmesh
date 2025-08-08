package main

import (
	"github.com/SMALL-head/zmesh/dataplane/config"
	"github.com/SMALL-head/zmesh/dataplane/proxy"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"golang.org/x/sync/errgroup"
)

func main() {
	_ = newCobraCommand().Execute()
}

func newCobraCommand() *cobra.Command {
	var configPath string

	command := &cobra.Command{
		Use: "zmesh dataplane",
		Run: func(cmd *cobra.Command, args []string) {
			run(cmd, args, configPath)
		},
	}

	command.Flags().StringVarP(&configPath, "config", "c", "", "指定配置文件路径")

	return command
}

func run(cmd *cobra.Command, args []string, configPath string) {
	eg := errgroup.Group{}
	vCfg, err := config.ParseConfig(configPath)
	if err != nil {
		logrus.Fatal("error parsing config: ", err)
	}
	var oMode, iMode proxy.Mode
	switch vCfg.OutBoundConfig.Mode {
	case "sidecar":
		oMode = proxy.SidecarMode
	case "proxy":
		oMode = proxy.ProxyMode
	default:
		logrus.Fatalf("invalid outbound mode: %s", vCfg.OutBoundConfig.Mode)
	}
	switch vCfg.InBoundConfig.Mode {
	case "sidecar":
		iMode = proxy.SidecarMode
	case "proxy":
		iMode = proxy.ProxyMode
	default:
		logrus.Fatalf("invalid inbound mode: %s", vCfg.InBoundConfig.Mode)
	}
	// 启动转发代理服务器
	po := proxy.NewProxyOutBound(
		proxy.WithHost(vCfg.OutBoundConfig.Host),
		proxy.WithPort(vCfg.OutBoundConfig.Port),
		proxy.WithMode(oMode),
	)
	pi := proxy.NewProxyInBound(
		proxy.WithHost(vCfg.InBoundConfig.Host),
		proxy.WithPort(vCfg.InBoundConfig.Port),
		proxy.WithMode(iMode),
	)
	eg.Go(func() error {
		if err := po.Start(); err != nil {
			return err
		}
		return nil
	})

	eg.Go(func() error {
		if err := pi.Start(); err != nil {
			return err
		}
		return nil
	})

	if err := eg.Wait(); err != nil {
		logrus.Fatal("error running proxy: ", err)
	}

}
