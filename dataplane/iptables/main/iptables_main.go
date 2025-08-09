package main

import (
	"github.com/SMALL-head/zmesh/dataplane/iptables"
	"github.com/sirupsen/logrus"
)

func main() {
	m, err := iptables.New("zmesh")
	if err != nil {
		logrus.Fatal("error creating iptables manager: ", err)
	}

	m.SetupBasicRules()
	SceneOutBound(m)
	SceneInbound(m)

	// SceneOutBoundClean(m)
	// SceneInboundClean(m)
	// m.ClearBasicRules()
}

func SceneOutBound(m iptables.Manager) {
	// 注：数据包流出方向，首先经过四表的OUTPUT链。路由选择后走POSTROUTING链。
	err := m.Ipt.NewChain("nat", iptables.MESH_OUPUT_CHAIN)
	if err != nil {
		logrus.Errorf("[Scene1] error creating MESH_OUTPUT_CHAIN: %s", err)
		return
	}

	// err = m.SetupBasicRules()
	// if err != nil {
	// 	logrus.Fatal("error setting up basic rules: ", err)
	// }
	err = m.Ipt.AppendUnique("nat", "OUTPUT", "-p", "tcp", "-j", iptables.MESH_OUPUT_CHAIN)
	if err != nil {
		logrus.Errorf("[Scene1] error appending rule to OUTPUT chain: %s", err)
		return
	}
	m.PodCIDR = "10.10.0.0/16"

	err = m.Ipt.AppendUnique(
		"nat", iptables.MESH_OUPUT_CHAIN,
		"-p", "tcp",
		"-m", "owner", "--uid-owner", "1337",
		"-j", "RETURN",
	)

	if err != nil {
		logrus.Errorf("[Scene1] error appending rule1 to MESH_OUTPUT_CHAIN: %s", err)
		return
	}

	err = m.Ipt.AppendUnique(
		"nat", iptables.MESH_OUPUT_CHAIN,
		"-p", "tcp",
		"-d", m.PodCIDR,
		"!", "--sport", "8090", // 从proxy返回给src的流量不应该被重定向
		"-j", "REDIRECT",
		"--to-ports", "8090") // 转发流量至proxy

	if err != nil {
		logrus.Errorf("[Scene1] error appending rule to MESH_OUTPUT_CHAIN: %s", err)
		return
	}

	// output方，对于新建立的连接，打上某个特定的标记
	err = m.Ipt.AppendUnique(
		"mangle", iptables.MESH_OUPUT_CHAIN,
		"-p", "tcp",
		"-d", m.PodCIDR,
		"-m", "conntrack", "--ctstate", "NEW",
		"-j", "CONNMARK", "--set-mark", iptables.OUTBOUND_CONNTRACK_MARK,
	)
	if err != nil {
		logrus.Errorf("[Scene1] error appending rule to mangle MESH_OUTPUT_CHAIN: %s", err)
		return
	}

	// 收到流量后，恢复这个标记，然后后续inbound规则对于拥有该标记的流不进行redirect处理
	// 虽然这里是处理的事inbound流量，但是我仍然把它写在SceneOutBound里，
	// 因为它是和outbound的conntrack标记配合使用的。或者说这个规则被触发的前提是outbound发出流量
	err = m.Ipt.AppendUnique(
		"mangle", iptables.MESH_PREROUTING_CHAIN,
		"-j", "CONNMARK", "--restore-mark",
	)
	if err != nil {
		logrus.Errorf("[Scene1] error appending rule to mangle MESH_PREROUTING_CHAIN: %s", err)
		return
	}

}

func SceneOutBoundClean(m iptables.Manager) {
	m.ClearBasicRules()
	ok, _ := m.Ipt.ChainExists("nat", iptables.MESH_OUPUT_CHAIN)
	if ok {
		err := m.Ipt.ClearChain("nat", iptables.MESH_OUPUT_CHAIN)
		if err != nil {
			logrus.Errorf("[Scene1Clean] error clearing MESH_OUTPUT_CHAIN: %s", err)
		}
		err = m.Ipt.DeleteChain("nat", iptables.MESH_OUPUT_CHAIN)
		if err != nil {
			logrus.Errorf("[Scene1Clean] error deleting MESH_OUTPUT_CHAIN: %s", err)
		}
	}
	ok, _ = m.Ipt.ChainExists("nat", iptables.MESH_PREROUTING_CHAIN)
	if ok {
		err := m.Ipt.ClearChain("nat", iptables.MESH_PREROUTING_CHAIN)
		if err != nil {
			logrus.Errorf("[Scene1Clean] error clearing MESH_PREROUTING_CHAIN: %s", err)
		}
		err = m.Ipt.DeleteChain("nat", iptables.MESH_PREROUTING_CHAIN)
		if err != nil {
			logrus.Errorf("[Scene1Clean] error deleting MESH_PREROUTING_CHAIN: %s", err)
		}
	}

	ok, _ = m.Ipt.ChainExists("mangle", "OUTPUT")
	if ok {
		err := m.Ipt.ClearChain("mangle", "OUTPUT")
		if err != nil {
			logrus.Errorf("[Scene1Clean] error clearing mangle OUTPUT chain: %s", err)
		}
		// mangle OUTPUT是自带的Chain，就不删除了
	}

}

// SceneInbound 这个场景是为了测试Mesh2Mesh的转发逻辑
func SceneInbound(m iptables.Manager) {
	err := m.Ipt.NewChain("nat", iptables.MESH_PREROUTING_CHAIN)
	if err != nil {
		logrus.Errorf("[SceneInbound] error creating MESH_PREROUTING_CHAIN: %s", err)
		return
	}
	err = m.Ipt.AppendUnique("nat", "PREROUTING", "-p", "tcp", "-j", iptables.MESH_PREROUTING_CHAIN)
	if err != nil {
		logrus.Errorf("[SceneInbound] error appending rule to PREROUTING chain: %s", err)
		return
	}

	err = m.Ipt.AppendUnique("nat", iptables.MESH_PREROUTING_CHAIN,
		"-p", "tcp",
		"-d", m.PodCIDR,
		"-m", "connmark", "!", "--mark", iptables.OUTBOUND_CONNTRACK_MARK,
		"-j", "REDIRECT",
		"--to-ports", "8092",
	)
	if err != nil {
		logrus.Errorf("[SceneInbound] error appending rule to MESH_PREROUTING_CHAIN: %s", err)
		return
	}

}

func SceneInboundClean(m iptables.Manager) {
	m.ClearBasicRules()
	ok, _ := m.Ipt.ChainExists("nat", iptables.MESH_PREROUTING_CHAIN)
	if ok {
		err := m.Ipt.ClearChain("nat", iptables.MESH_PREROUTING_CHAIN)
		if err != nil {
			logrus.Errorf("[SceneInboundClean] error clearing MESH_PREROUTING_CHAIN: %s", err)
		}
		err = m.Ipt.DeleteChain("nat", iptables.MESH_PREROUTING_CHAIN)
		if err != nil {
			logrus.Errorf("[SceneInboundClean] error deleting MESH_PREROUTING_CHAIN: %s", err)
		}
	}
}
