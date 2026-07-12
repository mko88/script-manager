// Tiny load/save pair for UI state persisted in localStorage (panel sizes,
// collapsed flags, view modes) — the load-on-mount/save-on-change pattern
// several panels share, with the corrupt/missing-data handling in one place.

// loadPersisted returns defaults overlaid with whatever was saved under key,
// keeping a saved field only when its type matches the default's — a corrupt
// or stale value (e.g. a string where a number belongs) falls back to that
// field's default instead of poisoning the UI state.
export function loadPersisted<T extends Record<string, unknown>>(key: string, defaults: T): T {
  const out = { ...defaults }
  try {
    const saved = JSON.parse(localStorage.getItem(key) ?? '{}') as Record<string, unknown>
    for (const k in defaults) {
      if (typeof saved[k] === typeof defaults[k]) out[k] = saved[k] as T[Extract<keyof T, string>]
    }
  } catch {
    // corrupt/missing entry — defaults already in place
  }
  return out
}

export function savePersisted(key: string, value: Record<string, unknown>) {
  localStorage.setItem(key, JSON.stringify(value))
}
