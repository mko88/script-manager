// Moved into internal/messages so the Go backend can embed it directly too
// (both apps' compiled defaults live in one shared package now — see
// internal/messages/messages.go) — this is still the single canonical copy,
// not a duplicate.
import messages from '../../../../internal/messages/gui.json'

type Messages = typeof messages

// Flattens the nested JSON into a dotted-path key union, e.g. "toast.runFailed",
// so a typo in a t() call is a compile-time error instead of a silent blank string.
type FlattenKeys<T, Prefix extends string = ''> = T extends string
  ? Prefix
  : {
      [K in keyof T & string]: FlattenKeys<T[K], `${Prefix}${Prefix extends '' ? '' : '.'}${K}`>
    }[keyof T & string]

export type MessageKey = FlattenKeys<Messages>

type Vars = Record<string, string | number>

// Runtime override loaded from GetMessages() by main.ts before the app is
// mounted — see setMessageOverride. null means "none loaded" (missing or
// invalid on-disk file), in which case resolve() falls back to the
// compiled messages.json entirely.
let override: unknown = null

// Called once by main.ts, before the Svelte app is constructed — top-level
// `let`/`$:` initializers run synchronously at component creation, before
// onMount ever fires, so the override must already be in place by the time
// any of them (or later template/`t()` calls) run.
export function setMessageOverride(data: unknown) {
  override = data
}

function lookup(obj: unknown, parts: string[]): unknown {
  let node: unknown = obj
  for (const p of parts) {
    node = (node as Record<string, unknown> | undefined)?.[p]
  }
  return node
}

function resolve(path: string): string {
  const parts = path.split('.')
  const overridden = override ? lookup(override, parts) : undefined
  const node = typeof overridden === 'string' ? overridden : lookup(messages, parts)
  if (typeof node !== 'string') throw new Error(`Missing message: ${path}`)
  return node
}

export function t(key: MessageKey, vars?: Vars): string {
  let s = resolve(key)
  if (vars) {
    for (const [k, v] of Object.entries(vars)) {
      s = s.split(`{${k}}`).join(String(v))
    }
  }
  return s
}
