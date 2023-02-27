package config

import (
	"os"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Type            string                `yaml:"kind"`
	HTTP            HTTPConfig            `yaml:"http"`
	JSON            JsonConfig            `yaml:"json"`
	Instrumentation InstrumentationConfig `yaml:"instrumentation"`
}

type JsonConfig struct {
	Path string
}

type HTTPConfig struct {
	Address string `yaml:"address"`
	Port    string `yaml:"port"`
}

type InstrumentationConfig struct {
	Enabled  bool   `yaml:"enabled"`
	Endpoint string `yaml:"endpoint"`
}

func Load(path string) (*Config, error) {
	fp, err := os.OpenFile(path, os.O_RDONLY, os.ModePerm)
	if err != nil {
		return nil, err
	}

	var cfg Config
	decoder := yaml.NewDecoder(fp)
	if err := decoder.Decode(&cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}
