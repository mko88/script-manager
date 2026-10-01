import { StateEffect, StateField } from '@codemirror/state'
import { Decoration, EditorView } from '@codemirror/view'
import type { DecorationSet } from '@codemirror/view'

// One styled run of command output, as the backend parsed it out of the
// escape sequences. Offsets are positions in the text it came with.
export type AnsiSpan = {
  from: number
  to: number
  fg?: string
  bg?: string
  bold?: boolean
  dim?: boolean
  italic?: boolean
  underline?: boolean
  strike?: boolean
}

// 'fg' and 'bg' are what inverse video leaves when a colour was the default.
function cssColour(value: string): string {
  if (value === 'fg') return 'var(--sm-text)'
  if (value === 'bg') return 'var(--sm-bg-deep)'
  if (value.startsWith('ansi-')) return `var(--sm-${value})`
  return value
}

function spanStyle(span: AnsiSpan): string {
  const rules: string[] = []
  if (span.fg) rules.push(`color: ${cssColour(span.fg)}`)
  if (span.bg) rules.push(`background-color: ${cssColour(span.bg)}`)
  if (span.bold) rules.push('font-weight: 700')
  if (span.dim) rules.push('opacity: 0.6')
  if (span.italic) rules.push('font-style: italic')
  const lines = [span.underline && 'underline', span.strike && 'line-through'].filter(Boolean)
  if (lines.length > 0) rules.push(`text-decoration: ${lines.join(' ')}`)
  return rules.join('; ')
}

// A range outside the document makes CodeMirror throw while drawing, so
// spans are fitted to the document they are about to decorate.
export function ansiDecorations(spans: AnsiSpan[], docLength: number): DecorationSet {
  const ranges = []
  for (const span of spans) {
    const from = Math.max(0, span.from)
    const to = Math.min(span.to, docLength)
    if (from >= to) continue
    const style = spanStyle(span)
    if (style) ranges.push(Decoration.mark({ attributes: { style } }).range(from, to))
  }
  return Decoration.set(ranges, true)
}

export const setAnsiSpans = StateEffect.define<DecorationSet>()

export const ansiSpanField = StateField.define<DecorationSet>({
  create: () => Decoration.none,
  update(decorations, tr) {
    for (const effect of tr.effects) {
      if (effect.is(setAnsiSpans)) return effect.value
    }
    return decorations.map(tr.changes)
  },
  provide: (field) => EditorView.decorations.from(field),
})
