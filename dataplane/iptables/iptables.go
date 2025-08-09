package iptables

import (
	"github.com/coreos/go-iptables/iptables"
	"github.com/sirupsen/logrus"
)

var (
	// Chain names

	MESH_OUPUT_CHAIN      = "ZMESH_OUTPUT"
	MESH_PREROUTING_CHAIN = "ZMESH_PREROUTING"

	// MARK
	PROXY_PACKET_MARK       = "77"
	OUTBOUND_CONNTRACK_MARK = "0x43"

	// Basic rules for zmesh
	basicRules = [][]string{
		// jump rules
		{"nat", "OUTPUT", "-p", "tcp", "-j", MESH_OUPUT_CHAIN},
		{"nat", "PREROUTING", "-p", "tcp", "-j", MESH_PREROUTING_CHAIN},
		{"mangle", "PREROUTING", "-p", "tcp", "-j", MESH_PREROUTING_CHAIN},
		{"mangle", "OUTPUT", "-p", "tcp", "-j", MESH_OUPUT_CHAIN},
	}
)

type Manager struct {
	Ipt       *iptables.IPTables
	ChainName string
	PodCIDR   string
}

func New(chainName string) (Manager, error) {
	tables, err := iptables.New()
	if err != nil {
		logrus.Fatalf("error creating iptables manager: %s", err)
		return Manager{}, nil
	}
	return Manager{
		Ipt:       tables,
		ChainName: chainName,
		PodCIDR:   "10.10.0.0/16",
	}, nil
}

// SetupBasicRules 创建zmesh所需要的基本规则
func (m *Manager) SetupBasicRules() error {
	// 创建必要的链条
	if err := m.Ipt.NewChain("nat", MESH_OUPUT_CHAIN); err != nil {
		logrus.Errorf("[SetupBasicRules] error creating MESH_OUTPUT_CHAIN: %s", err)
	}
	if err := m.Ipt.NewChain("nat", MESH_PREROUTING_CHAIN); err != nil {
		logrus.Errorf("[SetupBasicRules] error creating MESH_PREROUTING_CHAIN: %s", err)
	}
	if err := m.Ipt.NewChain("mangle", MESH_OUPUT_CHAIN); err != nil {
		logrus.Errorf("[SetupBasicRules] error creating MESH_OUPUT_CHAIN in mangle table: %s", err)
	}
	if err := m.Ipt.NewChain("mangle", MESH_PREROUTING_CHAIN); err != nil {
		logrus.Errorf("[SetupBasicRules] error creating MESH_PREROUTING_CHAIN in mangle table: %s", err)
	}

	// 基础跳转规则
	for _, rule := range basicRules {
		if err := m.Ipt.AppendUnique(rule[0], rule[1], rule[2:]...); err != nil {
			return err
		}
	}
	return nil
}

func (m *Manager) ClearBasicRules() {
	for _, rule := range basicRules {
		err := m.Ipt.Delete(rule[0], rule[1], rule[2:]...)
		if err != nil {
			logrus.Errorf("[ClearBasicRules] error deleting rule %v: %s", rule, err)
		}
	}
}
