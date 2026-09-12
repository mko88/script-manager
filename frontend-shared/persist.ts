
export function loadPersisted<T extends Record<string, unknown>>(key: string, defaults: T): T {
  const out = { ...defaults }
  try {
    const saved = JSON.parse(localStorage.getItem(key) ?? '{}') as Record<string, unknown>
    for (const k in defaults) {
      if (typeof saved[k] === typeof defaults[k]) out[k] = saved[k] as T[Extract<keyof T, string>]
    }
  } catch {
  }
  return out
}

export function savePersisted(key: string, value: Record<string, unknown>) {
  localStorage.setItem(key, JSON.stringify(value))
}
