package iptables

import (
	"github.com/coreos/go-iptables/iptables"
	"github.com/sirupsen/logrus"
)

type Manager struct {
	ipt       *iptables.IPTables
	chainName string
}

func New() (Manager, error) {
	tables, err := iptables.New()
	if err != nil {
		logrus.Fatalf("error creating iptables manager: %s", err)
		return Manager{}, nil
	}
	return Manager{
		ipt:       tables,
		chainName: "zmesh",
	}, nil
}

// SetupBasicRules 创建zmesh所需要的基本规则
func (m *Manager) SetupBasicRules() error {
	return nil
}

func (m *Manager) AddRule(ruleArgs ...string) error {
	return nil
}
