
type Subscriber = (value: string) => void

let current = ''
const subscribers = new Set<Subscriber>()

function set(value: string) {
  current = value
  for (const fn of subscribers) fn(value)
}

export const toastMessage = {
  subscribe(fn: Subscriber): () => void {
    subscribers.add(fn)
    fn(current)
    return () => subscribers.delete(fn)
  },
}

const TOAST_DURATION_MS = 3000

let timer: ReturnType<typeof setTimeout>

export function flash(msg: string) {
  set(msg)
  clearTimeout(timer)
  timer = setTimeout(() => set(''), TOAST_DURATION_MS)
}
