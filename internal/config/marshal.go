package config

import "gopkg.in/yaml.v3"

// Marshal serializes the config back to YAML. This is the write-side
// counterpart to LoadWithError/LoadFromWithError, used by config-editing
// tools rather than the TUI/GUI (which only ever read). Round-tripping
// through Marshal does not preserve comments or the original file's exact
// formatting/key order — see DisplayList's legacy-single-mapping format,
// which always marshals back out as the modern sequence form.
func (c *Config) Marshal() ([]byte, error) {
	return yaml.Marshal(c)
}
