export const UI_SCALE = { min: 70, max: 200, step: 10, default: 100 } as const

export type UIPrefs = { fontUi?: string; fontMono?: string; scalePercent?: number }

const UI_STACK = `-apple-system, BlinkMacSystemFont, "Segoe UI", "Roboto", "Helvetica Neue", sans-serif`
const MONO_STACK = `"SF Mono", Consolas, "Liberation Mono", monospace`

export function applyUIPrefs(prefs: UIPrefs): void {
  const root = document.documentElement
  const ui = (prefs.fontUi ?? '').trim()
  const mono = (prefs.fontMono ?? '').trim()
  root.style.setProperty('--sm-font-ui', ui ? `"${ui}", "Nunito", ${UI_STACK}` : `"Nunito", ${UI_STACK}`)
  root.style.setProperty('--sm-font-mono', mono ? `"${mono}", ${MONO_STACK}` : MONO_STACK)
  scale = clampScale(prefs.scalePercent) / 100
  root.style.setProperty('--sm-ui-scale', String(scale))
  setRootHeight()
}

let scale = 1

// The app root is zoomed (see theme.css) and must still be one window tall.
// In px, pre-divided: WebView2 and WebKitGTK disagree on what a viewport
// unit means inside a zoomed element, but not on what zoom does to a length.
function setRootHeight(): void {
  const root = document.documentElement
  root.style.setProperty('--sm-root-height', `${root.clientHeight / scale}px`)
}

if (typeof window !== 'undefined') {
  window.addEventListener('resize', setRootHeight)
}

export function clampScale(percent: number | undefined): number {
  if (!percent || percent <= 0) return UI_SCALE.default
  return Math.min(UI_SCALE.max, Math.max(UI_SCALE.min, percent))
}

// Ctrl +, Ctrl - and Ctrl 0. Both spellings of each key: the main row's +
// arrives as "+" or "=" depending on layout, and the numpad sends its own
// codes. Returns the requested percent, or null when the event isn't ours.
export function scaleFromKeydown(e: KeyboardEvent, current: number): number | null {
  if (!e.ctrlKey || e.altKey) return null
  if (e.key === '+' || e.key === '=' || e.code === 'NumpadAdd') {
    return clampScale(current + UI_SCALE.step)
  }
  if (e.key === '-' || e.key === '_' || e.code === 'NumpadSubtract') {
    return clampScale(current - UI_SCALE.step)
  }
  if (e.key === '0' || e.code === 'Numpad0') return UI_SCALE.default
  return null
}
