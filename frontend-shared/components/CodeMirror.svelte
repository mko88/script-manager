<script lang="ts" context="module">
  import { EditorView } from '@codemirror/view'
  import { HighlightStyle, StreamLanguage, syntaxHighlighting } from '@codemirror/language'
  import { shell } from '@codemirror/legacy-modes/mode/shell'
  import { powerShell } from '@codemirror/legacy-modes/mode/powershell'
  import { yaml } from '@codemirror/legacy-modes/mode/yaml'
  import { tags } from '@lezer/highlight'

  export type CodeLanguage = 'shell' | 'powershell' | 'yaml' | 'markdown' | 'plain' | string

  // Everything is a theme token, so a CodeMirror pane retints with the rest
  // of the app the moment a theme is switched or edited — no second palette
  // to keep in sync.
  export const smTheme = EditorView.theme({
    '&': {
      color: 'var(--sm-text)',
      backgroundColor: 'var(--sm-bg-deep)',
      fontSize: 'var(--sm-type-base)',
      borderRadius: '4px',
    },
    '.cm-content': {
      fontFamily: 'var(--sm-font-mono)',
      caretColor: 'var(--sm-text-heading)',
    },
    '.cm-cursor, .cm-dropCursor': { borderLeftColor: 'var(--sm-text-heading)' },
    '&.cm-focused .cm-selectionBackground, .cm-selectionBackground, ::selection': {
      backgroundColor: 'var(--sm-tint-hover)',
    },
    '.cm-gutters': {
      backgroundColor: 'var(--sm-bg-deep)',
      color: 'var(--sm-line-number)',
      border: 'none',
      paddingLeft: '4px',
      fontFamily: 'var(--sm-font-mono)',
    },
    '.cm-activeLine': { backgroundColor: 'var(--sm-overlay-soft)' },
    '.cm-activeLineGutter': { backgroundColor: 'transparent', color: 'var(--sm-text-muted)' },
    '.cm-foldPlaceholder': {
      backgroundColor: 'var(--sm-bg-secondary)',
      border: '1px solid var(--sm-border)',
      color: 'var(--sm-text-muted)',
    },
    '.cm-scroller': { scrollbarWidth: 'thin', scrollbarColor: 'var(--sm-scrollbar) transparent' },
    '.cm-scroller::-webkit-scrollbar': { width: '5px', height: '5px' },
    '.cm-scroller::-webkit-scrollbar-thumb': { backgroundColor: 'var(--sm-scrollbar)', borderRadius: '3px' },
    '&.cm-focused': { outline: 'none' },
    '.cm-placeholder': { color: 'var(--sm-text-faint)' },
    '.cm-tooltip': {
      backgroundColor: 'var(--sm-panel-header)',
      border: '1px solid var(--sm-border)',
      borderRadius: '4px',
      color: 'var(--sm-text)',
    },
    '.cm-tooltip-autocomplete > ul > li': {
      fontFamily: 'var(--sm-font-mono)',
      padding: '2px 6px',
    },
    '.cm-tooltip-autocomplete > ul > li[aria-selected]': {
      backgroundColor: 'var(--sm-bg-primary)',
      color: 'var(--sm-text-primary)',
    },
    '.cm-completionDetail': { color: 'var(--sm-text-muted)', fontStyle: 'normal', marginLeft: '8px' },
  })

  export const smHighlight = HighlightStyle.define([
    { tag: [tags.keyword, tags.definitionKeyword, tags.controlKeyword], color: 'var(--sm-text-tab)' },
    { tag: [tags.string, tags.special(tags.string)], color: 'var(--sm-run-ok)' },
    { tag: [tags.number, tags.bool, tags.atom], color: 'var(--sm-text-highlight)' },
    { tag: [tags.propertyName, tags.attributeName, tags.tagName], color: 'var(--sm-text-heading)' },
    { tag: [tags.variableName, tags.labelName], color: 'var(--sm-text)' },
    { tag: [tags.comment, tags.meta], color: 'var(--sm-text-faint)', fontStyle: 'italic' },
    { tag: [tags.operator, tags.punctuation, tags.bracket], color: 'var(--sm-text-muted)' },
    { tag: tags.invalid, color: 'var(--sm-error)' },
  ])

  export function languageExtension(language: string) {
    switch (language) {
      case 'shell':
        return [StreamLanguage.define(shell)]
      case 'powershell':
        return [StreamLanguage.define(powerShell)]
      case 'yaml':
        return [StreamLanguage.define(yaml)]
      default:
        return []
    }
  }

  export const smSyntax = syntaxHighlighting(smHighlight)
</script>

<script lang="ts">
  import { onDestroy, onMount } from 'svelte'
  import { Compartment, EditorState } from '@codemirror/state'
  import { keymap, lineNumbers, placeholder as placeholderExt } from '@codemirror/view'
  import { defaultKeymap, history, historyKeymap, indentWithTab } from '@codemirror/commands'
  import { autocompletion, completionKeymap, type CompletionContext } from '@codemirror/autocomplete'
  import { bracketMatching, foldGutter, foldKeymap, indentOnInput } from '@codemirror/language'

  export let value = ''
  export let language: string = 'plain'
  export let readOnly = false
  export let wrap = true
  export let showLineNumbers = true
  export let placeholder = ''
  export let minHeight = ''
  export let maxHeight = ''
  // Keeps a streaming document pinned to its last line — the inline run
  // output, which grows while it is watched.
  export let followTail = false
  // Called after an edit made in this editor, for callers that validate or
  // react to the new text. A callback rather than a dispatched event: these
  // components are consumed as plain props everywhere else.
  export let onChange: ((value: string) => void) | null = null
  // Names offered while typing inside a {{ }} reference. Empty turns
  // completion off entirely.
  export let completions: { label: string; apply?: string; applyStandalone?: string; detail?: string }[] = []

  let host: HTMLElement
  let view: EditorView | undefined

  // While typing, only inside an unclosed {{ — the rest of a details
  // template is prose, where a popup on every word would be in the way.
  // Asked for explicitly (Ctrl+Space) it answers anywhere, and inserts the
  // standalone form, which is the whole reference rather than its middle.
  function completeTemplateRef(ctx: CompletionContext) {
    if (completions.length === 0) return null
    const insideRef = ctx.matchBefore(/\{\{[^}]*$/) !== null
    if (!insideRef && !ctx.explicit) return null
    const token = ctx.matchBefore(/[\w.]*$/)
    return {
      from: token ? token.from : ctx.pos,
      options: completions.map((c) => ({
        label: c.label,
        apply: (insideRef ? c.apply : c.applyStandalone ?? c.apply) ?? c.label,
        detail: c.detail,
        type: 'variable',
      })),
      validFor: /^[\w.]*$/,
    }
  }
  const languageCompartment = new Compartment()

  onMount(() => {
    const extensions = [
      languageCompartment.of(languageExtension(language)),
      smTheme,
      smSyntax,
      EditorView.editable.of(!readOnly),
      EditorState.readOnly.of(readOnly),
      bracketMatching(),
      EditorView.updateListener.of((u) => {
        if (u.docChanged && !readOnly) {
          value = u.state.doc.toString()
          onChange?.(value)
        }
      }),
    ]
    if (showLineNumbers) extensions.push(lineNumbers(), foldGutter())
    if (wrap) extensions.push(EditorView.lineWrapping)
    if (placeholder) extensions.push(placeholderExt(placeholder))
    if (!readOnly) {
      extensions.push(history(), indentOnInput())
      if (completions.length > 0) {
        extensions.push(autocompletion({ override: [completeTemplateRef] }))
      }
      extensions.push(
        keymap.of([...completionKeymap, ...defaultKeymap, ...historyKeymap, ...foldKeymap, indentWithTab]),
      )
    }

    view = new EditorView({ doc: value, parent: host, extensions })
  })

  onDestroy(() => view?.destroy())

  // An external change replaces the document; a change typed in here is
  // already in it, so comparing first keeps the cursor where it was.
  //
  // Both dispatches are guarded: they run inside Svelte's update cycle, so
  // an exception here would take the whole render loop down with it and
  // leave a window that paints hover states but never updates again.
  $: if (view && value !== view.state.doc.toString()) {
    try {
      // No selection in this transaction: CodeMirror normalizes CRLF to LF
      // as it inserts, so the new document is shorter than the string that
      // produced it and any position derived from that string lands outside
      // it. The follow-up scroll reads the length the document actually has.
      view.dispatch({ changes: { from: 0, to: view.state.doc.length, insert: value } })
      if (followTail) {
        view.dispatch({ effects: EditorView.scrollIntoView(view.state.doc.length) })
      }
    } catch (err) {
      console.error('CodeMirror: setting the document failed', err)
    }
  }

  $: if (view) {
    try {
      view.dispatch({ effects: languageCompartment.reconfigure(languageExtension(language)) })
    } catch (err) {
      console.error('CodeMirror: switching language failed', err)
    }
  }

  // Leaves the inserted text selected, so an inserted variable can be
  // formatted or replaced straight away without reaching for the mouse.
  export function insertAtCursor(text: string) {
    if (!view) return
    const { from, to } = view.state.selection.main
    view.dispatch({
      changes: { from, to, insert: text },
      selection: { anchor: from, head: from + text.length },
    })
    view.focus()
  }

  // Wraps the selection, or unwraps it when the markers are already there,
  // so the same button toggles bold/italic/code the way it did before.
  export function wrapSelection(before: string, after: string = before) {
    if (!view) return
    const { from, to } = view.state.selection.main
    const selected = view.state.sliceDoc(from, to)
    const leading = view.state.sliceDoc(Math.max(0, from - before.length), from)
    const trailing = view.state.sliceDoc(to, Math.min(view.state.doc.length, to + after.length))

    if (before.length > 0 && leading === before && trailing === after) {
      const start = from - before.length
      view.dispatch({
        changes: { from: start, to: to + after.length, insert: selected },
        selection: { anchor: start, head: start + selected.length },
      })
      view.focus()
      return
    }

    view.dispatch({
      changes: { from, to, insert: before + selected + after },
      selection: { anchor: from + before.length, head: from + before.length + selected.length },
    })
    view.focus()
  }

  export function selectedText(): string {
    if (!view) return ''
    const { from, to } = view.state.selection.main
    return view.state.sliceDoc(from, to)
  }

  // Also leaves the replacement selected — the mask button replaces a
  // variable reference, and seeing what it produced is the point.
  export function replaceSelection(text: string) {
    if (!view) return
    const { from, to } = view.state.selection.main
    view.dispatch({
      changes: { from, to, insert: text },
      selection: { anchor: from, head: from + text.length },
    })
    view.focus()
  }
</script>

<div
  class="sm-code"
  class:sm-code-readonly={readOnly}
  style:--sm-code-min-height={minHeight}
  style:--sm-code-max-height={maxHeight}
  bind:this={host}
></div>

<style>
  .sm-code {
    min-width: 0;
    border-radius: 4px;
    overflow: hidden;
  }

  .sm-code :global(.cm-editor) {
    min-height: var(--sm-code-min-height, auto);
    max-height: var(--sm-code-max-height, none);
  }

  .sm-code :global(.cm-scroller) {
    overflow: auto;
  }
</style>
