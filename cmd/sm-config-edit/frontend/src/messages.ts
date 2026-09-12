import messages from '../../../../internal/messages/configedit.json'

type Messages = typeof messages

type FlattenKeys<T, Prefix extends string = ''> = T extends string
  ? Prefix
  : {
      [K in keyof T & string]: FlattenKeys<T[K], `${Prefix}${Prefix extends '' ? '' : '.'}${K}`>
    }[keyof T & string]

export type MessageKey = FlattenKeys<Messages>

type Vars = Record<string, string | number>

let override: unknown = null

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
