// Geometry for the resizable/collapsible panel layout: drag handlers for the
// column and row dividers, and the flex styles that let one panel of a pair
// own the explicit size while the other soaks up the remainder. Pure
// math/DOM-event code — which panels exist, their persistence, and their
// state all stay in App.svelte.

export const HEADER_H = 33
export const MIN_PANEL = 60
export const MIN_COL = 180
export const RESIZER = 6

type DragOpts = {
  // Total size (width for a column drag, height for a row drag) of the
  // container being divided.
  getTotal: () => number
  get: () => number
  set: (v: number) => void
  // Called once on mouseup — the point to persist the final size.
  onDone: () => void
}

function drag(e: MouseEvent, opts: DragOpts, axis: 'x' | 'y', min: number, reserved: number) {
  e.preventDefault()
  const start = axis === 'x' ? e.clientX : e.clientY
  const startSize = opts.get()
  function onMove(ev: MouseEvent) {
    const max = opts.getTotal() - min - reserved
    const pos = axis === 'x' ? ev.clientX : ev.clientY
    opts.set(Math.min(max, Math.max(min, startSize + (pos - start))))
  }
  function onUp() {
    window.removeEventListener('mousemove', onMove)
    window.removeEventListener('mouseup', onUp)
    opts.onDone()
  }
  window.addEventListener('mousemove', onMove)
  window.addEventListener('mouseup', onUp)
}

export function dragColumn(e: MouseEvent, opts: DragOpts) {
  drag(e, opts, 'x', MIN_COL, RESIZER)
}

export function dragRow(e: MouseEvent, opts: DragOpts) {
  drag(e, opts, 'y', MIN_PANEL, RESIZER + HEADER_H)
}

// The "top" panel in a pair (Items/Details) gets an explicit height; the
// "bottom" panel (Actions/Command) fills whatever space is left. Collapsing
// either one just swaps who gets the fixed header-only height. Panels whose
// collapsed header shows a selected-item label that can wrap onto multiple
// lines (Items, Actions) get an auto flex-basis instead of the fixed
// HEADER_H so the wrapped text isn't clipped.
export function topStyle(topCollapsed: boolean, bottomCollapsed: boolean, size: number, autoCollapse = false) {
  if (topCollapsed) return autoCollapse ? `flex: 0 0 auto;` : `flex: 0 0 ${HEADER_H}px;`
  if (bottomCollapsed) return `flex: 1 1 auto;`
  return `flex: 0 0 ${size}px;`
}

export function bottomStyle(bottomCollapsed: boolean, autoCollapse = false) {
  if (bottomCollapsed) return autoCollapse ? `flex: 0 0 auto;` : `flex: 0 0 ${HEADER_H}px;`
  return `flex: 1 1 auto; min-height: 0;`
}
