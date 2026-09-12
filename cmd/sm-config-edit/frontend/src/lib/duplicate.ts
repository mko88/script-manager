// Duplicating one config entry. Every Copy button in the app routes its
// naming through copyLabel/copyId here, so "<name> - Copy" means the same
// thing in Items, Action Groups, Actions, Displays and Themes alike.
import { t } from '../messages'

// A DTO copy has to be deep: ItemDTO carries four arrays (actions,
// actionGroups, customActions, fields) and ActionDTO/ActionGroupDTO carry
// groups, so a plain spread would leave the duplicate sharing those arrays
// with its original — editing one would silently edit the other. The DTOs
// are pure JSON data, so a round-trip is enough; it also drops the
// Wails-generated class prototype, which is what the editors' own
// `as unknown as configedit.XDTO` object literals already produce.
export function deepCopy<T>(src: T): T {
  return JSON.parse(JSON.stringify(src)) as T
}

// Appends `suffix` to `base`, then keeps appending `sep` + a counter until
// the result isn't already in `taken`. Copying the same entry twice gives
// "X - Copy" then "X - Copy 2" rather than two identical names — which for
// action/action-group ids is not just tidiness: duplicate ids are a
// Save-blocking validation error (see internal/configedit/validate.go).
function unique(base: string, suffix: string, sep: string, taken: string[]): string {
  const wanted = `${base}${suffix}`
  if (!taken.includes(wanted)) return wanted
  for (let n = 2; ; n++) {
    const candidate = `${wanted}${sep}${n}`
    if (!taken.includes(candidate)) return candidate
  }
}

// The human-readable label (an item's name, an action's/group's title).
// An empty one stays empty: a copy of a still-unnamed entry shouldn't be
// christened " - Copy", and the list rows already fall back to "(unnamed)".
export function copyLabel(base: string, taken: string[]): string {
  if (!base) return ''
  return unique(base, t('text.copySuffix'), ' ', taken)
}

// The machine-readable id (an action's or action group's), which items
// reference by exact string — kept in the same token shape as the original
// rather than given the label's spaced suffix. An empty id stays empty, so
// copying a not-yet-identified entry doesn't invent one.
export function copyId(base: string, taken: string[]): string {
  if (!base) return ''
  return unique(base, t('text.copyIdSuffix'), '-', taken)
}

// Splices `dup` in directly below index `i`. Appending to the end is fine
// for Displays' dropdown, but the three master lists are ordered,
// reorder-gated, and can get long — a duplicate that lands next to its
// original doesn't have to be dragged back up.
export function insertAfter<T>(list: T[], i: number, dup: T): T[] {
  return [...list.slice(0, i + 1), dup, ...list.slice(i + 1)]
}
