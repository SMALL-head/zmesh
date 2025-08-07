package config

type BootStrapConfig struct {
	InBoundConfig  ServerConfig `yaml:"inbound"`
	OutBoundConfig ServerConfig `yaml:"outbound"`
}

type ServerConfig struct {
	Host string `yaml:"host"`
	Port int    `yaml:"port"`
	Mode string `yaml:"mode"` // 代理模式，sidecar或proxy
}

func DefaultBootStrapConfig() BootStrapConfig {
	return BootStrapConfig{
		InBoundConfig: ServerConfig{
			Host: "0.0.0.0",
			Port: 8091,
			Mode: "sidecar",
		},
		OutBoundConfig: ServerConfig{
			Host: "0.0.0.0",
			Port: 8090,
			Mode: "sidecar",
		},
	}
}
