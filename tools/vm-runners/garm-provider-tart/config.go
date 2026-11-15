package main

import (
	"errors"
	"fmt"

	"github.com/BurntSushi/toml"
)

type config struct {
	Tart     string            `toml:"tart"`
	StateDir string            `toml:"state_dir"`
	CPU      int               `toml:"cpu"`
	Memory   int               `toml:"memory"`
	Images   map[string]string `toml:"images"`
}

func readConfig(path string) (config, error) {
	var cfg config
	if path == "" {
		return config{}, errors.New("GARM_PROVIDER_CONFIG_FILE is not set")
	}
	if _, err := toml.DecodeFile(path, &cfg); err != nil {
		return config{}, fmt.Errorf("read config %s: %w", path, err)
	}
	if cfg.Tart == "" {
		cfg.Tart = "tart"
	}
	if cfg.StateDir == "" {
		return config{}, fmt.Errorf("config %s: state_dir is empty", path)
	}
	if len(cfg.Images) == 0 {
		return config{}, fmt.Errorf("config %s: images is empty", path)
	}
	return cfg, nil
}
