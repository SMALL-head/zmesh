package main

import (
	"github.com/SMALL-head/zmesh/dataplane/iptables"
)

const (
	MY_NAT_CHAIN        = "MY_NAT_CHAIN"
	MY_PREROUTING_CHAIN = "MY_PREROUTING_CHAIN"
)

func SimpleNat(m iptables.Manager) {
	ok, err := m.Ipt.ChainExists("nat", MY_NAT_CHAIN)
	if err != nil {
		panic(err)
	}
	if !ok {
		err = m.Ipt.NewChain("nat", MY_NAT_CHAIN)
		if err != nil {
			panic(err)
		}
	}

	if err = m.Ipt.AppendUnique("nat", "OUTPUT", "-p", "tcp", "-j", MY_NAT_CHAIN); err != nil {
		panic(err)
	}
	if err := m.Ipt.AppendUnique("nat", MY_NAT_CHAIN,
		"-p", "tcp",
		"-d", "127.0.0.1",
		"--dport", "8080",
		"-j", "DNAT", "--to-destination", "127.0.0.0:8090",
	); err != nil {
		panic(err)
	}

}

func SimpleNatClean(m iptables.Manager) {
	ok, err := m.Ipt.ChainExists("nat", MY_NAT_CHAIN)
	if err != nil {
		panic(err)
	}
	if ok {
		m.Ipt.ClearChain("nat", MY_NAT_CHAIN)
		m.Ipt.DeleteIfExists("nat", "OUTPUT", "-p", "tcp", "-j", MY_NAT_CHAIN)
		m.Ipt.DeleteChain("nat", MY_NAT_CHAIN)
	}
}

// NatInbound 添加一条prerouting转发规则，我想知道对端流量resp是否会命中这条规则，所以创建了这个测试场景
// 测试方法：本机作为客户端去请求另一个服务器的7777端口，如果传输数据没有问题，就说明resp没有被这条规则重定向
func NatInbound(m iptables.Manager) {
	ok, err := m.Ipt.ChainExists("nat", MY_PREROUTING_CHAIN)
	if err != nil {
		panic(err)
	}
	if !ok {
		err = m.Ipt.NewChain("nat", MY_PREROUTING_CHAIN)
		if err != nil {
			panic(err)
		}
	}

	if err = m.Ipt.AppendUnique("nat", "PREROUTING", "-p", "tcp", "-j", MY_PREROUTING_CHAIN); err != nil {
		panic(err)
	}
	if err := m.Ipt.AppendUnique("nat", MY_PREROUTING_CHAIN,
		"-p", "tcp",
		"--sport", "7777",
		"-j", "REDIRECT",
		"--to-ports", "8090",
	); err != nil {
		panic(err)
	}
}

func NatInboundClean(m iptables.Manager) {
	ok, err := m.Ipt.ChainExists("nat", MY_PREROUTING_CHAIN)
	if err != nil {
		panic(err)
	}
	if ok {
		m.Ipt.ClearChain("nat", MY_PREROUTING_CHAIN)
		m.Ipt.DeleteIfExists("nat", "PREROUTING", "-p", "tcp", "-j", MY_PREROUTING_CHAIN)
		m.Ipt.DeleteChain("nat", MY_PREROUTING_CHAIN)
	}
}
