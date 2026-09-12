import { dndzone } from 'svelte-dnd-action'
import type { DndEvent } from 'svelte-dnd-action'

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

export type SyncFn<T> = (e: CustomEvent<DndEvent<DndEntry<T>>>, final: boolean) => void
export type SortableParams<T> = { items: DndEntry<T>[]; onSync: SyncFn<T>; dragDisabled: boolean }

export function syncList<T>(config: {
  setEntries: (entries: DndEntry<T>[]) => void
  setDragging: (dragging: boolean) => void
  setList: (list: T[]) => void
}): SyncFn<T> {
  return (e, final) => {
    config.setEntries(e.detail.items)
    config.setDragging(!final)
    if (final) config.setList(e.detail.items.filter((w) => w.ref).map((w) => w.ref))
  }
}

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
