export function looksLikeSecretKey(key: string): boolean {
  const lower = key.toLowerCase()
  return lower.endsWith('secret') || lower.endsWith('password') || lower.endsWith('key')
}
