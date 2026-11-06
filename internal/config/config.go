package config

import (
	"os"

	"github.com/BurntSushi/toml"
)

type Config struct {
	Listen string     `toml:"listen"`
	Db     Database   `toml:"database"`
	LogLv  string     `toml:"loglv"`
	GRPC   GRPCConfig `toml:"grpc"`
}

type Database struct {
	Db       string `toml:"db"`
	User     string `toml:"user"`
	Pwd      string `toml:"password"`
	Endpoint string `toml:"endpoint"`
}

// GRPCConfig 启用时必须配置证书与私钥；不提供明文降级路径。
type GRPCConfig struct {
	Listen   string `toml:"listen"`
	CertFile string `toml:"cert_file"`
	KeyFile  string `toml:"key_file"`
}

func Load(path string) (*Config, error) {
	cfg := &Config{}
	_, err := toml.DecodeFile(path, cfg)
	if err != nil {
		return nil, err
	}
	if cfg.LogLv == "" {
		cfg.LogLv = "info"
	}
	return cfg, nil
}

func (cfg *Config) Dump(path string) error {
	if path == "" {
		path = "./conf.dump.toml"
	}
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return toml.NewEncoder(f).Encode(cfg)
}
