export type Theme = string
export type CustomPalette = Record<string, string>
export interface ThemeState {
  active: string
  themes?: Record<string, CustomPalette>
}

const STORAGE_KEY = 'sm-theme'
const THEMES_STORAGE_KEY = 'sm-theme-themes'

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
      'text-tab',
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
    tokens: [
      'secondary-hover',
      'tint-hover',
      'overlay-soft',
      'scrollbar',
      'shadow',
      'primary-hover',
      'warning-tint',
      'scrim',
    ],
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
    root.setAttribute('data-theme', 'dark')
    for (const name of TOKEN_NAMES) {
      const value = palette[name]
      if (value) root.style.setProperty(`--sm-${name}`, value)
      else root.style.removeProperty(`--sm-${name}`)
    }
  } else {
    root.setAttribute('data-theme', theme === 'light' ? 'light' : 'dark')
    for (const name of TOKEN_NAMES) root.style.removeProperty(`--sm-${name}`)
  }
}

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

export async function syncTheme(
  getRemote: () => Promise<ThemeState>,
): Promise<{ theme: Theme; themes: Record<string, CustomPalette> | null }> {
  try {
    return applyRemoteState(await getRemote())
  } catch {
    return { theme: getTheme(), themes: getThemes() }
  }
}

const THEME_CHANGED_EVENT = 'theme:changed'

export function watchTheme(
  eventsOn: (eventName: string, callback: (...data: unknown[]) => void) => () => void,
  onChange: (theme: Theme, themes: Record<string, CustomPalette> | null) => void,
): () => void {
  return eventsOn(THEME_CHANGED_EVENT, (...data: unknown[]) => {
    const { theme, themes } = applyRemoteState(data[0] as ThemeState)
    onChange(theme, themes)
  })
}
