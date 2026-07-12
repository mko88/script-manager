// One shared transient-notification channel per app window. flash() is the
// only way UI code shows a toast; Toast.svelte subscribes to the store
// directly, so neither the message nor a flash callback ever needs to be
// threaded through component props.
//
// Hand-rolled store contract ({ subscribe }) instead of svelte/store:
// frontend-shared sits outside both apps' node_modules, so a bare `svelte`
// import doesn't resolve from here — and Svelte's `$store` auto-subscription
// only requires the contract, not the package.

type Subscriber = (value: string) => void

let current = ''
const subscribers = new Set<Subscriber>()

function set(value: string) {
  current = value
  for (const fn of subscribers) fn(value)
}

// Readable store view for Toast.svelte; everything else should call flash().
export const toastMessage = {
  subscribe(fn: Subscriber): () => void {
    subscribers.add(fn)
    fn(current)
    return () => subscribers.delete(fn)
  },
}

const TOAST_DURATION_MS = 3000

let timer: ReturnType<typeof setTimeout>

// Shows msg as a transient toast, replacing (and re-timing) whatever toast
// is currently visible.
export function flash(msg: string) {
  set(msg)
  clearTimeout(timer)
  timer = setTimeout(() => set(''), TOAST_DURATION_MS)
}
