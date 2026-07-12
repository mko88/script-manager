// Drag-and-drop reordering machinery for the master lists (Items, Action
// Groups, Actions), via svelte-dnd-action rather than native HTML5
// drag-and-drop. Native dnd's cursor is browser-controlled, and it disagreed
// with the live reorder + animation (dragover hit-tests against the real,
// already-reordered layout, while the FLIP transform visually lags behind
// it) — that mismatch is what read as the cursor flickering between "grab"
// and a no-drop icon. svelte-dnd-action drives everything from pointer
// events instead, so there's no browser drag cursor involved at all, and it
// handles the live-reorder animation and cancelled-drag revert internally.
import { dndzone } from 'svelte-dnd-action'
import type { DndEvent } from 'svelte-dnd-action'

// dndzone needs every item to carry an "id" it can track across reorders.
// None of items/actions/action groups reliably have one — an item has no id
// field at all, and a brand-new action/action group defaults its own id to
// "" (several can exist before being named) — so each list is wrapped in an
// {id, ref} pair with a synthetic, session-only id (a WeakMap keyed by
// object identity; never touches the saved data) instead of reusing the
// domain id field.
let dndSeq = 0
const dndIds = new WeakMap<object, string>()
function dndId(ref: object): string {
  let id = dndIds.get(ref)
  if (id === undefined) {
    id = `d${dndSeq++}`
    dndIds.set(ref, id)
  }
  return id
}

export type DndEntry<T> = { id: string; ref: T }

export function wrap<T extends object>(list: T[]): DndEntry<T>[] {
  return list.map((ref) => ({ id: dndId(ref), ref }))
}

// svelte-dnd-action's consider/finalize are custom events the dndzone
// action adds to its node, not real attributes of a plain <div> — this
// project's Svelte/svelte-check versions don't have a working ambient
// typing hook for that, so on:consider/on:finalize on the element itself
// won't type-check. Attaching them here via plain addEventListener
// sidesteps Svelte's (mistaken) typed-attribute check entirely; nothing
// about actual behavior changes.
export type SyncFn<T> = (e: CustomEvent<DndEvent<DndEntry<T>>>, final: boolean) => void
export type SortableParams<T> = { items: DndEntry<T>[]; onSync: SyncFn<T>; dragDisabled: boolean }

export function sortableList<T extends object>(node: HTMLElement, params: SortableParams<T>) {
  const zone = dndzone(node, { items: params.items, flipDurationMs: 200, dragDisabled: params.dragDisabled })
  const considerHandler = (e: Event) => params.onSync(e as CustomEvent<DndEvent<DndEntry<T>>>, false)
  const finalizeHandler = (e: Event) => params.onSync(e as CustomEvent<DndEvent<DndEntry<T>>>, true)
  node.addEventListener('consider', considerHandler)
  node.addEventListener('finalize', finalizeHandler)
  return {
    update(newParams: SortableParams<T>) {
      zone.update?.({ items: newParams.items, flipDurationMs: 200, dragDisabled: newParams.dragDisabled })
    },
    destroy() {
      node.removeEventListener('consider', considerHandler)
      node.removeEventListener('finalize', finalizeHandler)
      zone.destroy?.()
    },
  }
}
