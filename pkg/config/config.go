// Public Domain (-) 2018-present, The Elko Authors.
// See the Elko UNLICENSE file for details.

package config

import (
	"errors"
	"os"
	"path/filepath"
	"time"
)

var NotFound = errors.New("config: could not locate .elko/config.yaml in the current or parent directories")

type Elko struct {
	Clusters map[string]struct {
		Env      []string `yaml:"env"`
		Services []string `yaml:"services"`
	} `yaml:"clusters"`
	Dev struct {
		DataSystems []string          `yaml:"datasystems"`
		Relay       []string          `yaml:"relay"`
		EnvSet      map[string]string `yaml:"env.set"`
	} `yaml:"dev"`
}

type Node struct {
	Cluster       string        `yaml:"cluster"`
	ConsulKey     string        `yaml:"consul.key"`
	ConsulServers []string      `yaml:"consul.servers"`
	HostType      string        `yaml:"host.type"`
	LeaseDuration time.Duration `yaml:"lease.duration"`
}

type Service map[string]interface{}

func GetRoot() (string, error) {
	root, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for {
		cfg := filepath.Join(root, ".elko", "config.yaml")
		_, err = os.Stat(cfg)
		if err == nil {
			return root, nil
		}
		if !os.IsNotExist(err) {
			return "", err
		}
		if root == "/" {
			return "", NotFound
		}
		root = filepath.Dir(root)
	}
}
