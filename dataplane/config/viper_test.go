package config_test

import (
	"fmt"
	"os"
	"testing"

	"github.com/SMALL-head/zmesh/dataplane/config"
	"github.com/stretchr/testify/require"
)

func TestParseConfig(t *testing.T) {
	cfg, err := config.ParseConfig()
	pwd, err := os.Getwd()
	require.NoError(t, err)
	fmt.Println("Current working directory:", pwd)
	require.NoError(t, err)
	fmt.Println(cfg.InBoundConfig.Host)
	fmt.Println(cfg.InBoundConfig.Mode)
	fmt.Println(cfg.OutBoundConfig.Mode)
}
