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
