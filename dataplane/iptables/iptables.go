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
	PROXY_PACKET_MARK = "77"

	// Basic rules for zmesh
	basicRules = [][]string{
		// jump rules
		{"nat", "OUTPUT", "-p", "tcp", "-j", MESH_OUPUT_CHAIN},
		{"nat", "PREROUTING", "-p", "tcp", "-j", MESH_PREROUTING_CHAIN},
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
