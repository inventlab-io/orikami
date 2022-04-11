package config

import (
	"fmt"
	"github.com/spf13/viper"
	"path/filepath"
	"strings"
)

type ServerConfig struct {
	RequestTimeout int

	Etcd struct {
		Endpoints         []string
		ConnectionTimeout int
	}
}

func LoadServerConfig(path string) (config ServerConfig, err error) {
	agentV := viper.New()

	agentV.SetDefault("RequestTimeout", 2)
	agentV.SetDefault("Etcd.Endpoints", []string{"127.0.0.1:2379"})
	agentV.SetDefault("Etcd.ConnectionTimeout", 2)

	agentV.SetConfigType("yaml")

	if path != "" {

		dir := filepath.Dir(path)
		if dir != "" {
			agentV.AddConfigPath(dir)
		}

		fn := strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
		if fn == "" {
			fn = "ignition"
		}
		agentV.SetConfigName(fn)

		err = agentV.ReadInConfig()
		if err != nil {
			panic(fmt.Errorf("Fatal error config file: %w \n", err))
		}
	}

	err = agentV.Unmarshal(&config)

	if err != nil {
		panic(fmt.Errorf("Fatal error config file: %w \n", err))
	}
	return
}