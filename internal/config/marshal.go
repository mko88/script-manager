package config

import "gopkg.in/yaml.v3"

func (c *Config) Marshal() ([]byte, error) {
	return yaml.Marshal(c)
}
