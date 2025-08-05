package main

import (
	"github.com/SMALL-head/zmesh/dataplane/proxy"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

func main() {
	_ = newCobraCommand().Execute()
}

func newCobraCommand() *cobra.Command {
	command := &cobra.Command{
		Use: "zmesh dataplane",
		Run: run,
	}

	return command
}

func run(cmd *cobra.Command, args []string) {
	// 启动转发代理服务器
	p := proxy.New(proxy.WithHost(""), proxy.WithPort(8090), proxy.WithMode(proxy.ProxyMode))

	if err := p.Start(); err != nil {
		logrus.Errorf("error starting proxy: %s", err)
	}

}
