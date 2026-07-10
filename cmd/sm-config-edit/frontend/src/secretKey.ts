// Mirrors internal/configedit's looksLikeSecretKey — a field/env-var name is
// treated as holding a secret the moment it looks like one (without waiting
// for a config reload to notice), whether that's FieldGrid defaulting a new
// field to secret/masked or App.svelte defaulting an inserted env reference
// to `{{mask .key}}` instead of a plain `{{.key}}`.
export function looksLikeSecretKey(key: string): boolean {
  const lower = key.toLowerCase()
  return lower.endsWith('secret') || lower.endsWith('password') || lower.endsWith('key')
}
