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
  root.style.setProperty('--sm-ui-scale', String(clampScale(prefs.scalePercent) / 100))
  requestAnimationFrame(fitRootToViewport)
}

// The interface is scaled with CSS zoom (see theme.css), which leaves the
// app root needing a height of exactly one window. WebView2 and WebKitGTK
// disagree on what 100vh means inside a zoomed element — one counts the
// zoom, the other doesn't — so no fixed formula fits both: whichever way
// it's written, one engine ends up with scrollbars and the other with dead
// space below the app. So don't assume either: measure what this engine
// actually laid out and keep the correction in --sm-vh-fit, which the app
// roots multiply their 100vh by. Height is linear in the factor, so one
// pass lands exactly on the window.
export function fitRootToViewport(): void {
  const root = document.documentElement
  const app = document.querySelector<HTMLElement>('.app-root')
  if (!app) return
  const wanted = root.clientHeight
  const measured = app.getBoundingClientRect().height
  if (!wanted || !measured) return
  const current = Number(root.style.getPropertyValue('--sm-vh-fit')) || 1
  const next = current * (wanted / measured)
  if (Math.abs(next - current) > 0.001) root.style.setProperty('--sm-vh-fit', String(next))
}

// A resize can't change the factor — it's a ratio — but it does bring the
// app root back after the window was too small to lay it out, and after
// script-manager-gui's shrink badge replaces it entirely.
if (typeof window !== 'undefined') {
  window.addEventListener('resize', () => requestAnimationFrame(fitRootToViewport))
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
