// Shared t()/override machinery for both apps' message packs. Each app calls
// createMessages with its own compiled JSON (gui.json / configedit.json from
// internal/messages — the single canonical copies, embedded by the Go backend
// too), so the returned t() is typed against exactly that app's keys.

// Flattens the nested JSON into a dotted-path key union, e.g. "toast.saveFailed",
// so a typo in a t() call is a compile-time error instead of a silent blank string.
export type FlattenKeys<T, Prefix extends string = ''> = T extends string
  ? Prefix
  : {
      [K in keyof T & string]: FlattenKeys<T[K], `${Prefix}${Prefix extends '' ? '' : '.'}${K}`>
    }[keyof T & string]

type Vars = Record<string, string | number>

function lookup(obj: unknown, parts: string[]): unknown {
  let node: unknown = obj
  for (const p of parts) {
    node = (node as Record<string, unknown> | undefined)?.[p]
  }
  return node
}

export function createMessages<M>(messages: M) {
  // Runtime override loaded from GetMessages() by main.ts before the app is
  // mounted — see setMessageOverride. null means "none loaded" (missing or
  // invalid on-disk file), in which case resolve() falls back to the
  // compiled messages JSON entirely.
  let override: unknown = null

  // Called once by main.ts, before the Svelte app is constructed — top-level
  // `let`/`$:` initializers run synchronously at component creation, before
  // onMount ever fires, so the override must already be in place by the time
  // any of them (or later template/`t()` calls) run.
  function setMessageOverride(data: unknown) {
    override = data
  }

  function resolve(path: string): string {
    const parts = path.split('.')
    const overridden = override ? lookup(override, parts) : undefined
    const node = typeof overridden === 'string' ? overridden : lookup(messages, parts)
    if (typeof node !== 'string') throw new Error(`Missing message: ${path}`)
    return node
  }

  function t(key: FlattenKeys<M>, vars?: Vars): string {
    let s = resolve(key)
    if (vars) {
      for (const [k, v] of Object.entries(vars)) {
        s = s.split(`{${k}}`).join(String(v))
      }
    }
    return s
  }

  return { t, setMessageOverride }
}
