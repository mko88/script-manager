import { t } from '../messages'

export function deepCopy<T>(src: T): T {
  return JSON.parse(JSON.stringify(src)) as T
}

function unique(base: string, suffix: string, sep: string, taken: string[]): string {
  const wanted = `${base}${suffix}`
  if (!taken.includes(wanted)) return wanted
  for (let n = 2; ; n++) {
    const candidate = `${wanted}${sep}${n}`
    if (!taken.includes(candidate)) return candidate
  }
}

export function copyLabel(base: string, taken: string[]): string {
  if (!base) return ''
  return unique(base, t('text.copySuffix'), ' ', taken)
}

export function copyId(base: string, taken: string[]): string {
  if (!base) return ''
  return unique(base, t('text.copyIdSuffix'), '-', taken)
}

export function insertAfter<T>(list: T[], i: number, dup: T): T[] {
  return [...list.slice(0, i + 1), dup, ...list.slice(i + 1)]
}
