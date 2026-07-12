// Chip coloring for action groups, shared by the group-filter chips and the
// Command pane's group tags.
import type { gui } from '../../wailsjs/go/models'

// Only groups with a configured color get an entry — a group with no catalog
// entry (or an entry with no color set) just keeps the default chip styling,
// so this feature is fully opt-in per group.
export function buildGroupColors(catalog: gui.ActionGroupDTO[]): Record<string, string> {
  return Object.fromEntries(
    catalog.filter((g) => /^#[0-9a-fA-F]{6}$/.test(g.color)).map((g) => [g.id, g.color]),
  ) as Record<string, string>
}

function readableTextColor(hex: string): string {
  const r = parseInt(hex.slice(1, 3), 16)
  const g = parseInt(hex.slice(3, 5), 16)
  const b = parseInt(hex.slice(5, 7), 16)
  const brightness = (r * 299 + g * 587 + b * 114) / 1000
  return brightness > 128 ? '#1b2636' : '#d7dee8'
}

// Active/selected chips keep the existing accent-warm highlight regardless
// of the group's own color, so "this filter is active" stays unambiguous.
export function groupChipStyle(colors: Record<string, string>, group: string, active: boolean): string {
  const color = colors[group]
  if (active || !color) return ''
  return `background: ${color}; border-color: ${color}; color: ${readableTextColor(color)};`
}
