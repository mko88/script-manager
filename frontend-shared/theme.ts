export type Theme = 'dark' | 'light'

const STORAGE_KEY = 'sm-theme'

export function getTheme(): Theme {
  return localStorage.getItem(STORAGE_KEY) === 'light' ? 'light' : 'dark'
}

function applyTheme(theme: Theme) {
  document.documentElement.setAttribute('data-theme', theme)
}

// Call as the very first line of main.ts, before Svelte mounts — sets
// data-theme on <html> synchronously so there's no flash of the wrong theme.
export function initTheme(): Theme {
  const theme = getTheme()
  applyTheme(theme)
  return theme
}

export function setTheme(theme: Theme) {
  localStorage.setItem(STORAGE_KEY, theme)
  applyTheme(theme)
}

// Reconciles the locally cached theme (localStorage, per-app since each is
// its own WebView) against the value persisted by the Go backend, which is
// shared by both apps via a file next to the executables (internal/theme) —
// so switching the theme in one app is picked up by the other the next
// time it starts. getRemote is each app's own bound GetTheme call; the two
// apps bind under different Wails namespaces, so this can't import a
// shared binding directly. Best-effort: a rejected call leaves the locally
// cached theme in place rather than throwing.
export async function syncTheme(getRemote: () => Promise<string>): Promise<Theme> {
  let remote: string
  try {
    remote = await getRemote()
  } catch {
    return getTheme()
  }
  const theme: Theme = remote === 'light' ? 'light' : 'dark'
  if (theme !== getTheme()) {
    setTheme(theme)
  }
  return theme
}
