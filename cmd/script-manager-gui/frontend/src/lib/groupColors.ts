import type { gui } from '../../wailsjs/go/models'

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

export function groupChipStyle(colors: Record<string, string>, group: string, active: boolean): string {
  const color = colors[group]
  if (active || !color) return ''
  return `background: ${color}; border-color: ${color}; color: ${readableTextColor(color)};`
}
