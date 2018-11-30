package config

import (
	"gopkg.in/yaml.v2"
)

type Config struct {
	Output struct {
		Type string `yaml:"type,omitempty"`
		Path string `yaml:"path,omitempty"`
	} `yaml:"output,omitempty"`
	LogLevel string `yaml:"log_level,omitempty"`
}

func LoadConfig(y []byte) (conf Config, err error) {
	err = yaml.Unmarshal(y, conf)
	if err != nil {
		return
	}
	return
}
