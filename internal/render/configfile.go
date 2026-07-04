package render

import "strings"

// ConfigFilePlaceholder is a literal placeholder a details template can
// include verbatim (not inside {{ }}) to show the full path of the config
// file actually in use — useful for telling config.yaml and config-win.yaml
// apart when both exist. Expanded the same way as #ALL_ENV_LIST#/#ALL_ENV_TABLE#.
const ConfigFilePlaceholder = "#CONFIG_FILE#"

const noConfigFile = "_No config file loaded._"

// ExpandConfigFile replaces #CONFIG_FILE# in md, if present, with path.
func ExpandConfigFile(md, path string) string {
	if !strings.Contains(md, ConfigFilePlaceholder) {
		return md
	}
	if path == "" {
		path = noConfigFile
	}
	return strings.ReplaceAll(md, ConfigFilePlaceholder, path)
}
