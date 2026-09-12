
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
  let override: unknown = null

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
