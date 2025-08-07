package config

import (
	"github.com/go-viper/mapstructure/v2"
	"github.com/spf13/viper"
)

func ParseConfig() (BootStrapConfig, error) {
	// 解析配置文件
	config := BootStrapConfig{}
	// 这里可以添加实际的配置解析逻辑
	v := viper.New()
	v.AddConfigPath(".")
	v.AddConfigPath("./config")
	v.SetConfigName("application")
	v.SetConfigType("yaml")
	if err := v.ReadInConfig(); err != nil {
		return config, err
	}
	if err := v.Unmarshal(&config, func(config *mapstructure.DecoderConfig) {
		config.TagName = "yaml"
	}); err != nil {
		return config, err
	}
	return config, nil
}
