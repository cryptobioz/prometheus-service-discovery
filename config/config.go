package config

import (
	"gopkg.in/yaml.v2"
)

// Config stores main configuration options
type Config struct {
	Config struct {
		Output struct {
			Type string `yaml:"type,omitempty"`
			Path string `yaml:"path,omitempty"`
		} `yaml:"output,omitempty"`
		LogLevel string `yaml:"log_level,omitempty"`
	} `yaml:"config,omitempty"`
}

// LoadConfig read config file and unmarshal content into a struct
func LoadConfig(y []byte) (conf Config, err error) {
	err = yaml.Unmarshal(y, &conf)
	if err != nil {
		return
	}

	if conf.Config.Output.Type == "" {
		conf.Config.Output.Type = "stdout"
	}
	return
}
