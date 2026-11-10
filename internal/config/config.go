package config

import (
	"fmt"
	"os"

	"github.com/BurntSushi/toml"
)

type Config struct {
	Listen    string     `toml:"listen"`
	Db        Database   `toml:"database"`
	LogLv     string     `toml:"loglv"`
	Directory string     `toml:"directory"`
	GRPC      GRPCConfig `toml:"grpc"`
}

type Database struct {
	Db       string `toml:"db"`
	User     string `toml:"user"`
	Pwd      string `toml:"password"`
	Endpoint string `toml:"endpoint"`
}

// GRPCConfig 配置 Hub 的 gRPC/TLS 监听地址。
type GRPCConfig struct {
	Listen string `toml:"listen"`
}

func Load(path string) (*Config, error) {
	cfg := &Config{}
	metadata, err := toml.DecodeFile(path, cfg)
	if err != nil {
		return nil, err
	}
	if metadata.IsDefined("agent_development") {
		return nil, fmt.Errorf("agent_development has been removed; move directory to the top level and remove enabled")
	}
	if cfg.Directory == "" {
		cfg.Directory = "~/.reinhub"
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
