
export const HEADER_H = 33
export const MIN_PANEL = 60
export const MIN_COL = 180
export const RESIZER = 6

type DragOpts = {
  getTotal: () => number
  get: () => number
  set: (v: number) => void
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

export function topStyle(topCollapsed: boolean, bottomCollapsed: boolean, size: number, autoCollapse = false) {
  if (topCollapsed) return autoCollapse ? `flex: 0 0 auto;` : `flex: 0 0 ${HEADER_H}px;`
  if (bottomCollapsed) return `flex: 1 1 auto;`
  return `flex: 0 0 ${size}px;`
}

export function bottomStyle(bottomCollapsed: boolean, autoCollapse = false) {
  if (bottomCollapsed) return autoCollapse ? `flex: 0 0 auto;` : `flex: 0 0 ${HEADER_H}px;`
  return `flex: 1 1 auto; min-height: 0;`
}
