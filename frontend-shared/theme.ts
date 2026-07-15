// A theme is either the built-in "dark"/"light" or the name of an entry
// in a themes map — a custom theme's name doubles as its selector, so
// there's no fixed enum here anymore.
export type Theme = string
export type CustomPalette = Record<string, string>
export interface ThemeState {
  active: string
  themes?: Record<string, CustomPalette>
}

const STORAGE_KEY = 'sm-theme'
const THEMES_STORAGE_KEY = 'sm-theme-themes'

// The canonical list of every customizable CSS custom property (without
// the "--sm-" prefix), grouped for the theme editor's UI — matches
// frontend-shared/theme.css's :root block exactly; keep both in sync.
export const TOKEN_GROUPS: { label: string; tokens: string[] }[] = [
  {
    label: 'Backgrounds',
    tokens: ['bg', 'bg-alt', 'row-bg', 'bg-secondary', 'bg-deep', 'panel-header', 'border', 'bg-primary'],
  },
  {
    label: 'Text',
    tokens: [
      'text',
      'text-muted',
      'text-faint',
      'text-heading',
      'text-highlight',
      'section-title',
      'text-primary',
      'masked',
      'warning',
      'error',
      'run-active',
      'run-ok',
      'run-error',
      'line-number',
    ],
  },
  {
    label: 'Effects',
    tokens: ['secondary-hover', 'tint-hover', 'overlay-soft', 'scrollbar', 'shadow', 'primary-hover'],
  },
]
export const TOKEN_NAMES: string[] = TOKEN_GROUPS.flatMap((g) => g.tokens)

export function getTheme(): Theme {
  return localStorage.getItem(STORAGE_KEY) || 'dark'
}

export function getThemes(): Record<string, CustomPalette> | null {
  const raw = localStorage.getItem(THEMES_STORAGE_KEY)
  if (!raw) return null
  try {
    return JSON.parse(raw) as Record<string, CustomPalette>
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

function applyTheme(theme: Theme, themes?: Record<string, CustomPalette> | null) {
  const root = document.documentElement
  const palette = theme !== 'dark' && theme !== 'light' ? themes?.[theme] : undefined
  if (palette) {
    // "dark" is a structural fallback only — every token below gets an
    // explicit inline override, so the underlying :root values never show.
    root.setAttribute('data-theme', 'dark')
    for (const name of TOKEN_NAMES) {
      const value = palette[name]
      if (value) root.style.setProperty(`--sm-${name}`, value)
      else root.style.removeProperty(`--sm-${name}`)
    }
  } else {
    root.setAttribute('data-theme', theme === 'light' ? 'light' : 'dark')
    // Clear any inline overrides a previous custom-theme selection left
    // behind — otherwise they'd keep shadowing the dark/light stylesheet
    // values.
    for (const name of TOKEN_NAMES) root.style.removeProperty(`--sm-${name}`)
  }
}

// Call as the very first line of main.ts, before Svelte mounts — sets
// data-theme (and any custom overrides) on <html> synchronously so
// there's no flash of the wrong theme.
export function initTheme(): Theme {
  const theme = getTheme()
  applyTheme(theme, getThemes())
  return theme
}

export function setTheme(theme: Theme, themes?: Record<string, CustomPalette> | null) {
  localStorage.setItem(STORAGE_KEY, theme)
  if (themes) localStorage.setItem(THEMES_STORAGE_KEY, JSON.stringify(themes))
  applyTheme(theme, themes ?? getThemes())
}

// Applies a ThemeState fetched from the Go backend, mirroring it into the
// local cache exactly rather than just merging into it — otherwise a
// custom theme cached from an earlier install/session would linger
// forever once the backend genuinely has none (e.g. its sm-theme.json was
// deleted or never existed on this machine). Shared by syncTheme (polled
// once at startup) and watchTheme (pushed live) so both reconcile the same
// way.
function applyRemoteState(remote: ThemeState): { theme: Theme; themes: Record<string, CustomPalette> | null } {
  const theme = remote.active || 'dark'
  const themes = remote.themes ?? null
  if (themes) {
    localStorage.setItem(THEMES_STORAGE_KEY, JSON.stringify(themes))
  } else {
    localStorage.removeItem(THEMES_STORAGE_KEY)
  }
  setTheme(theme, themes)
  return { theme, themes }
}

// Reconciles the locally cached theme (localStorage, per-app since each is
// its own WebView) against the value persisted by the Go backend, which is
// shared by both apps via a file next to the executables (internal/theme) —
// so switching the theme in one app, or saving a custom theme in
// sm-config-edit, is picked up by the other the next time it starts.
// getRemote is each app's own bound GetTheme call; the two apps bind under
// different Wails namespaces, so this can't import a shared binding
// directly. Best-effort: a rejected call leaves the locally cached theme
// in place rather than throwing.
export async function syncTheme(
  getRemote: () => Promise<ThemeState>,
): Promise<{ theme: Theme; themes: Record<string, CustomPalette> | null }> {
  try {
    return applyRemoteState(await getRemote())
  } catch {
    return { theme: getTheme(), themes: getThemes() }
  }
}

// The Wails event name internal/gui/themewatch.go emits on — must match
// its ThemeChangedEvent constant; there's no way to share a literal across
// the Go/JS boundary here, so keep the two in sync by hand.
const THEME_CHANGED_EVENT = 'theme:changed'

// Subscribes to the Go backend's live theme-change notification (currently
// script-manager-gui only — it watches sm-theme.json for writes made by
// sm-config-edit's Theme section) via eventsOn, each app's own bound
// wailsjs/runtime EventsOn — passed in rather than imported directly
// since, like GetTheme in syncTheme, the two apps' generated bindings live
// in different files even though the API shape is identical. Applies and
// persists the change locally the moment it arrives, then reports the new
// theme/themes to onChange so the caller can update its own reactive UI.
// Returns the unsubscribe function EventsOn itself returns.
export function watchTheme(
  eventsOn: (eventName: string, callback: (...data: unknown[]) => void) => () => void,
  onChange: (theme: Theme, themes: Record<string, CustomPalette> | null) => void,
): () => void {
  return eventsOn(THEME_CHANGED_EVENT, (...data: unknown[]) => {
    const { theme, themes } = applyRemoteState(data[0] as ThemeState)
    onChange(theme, themes)
  })
}
