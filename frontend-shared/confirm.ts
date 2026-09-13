export type ConfirmRequest = {
  title: string
  message: string
  confirmLabel: string
  cancelLabel: string
}

type Subscriber = (value: ConfirmRequest | null) => void

let current: ConfirmRequest | null = null
const subscribers = new Set<Subscriber>()

function set(value: ConfirmRequest | null) {
  current = value
  for (const fn of subscribers) fn(value)
}

export const confirmRequest = {
  subscribe(fn: Subscriber): () => void {
    subscribers.add(fn)
    fn(current)
    return () => subscribers.delete(fn)
  },
}

let answer: ((ok: boolean) => void) | null = null

// Resolves once the host component reports what the user chose. Only one
// question is on screen at a time; a second one cancels the first rather than
// leaving its caller waiting forever.
export function askConfirm(request: ConfirmRequest): Promise<boolean> {
  answer?.(false)
  return new Promise((resolve) => {
    answer = resolve
    set(request)
  })
}

export function answerConfirm(ok: boolean) {
  const resolve = answer
  answer = null
  set(null)
  resolve?.(ok)
}
