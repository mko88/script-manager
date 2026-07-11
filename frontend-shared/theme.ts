export type Theme = 'dark' | 'light' | 'custom'
export type CustomPalette = Record<string, string>
export interface ThemeState {
  active: string
  custom?: CustomPalette
}

const STORAGE_KEY = 'sm-theme'
const CUSTOM_STORAGE_KEY = 'sm-theme-custom'

// The canonical list of every customizable CSS custom property (without
// the "--sm-" prefix), grouped for the theme editor's UI — matches
// frontend-shared/theme.css's :root block exactly; keep both in sync.
export const TOKEN_GROUPS: { label: string; tokens: string[] }[] = [
  { label: 'Backgrounds', tokens: ['bg', 'bg-alt', 'bg-deep', 'panel-header', 'border'] },
  { label: 'Text', tokens: ['text', 'text-muted', 'text-faint'] },
  { label: 'Accents', tokens: ['accent', 'accent-warm', 'code', 'masked', 'error', 'line-number'] },
  {
    label: 'Effects',
    tokens: [
      'hover',
      'hover-strong',
      'tint-hover',
      'overlay-soft',
      'overlay-medium',
      'overlay-strong',
      'shadow',
      'btn-primary-hover',
    ],
  },
]
export const TOKEN_NAMES: string[] = TOKEN_GROUPS.flatMap((g) => g.tokens)

function normalizeTheme(value: string): Theme {
  return value === 'light' || value === 'custom' ? value : 'dark'
}

export function getTheme(): Theme {
  return normalizeTheme(localStorage.getItem(STORAGE_KEY) ?? '')
}

export function getCustomPalette(): CustomPalette | null {
  const raw = localStorage.getItem(CUSTOM_STORAGE_KEY)
  if (!raw) return null
  try {
    return JSON.parse(raw) as CustomPalette
  } catch {
    return null
  }
}

// Reads every token's value for a static base theme straight off theme.css
// (never hardcoded here, so it can't drift) — used by the theme editor to
// seed its working copy and by its "Reset to Dark"/"Reset to Light"
// buttons. Momentarily flips data-theme and clears any inline custom
// overrides to read the stylesheet's own values via getComputedStyle, then
// restores exactly what was there before — synchronous, so nothing ever
// paints the intermediate state.
export function readPaletteFor(base: 'dark' | 'light'): CustomPalette {
  const root = document.documentElement
  const prevAttr = root.getAttribute('data-theme')
  const prevInline: CustomPalette = {}
  for (const name of TOKEN_NAMES) prevInline[name] = root.style.getPropertyValue(`--sm-${name}`)

  for (const name of TOKEN_NAMES) root.style.removeProperty(`--sm-${name}`)
  root.setAttribute('data-theme', base)
  const style = getComputedStyle(root)
  const palette: CustomPalette = {}
  for (const name of TOKEN_NAMES) palette[name] = style.getPropertyValue(`--sm-${name}`).trim()

  root.setAttribute('data-theme', prevAttr ?? 'dark')
  for (const name of TOKEN_NAMES) {
    if (prevInline[name]) root.style.setProperty(`--sm-${name}`, prevInline[name])
  }
  return palette
}

function applyTheme(theme: Theme, custom?: CustomPalette | null) {
  const root = document.documentElement
  if (theme === 'custom' && custom) {
    // "dark" is a structural fallback only — every token below gets an
    // explicit inline override, so the underlying :root values never show.
    root.setAttribute('data-theme', 'dark')
    for (const name of TOKEN_NAMES) {
      const value = custom[name]
      if (value) root.style.setProperty(`--sm-${name}`, value)
      else root.style.removeProperty(`--sm-${name}`)
    }
  } else {
    root.setAttribute('data-theme', theme)
    // Clear any inline overrides a previous "custom" selection left behind
    // — otherwise they'd keep shadowing the dark/light stylesheet values.
    for (const name of TOKEN_NAMES) root.style.removeProperty(`--sm-${name}`)
  }
}

// Call as the very first line of main.ts, before Svelte mounts — sets
// data-theme (and any custom overrides) on <html> synchronously so
// there's no flash of the wrong theme.
export function initTheme(): Theme {
  const theme = getTheme()
  applyTheme(theme, getCustomPalette())
  return theme
}

export function setTheme(theme: Theme, custom?: CustomPalette | null) {
  localStorage.setItem(STORAGE_KEY, theme)
  if (custom) localStorage.setItem(CUSTOM_STORAGE_KEY, JSON.stringify(custom))
  applyTheme(theme, custom ?? getCustomPalette())
}

// Reconciles the locally cached theme (localStorage, per-app since each is
// its own WebView) against the value persisted by the Go backend, which is
// shared by both apps via a file next to the executables (internal/theme) —
// so switching the theme in one app, or saving a custom palette in
// sm-config-edit, is picked up by the other the next time it starts.
// getRemote is each app's own bound GetTheme call; the two apps bind under
// different Wails namespaces, so this can't import a shared binding
// directly. Best-effort: a rejected call leaves the locally cached theme
// in place rather than throwing. Returns whether a custom palette exists
// (for the dropdown), which the caller can't otherwise tell apart from
// "not saved yet" without inspecting the result itself.
export async function syncTheme(
  getRemote: () => Promise<ThemeState>,
): Promise<{ theme: Theme; hasCustomTheme: boolean }> {
  try {
    const remote = await getRemote()
    const theme = normalizeTheme(remote.active)
    const hasCustomTheme = !!remote.custom
    if (remote.custom) localStorage.setItem(CUSTOM_STORAGE_KEY, JSON.stringify(remote.custom))
    if (theme !== getTheme() || remote.custom) {
      setTheme(theme, remote.custom)
    }
    return { theme, hasCustomTheme }
  } catch {
    return { theme: getTheme(), hasCustomTheme: !!getCustomPalette() }
  }
}
