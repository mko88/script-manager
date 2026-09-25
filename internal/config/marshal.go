package config

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"script-manager/internal/secret"

	"gopkg.in/yaml.v3"
)

func isHiddenValue(v *yaml.Node) bool {
	return v.Kind == yaml.ScalarNode && (secret.IsLocked(v.Value) || v.Value == secret.Unset)
}

// Marshal renders the shared config: every locked or unset env value is
// written as a bare !secret, so the file carries no ciphertext.
func (c *Config) Marshal() ([]byte, error) {
	var root yaml.Node
	if err := root.Encode(c); err != nil {
		return nil, err
	}
	for _, env := range envMappings(&root) {
		for i := 1; i < len(env.Content); i += 2 {
			if isHiddenValue(env.Content[i]) {
				env.Content[i] = &yaml.Node{Kind: yaml.ScalarNode, Tag: secretTag}
			}
		}
	}
	return yaml.Marshal(&root)
}

// CheckSecretItemNames reports an error if an item holding a locked or unset
// env value has no name, or shares its name with any other item — secret or
// not. buildLocalSecrets resolves "is this item still declared?" by name, so
// a same-named plain item lets a secret item's keys look undeclared and its
// ciphertext gets replaced by a stale orphan.
func (c *Config) CheckSecretItemNames() error {
	counts := make(map[string]int, len(c.Items))
	for _, item := range c.Items {
		counts[item.Name]++
	}
	for _, item := range c.Items {
		if !hasSecretValue(item.Env) {
			continue
		}
		if item.Name == "" {
			return errors.New("an item with PIN-locked values needs a name")
		}
		if counts[item.Name] > 1 {
			return fmt.Errorf("item %q: items with PIN-locked values need a unique name", item.Name)
		}
	}
	return nil
}

func hasSecretValue(env map[string]any) bool {
	for _, v := range env {
		if secret.IsLockedValue(v) || secret.IsUnset(v) {
			return true
		}
	}
	return false
}

// Save writes the local secrets file first, when there's a PIN, so a watcher
// on path never reloads a config whose secrets aren't on disk yet.
func (c *Config) Save(path string) error {
	if err := c.CheckSecretItemNames(); err != nil {
		return err
	}
	if c.Secrets != nil {
		if c.SecretsID == "" {
			id, err := NewSecretsID()
			if err != nil {
				return err
			}
			c.SecretsID = id
		}
		if err := c.writeLocalSecrets(); err != nil {
			return err
		}
	}
	out, err := c.Marshal()
	if err != nil {
		return err
	}
	return os.WriteFile(path, out, 0o644)
}

func (c *Config) writeLocalSecrets() error {
	path := LocalSecretsPath(c.SecretsID)
	if path == "" {
		return errors.New("no per-user data directory to store the secrets in")
	}
	existing, _ := readLocalSecrets(path)
	out, err := yaml.Marshal(c.buildLocalSecrets(existing))
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	return os.WriteFile(path, out, 0o600)
}

// buildLocalSecrets collects every locked value. Entries in existing for items
// or keys this config doesn't mention are kept — they may belong to another
// branch of a shared config — but only while they're under the same PIN
// params, since nothing re-encrypts them.
func (c *Config) buildLocalSecrets(existing *localSecrets) localSecrets {
	out := localSecrets{Secrets: c.Secrets, Env: lockedOnly(c.Env), Items: map[string]map[string]string{}}
	for _, item := range c.Items {
		if vals := lockedOnly(item.Env); len(vals) > 0 {
			out.Items[item.Name] = vals
		}
	}
	if existing != nil && secret.SameParams(existing.Secrets, c.Secrets) {
		keepOrphans(out.Env, existing.Env, c.Env)
		known := make(map[string]map[string]any, len(c.Items))
		for _, item := range c.Items {
			known[item.Name] = item.Env
		}
		for name, vals := range existing.Items {
			env, ok := known[name]
			if !ok {
				out.Items[name] = vals
				continue
			}
			if out.Items[name] == nil {
				out.Items[name] = map[string]string{}
			}
			keepOrphans(out.Items[name], vals, env)
			if len(out.Items[name]) == 0 {
				delete(out.Items, name)
			}
		}
	}
	if len(out.Items) == 0 {
		out.Items = nil
	}
	return out
}

func lockedOnly(env map[string]any) map[string]string {
	out := map[string]string{}
	for k, v := range env {
		if s, ok := v.(string); ok && secret.IsLocked(s) {
			out[k] = s
		}
	}
	return out
}

func keepOrphans(dst, existing map[string]string, env map[string]any) {
	for k, v := range existing {
		if _, declared := env[k]; !declared {
			dst[k] = v
		}
	}
}

func RemoveLocalSecrets(id string) error {
	path := LocalSecretsPath(id)
	if path == "" {
		return nil
	}
	if err := os.Remove(path); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return err
	}
	return nil
}
