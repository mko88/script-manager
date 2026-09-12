package render

import "strings"

const ConfigFilePlaceholder = "#CONFIG_FILE#"

const noConfigFile = "_No config file loaded._"

func ExpandConfigFile(md, path string) string {
	if !strings.Contains(md, ConfigFilePlaceholder) {
		return md
	}
	if path == "" {
		path = noConfigFile
	}
	return strings.ReplaceAll(md, ConfigFilePlaceholder, path)
}
