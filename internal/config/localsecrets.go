package config

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"

	"script-manager/internal/appdata"
	"script-manager/internal/secret"

	"gopkg.in/yaml.v3"
)

const secretTag = "!secret"

var secretsIDPattern = regexp.MustCompile(`^[0-9a-f]{32}$`)

// localSecrets is the per-user file holding a config's PIN params and the
// ciphertext for every value the shared config declares with !secret.
type localSecrets struct {
	Secrets *secret.Params               `yaml:"secrets,omitempty"`
	Env     map[string]string            `yaml:"env,omitempty"`
	Items   map[string]map[string]string `yaml:"items,omitempty"`
}

func NewSecretsID() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

// LocalSecretsPath is where this user's secrets for the config with the given
// ID live, or "" when there is no ID or no app-data directory.
func LocalSecretsPath(id string) string {
	dir := appdata.Dir()
	if dir == "" || id == "" {
		return ""
	}
	return filepath.Join(dir, "secrets", id+".yaml")
}

func mappingValue(m *yaml.Node, key string) *yaml.Node {
	if m == nil || m.Kind != yaml.MappingNode {
		return nil
	}
	for i := 0; i+1 < len(m.Content); i += 2 {
		if m.Content[i].Value == key {
			return m.Content[i+1]
		}
	}
	return nil
}

func envMappings(root *yaml.Node) []*yaml.Node {
	var out []*yaml.Node
	if env := mappingValue(root, KeyEnv); env != nil && env.Kind == yaml.MappingNode {
		out = append(out, env)
	}
	if items := mappingValue(root, "items"); items != nil && items.Kind == yaml.SequenceNode {
		for _, item := range items.Content {
			if env := mappingValue(item, KeyEnv); env != nil && env.Kind == yaml.MappingNode {
				out = append(out, env)
			}
		}
	}
	return out
}

// claimSecretTags turns each env-level !secret into the Unset sentinel, then
// rejects any !secret left anywhere else.
func claimSecretTags(root *yaml.Node) error {
	for _, env := range envMappings(root) {
		for i := 1; i < len(env.Content); i += 2 {
			v := env.Content[i]
			if v.Tag != secretTag {
				continue
			}
			if v.Kind != yaml.ScalarNode {
				return fmt.Errorf("line %d: %s takes no value", v.Line, secretTag)
			}
			v.Tag = "!!str"
			v.Value = secret.Unset
			v.Style = 0
		}
	}
	return findStraySecretTag(root)
}

func findStraySecretTag(n *yaml.Node) error {
	if n.Tag == secretTag {
		return fmt.Errorf("line %d: %s is only allowed as a value under env:", n.Line, secretTag)
	}
	for _, c := range n.Content {
		if err := findStraySecretTag(c); err != nil {
			return err
		}
	}
	return nil
}

func parseConfig(data []byte) (Config, error) {
	var cfg Config
	var doc yaml.Node
	if err := yaml.Unmarshal(data, &doc); err != nil {
		return cfg, err
	}
	if len(doc.Content) == 0 {
		return cfg, nil
	}
	root := doc.Content[0]
	if err := claimSecretTags(root); err != nil {
		return cfg, err
	}
	if err := root.Decode(&cfg); err != nil {
		return cfg, err
	}
	if inline := mappingValue(root, "secrets"); inline != nil {
		var p secret.Params
		if err := inline.Decode(&p); err != nil {
			return cfg, fmt.Errorf("secrets: %w", err)
		}
		cfg.Secrets = &p
	}
	if cfg.SecretsID != "" && !secretsIDPattern.MatchString(cfg.SecretsID) {
		return cfg, fmt.Errorf("secretsId %q must be 32 lowercase hex characters", cfg.SecretsID)
	}
	return cfg, nil
}

func readLocalSecrets(path string) (*localSecrets, error) {
	data, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var local localSecrets
	if err := yaml.Unmarshal(data, &local); err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	return &local, nil
}

// applyLocalSecrets fills each Unset slot from this user's local secrets file.
// Local entries the shared config doesn't declare are ignored.
func (c *Config) applyLocalSecrets() error {
	path := LocalSecretsPath(c.SecretsID)
	if path == "" {
		return nil
	}
	local, err := readLocalSecrets(path)
	if err != nil || local == nil {
		return err
	}
	if local.Secrets != nil {
		c.Secrets = local.Secrets
	}
	fillUnset(c.Env, local.Env)
	for i := range c.Items {
		fillUnset(c.Items[i].Env, local.Items[c.Items[i].Name])
	}
	return nil
}

func fillUnset(env map[string]any, values map[string]string) {
	for k, v := range env {
		if s, ok := values[k]; ok && secret.IsUnset(v) && secret.IsLocked(s) {
			env[k] = s
		}
	}
}
