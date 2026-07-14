<script lang="ts">
  import { onMount } from 'svelte'
  import { loadPersisted, savePersisted } from '@shared/persist'
  import { flash } from '@shared/toast'
  import Icon from '@shared/components/Icon.svelte'
  import IconButton from '@shared/components/IconButton.svelte'
  import { t } from '../messages'
  import { looksLikeSecretKey } from '../secretKey'
  import type { configedit } from '../../wailsjs/go/models'

  // The Displays section: pick a display via combobox (no master-list
  // sidebar — a display isn't tied to one item the way an item's own
  // preview is), edit its list/details templates, and preview any item
  // against it, with a layout toggle for how much space editing vs.
  // previewing gets.

  // Two-way bound slices of the parent's cfg.
  export let displays: configedit.DisplayDTO[]
  export let selectedDisplay: number
  // Read-only context for the preview: the config's items (preview-item
  // picker) and env fields (template expansion).
  export let items: configedit.ItemDTO[] = []
  export let envFields: configedit.FieldDTO[] = []
  // The actual Wails binding, passed straight through like FieldGrid's
  // validateField prop — this component doesn't import bindings itself.
  export let previewItem: (
    item: configedit.ItemDTO,
    envFields: configedit.FieldDTO[],
    displays: configedit.DisplayDTO[],
    displayName: string,
  ) => Promise<configedit.PreviewDTO>

  type DisplayViewMode = 'edit' | 'preview' | 'split-v' | 'split-h'
  let previewItemForDisplay = -1
  let displayViewMode: DisplayViewMode = 'split-v'
  let displayPreview: configedit.PreviewDTO | null = null

  // Split-v sizes the edit pane by width, split-h by height; the other pane
  // always gets flex:1 to soak up whatever's left, so together they fill
  // the full available width (split-v) or height (split-h) — same
  // fixed-primary-pane-plus-flex:1-remainder pattern script-manager-gui's
  // own resizer uses (leftWidth/itemsHeight there). Minimums match gui's
  // MIN_COL/MIN_PANEL exactly — same reasoning: a width split needs more
  // headroom for usable content than a height split does.
  const DISPLAY_MIN_WIDTH = 180
  const DISPLAY_MIN_HEIGHT = 60
  const DISPLAY_RESIZER = 6
  let displayEditWidth = 480
  let displayEditHeight = 260
  let displaySplitEl: HTMLElement

  // The Details template's helper toolbar (Insert env / formatting buttons)
  // needs a real element handle for cursor/selection-based edits —
  // setRangeText operates on the DOM element directly, not on Svelte's
  // bound value.
  let detailsTextareaEl: HTMLTextAreaElement | undefined

  // Env var names available to insert into the Details template: global
  // Environment fields plus the currently-selected preview item's own
  // fields (if any), deduped — both already loaded client-side for the
  // preview feature, so no new backend call is needed just to list them.
  $: availableEnvKeys = Array.from(
    new Set([
      ...envFields.map((f) => f.key),
      ...(previewItemForDisplay >= 0 ? (items[previewItemForDisplay]?.fields ?? []).map((f) => f.key) : []),
    ]),
  ).filter((k) => k)

  function insertEnvVar(key: string) {
    // A key that looks like it holds a secret (same heuristic FieldGrid uses
    // to auto-lock a new field) is inserted already masked, not as a plain
    // reference someone would otherwise have to remember to wrap themselves.
    if (looksLikeSecretKey(key)) insertAtCursor('`{{mask .' + key + '}}`')
    else insertAtCursor(`{{.${key}}}`)
  }

  function onEnvSelectChange(e: Event) {
    const select = e.currentTarget as HTMLSelectElement
    const key = select.value
    if (key) insertEnvVar(key)
    select.selectedIndex = 0
  }

  // Replaces [start, end) in the textarea with text via execCommand, which —
  // unlike setRangeText — participates in the browser's native undo/redo, so
  // Ctrl+Z/Ctrl+Y still work after a helper-button edit, the same as they
  // would after ordinary typing. execCommand acts on the element's current
  // selection, so it has to be focused with that range selected first.
  // Falls back to setRangeText (functionally identical, just without undo
  // support) if execCommand is ever unavailable.
  function replaceRange(el: HTMLTextAreaElement, start: number, end: number, text: string) {
    el.focus()
    el.setSelectionRange(start, end)
    let handled = false
    try {
      handled = document.execCommand('insertText', false, text)
    } catch {
      handled = false
    }
    if (!handled) el.setRangeText(text, start, end, 'end')
    // Either path mutates the element directly without firing an input event
    // Svelte's bind:value would otherwise pick up, so the bound state has to
    // be synced back explicitly.
    displays[selectedDisplay].details = el.value
  }

  function insertAtCursor(text: string) {
    const el = detailsTextareaEl
    if (!el) return
    const start = el.selectionStart ?? el.value.length
    const end = el.selectionEnd ?? el.value.length
    replaceRange(el, start, end, text)
    // Highlight the just-inserted text, so it's clear what was inserted and
    // easy to overtype/replace.
    el.setSelectionRange(start, start + text.length)
  }

  function wrapSelection(before: string, after: string = before) {
    const el = detailsTextareaEl
    if (!el) return
    const start = el.selectionStart ?? 0
    const end = el.selectionEnd ?? 0
    const selected = el.value.slice(start, end)

    // Toggling the same button again on text it already wrapped removes the
    // markers instead of stacking another layer around them (e.g. **hello**
    // -> hello, not ****hello****).
    const alreadyWrapped =
      before.length > 0 &&
      el.value.slice(start - before.length, start) === before &&
      el.value.slice(end, end + after.length) === after

    if (alreadyWrapped) {
      const wrapStart = start - before.length
      replaceRange(el, wrapStart, end + after.length, selected)
      el.setSelectionRange(wrapStart, wrapStart + selected.length)
      return
    }

    replaceRange(el, start, end, before + selected + after)
    // Re-select just the original text (not the before/after markers), so
    // it stays visibly highlighted and a second formatting button or typed
    // replacement acts on the content, not the markup around it.
    el.setSelectionRange(start + before.length, start + before.length + selected.length)
  }

  // Matches a bare `{{.field}}`/`{{.field.nested}}` reference — the only
  // shape `mask` can meaningfully wrap; wrapping arbitrary literal text in
  // `{{mask ...}}` would just produce an invalid template. A `{{mask ...}}`
  // reference itself doesn't match (it doesn't start with a bare dot), so
  // re-selecting an already-masked span and clicking Mask again is already a
  // harmless no-op rather than double-masking it.
  const FIELD_REF_RE = /^\{\{\s*(\.[\w.]+)\s*\}\}$/

  // Turns a selected `{{.field}}` reference into a masked `` `{{mask .field}}` ``
  // one — the same transform insertEnvVar applies automatically for a
  // secret-looking key, but usable on a variable already in the template
  // (e.g. one written by hand, or inserted before its key was recognized).
  // No-ops with a flash if the selection isn't a bare field reference.
  function maskSelection() {
    const el = detailsTextareaEl
    if (!el) return
    const start = el.selectionStart ?? 0
    const end = el.selectionEnd ?? 0
    const match = el.value.slice(start, end).match(FIELD_REF_RE)
    if (!match) {
      flash(t('toast.maskNeedsVariable'))
      return
    }
    const replacement = '`{{mask ' + match[1] + '}}`'
    replaceRange(el, start, end, replacement)
    el.setSelectionRange(start, start + replacement.length)
  }

  const DISPLAY_LAYOUT_KEY = 'sm-config-edit:displayLayout'

  onMount(() => {
    ;({ viewMode: displayViewMode, editWidth: displayEditWidth, editHeight: displayEditHeight } = loadPersisted(
      DISPLAY_LAYOUT_KEY,
      { viewMode: displayViewMode, editWidth: displayEditWidth, editHeight: displayEditHeight },
    ))
  })

  function saveDisplayLayout() {
    savePersisted(DISPLAY_LAYOUT_KEY, {
      viewMode: displayViewMode,
      editWidth: displayEditWidth,
      editHeight: displayEditHeight,
    })
  }
  function setDisplayViewMode(mode: DisplayViewMode) {
    displayViewMode = mode
    saveDisplayLayout()
  }

  function dragDisplaySplit(e: MouseEvent) {
    e.preventDefault()
    const horizontal = displayViewMode === 'split-h'
    const min = horizontal ? DISPLAY_MIN_HEIGHT : DISPLAY_MIN_WIDTH
    const startPos = horizontal ? e.clientY : e.clientX
    const startSize = horizontal ? displayEditHeight : displayEditWidth
    function onMove(ev: MouseEvent) {
      const rect = displaySplitEl.getBoundingClientRect()
      const total = horizontal ? rect.height : rect.width
      const max = total - min - DISPLAY_RESIZER
      const pos = horizontal ? ev.clientY : ev.clientX
      const next = Math.min(max, Math.max(min, startSize + (pos - startPos)))
      if (horizontal) displayEditHeight = next
      else displayEditWidth = next
    }
    function onUp() {
      window.removeEventListener('mousemove', onMove)
      window.removeEventListener('mouseup', onUp)
      saveDisplayLayout()
    }
    window.addEventListener('mousemove', onMove)
    window.addEventListener('mouseup', onUp)
  }

  function newDisplay(): configedit.DisplayDTO {
    return { name: '', list: '{{.name}}', details: '' } as unknown as configedit.DisplayDTO
  }

  function addDisplay() {
    displays = [...displays, newDisplay()]
    selectedDisplay = displays.length - 1
  }
  function copyDisplay() {
    const src = displays[selectedDisplay]
    if (!src) return
    displays = [...displays, { ...src, name: `${src.name} - copy` }]
    selectedDisplay = displays.length - 1
  }
  function removeDisplay(i: number) {
    displays = displays.filter((_, idx) => idx !== i)
    if (selectedDisplay === i) selectedDisplay = -1
    else if (selectedDisplay > i) selectedDisplay -= 1
  }
  function confirmRemoveDisplay(i: number) {
    const name = displays[i]?.name || t('fallback.unnamed')
    if (confirm(t('confirm.removeDisplay', { name }))) removeDisplay(i)
  }

  let displayPreviewTimer: ReturnType<typeof setTimeout>
  // previewItemForDisplay >= -1 is always true — it's just there so Svelte
  // tracks it as a dependency of this statement (picking a different
  // preview item must re-trigger this the same way editing the template
  // does), not a real condition.
  $: if (selectedDisplay >= 0 && displays[selectedDisplay] && previewItemForDisplay >= -1) scheduleDisplayPreview()
  function scheduleDisplayPreview() {
    clearTimeout(displayPreviewTimer)
    displayPreviewTimer = setTimeout(async () => {
      const d = displays[selectedDisplay]
      const item = items[previewItemForDisplay]
      if (!d || !item) {
        displayPreview = null
        return
      }
      displayPreview = await previewItem(item, envFields, displays, d.name)
    }, 250)
  }
</script>

<div class="display-section">
  <div class="list-toolbar">
    <IconButton title={t('tooltip.addDisplay')} on:click={addDisplay}><Icon name="add" /></IconButton>
    <IconButton title={t('tooltip.copyDisplay')} disabled={selectedDisplay < 0} on:click={copyDisplay}><Icon name="copy" /></IconButton>
    <IconButton
      title={t('tooltip.removeDisplay')}
      disabled={selectedDisplay < 0}
      on:click={() => confirmRemoveDisplay(selectedDisplay)}><Icon name="remove" /></IconButton
    >
  </div>

  <div class="display-select-row">
    <label class="field display-select-field">
      <span>{t('field.display')}</span>
      <select bind:value={selectedDisplay}>
        <option value={-1}>{t('option.selectDisplay')}</option>
        {#each displays as d, i (i)}<option value={i}>{d.name || t('option.unnamedDisplay', { n: i + 1 })}</option
          >{/each}
      </select>
    </label>
    {#if selectedDisplay >= 0 && displays[selectedDisplay]}
      <label class="field display-name-field">
        <span>{t('field.name')}</span>
        <input type="text" bind:value={displays[selectedDisplay].name} />
      </label>
    {/if}
  </div>

  {#if selectedDisplay >= 0 && displays[selectedDisplay]}
    <div class="display-toolbar">
      <div class="view-mode-group">
        <IconButton
          active={displayViewMode === 'edit'}
          title={t('tooltip.editOnly')}
          on:click={() => setDisplayViewMode('edit')}><Icon name="edit" /></IconButton
        >
        <IconButton
          active={displayViewMode === 'preview'}
          title={t('tooltip.previewOnly')}
          on:click={() => setDisplayViewMode('preview')}><Icon name="preview" /></IconButton
        >
        <IconButton
          active={displayViewMode === 'split-v'}
          title={t('tooltip.sideBySide')}
          on:click={() => setDisplayViewMode('split-v')}><Icon name="split-v" /></IconButton
        >
        <IconButton
          active={displayViewMode === 'split-h'}
          title={t('tooltip.stacked')}
          on:click={() => setDisplayViewMode('split-h')}><Icon name="split-h" /></IconButton
        >
      </div>
      <label class="field preview-item-picker">
        <span>{t('field.previewItem')}</span>
        <select bind:value={previewItemForDisplay} on:change={scheduleDisplayPreview}>
          <option value={-1}>{t('option.none')}</option>
          {#each items as it, i}<option value={i}>{it.name || t('option.unnamedItem', { n: i + 1 })}</option
            >{/each}
        </select>
      </label>
    </div>

    <div
      class="display-edit-preview"
      class:split-v={displayViewMode === 'split-v'}
      class:split-h={displayViewMode === 'split-h'}
      bind:this={displaySplitEl}
    >
      {#if displayViewMode !== 'preview'}
        <div
          class="edit-pane panel"
          style={displayViewMode === 'split-v'
            ? `flex: 0 1 ${displayEditWidth}px`
            : displayViewMode === 'split-h'
              ? `flex: 0 1 ${displayEditHeight}px`
              : ''}
        >
          <header class="panel-title"><span>{t('panel.edit')}</span></header>
          <div class="panel-body edit-pane-body">
            <label class="field list-template-field">
              <span>{t('field.listTemplate')}</span>
              <input type="text" bind:value={displays[selectedDisplay].list} />
            </label>
            <label class="field details-template-field">
              <span>{t('field.detailsTemplate')}</span>
              <div class="details-helper-toolbar">
                <select class="env-insert-select" title={t('tooltip.insertEnvVar')} on:change={onEnvSelectChange}>
                  <option value="">{t('option.insertEnv')}</option>
                  {#each availableEnvKeys as key (key)}<option value={key}>{key}</option>{/each}
                </select>
                <IconButton title={t('tooltip.bold')} on:click={() => wrapSelection('**')}><strong>B</strong></IconButton>
                <IconButton title={t('tooltip.italic')} on:click={() => wrapSelection('_')}><em>I</em></IconButton>
                <IconButton title={t('tooltip.highlight')} on:click={() => wrapSelection('`')}><code>`</code></IconButton>
                <IconButton title={t('tooltip.mask')} on:click={maskSelection}
                  ><svg viewBox="0 0 16 16" width="13" height="13" aria-hidden="true">
                    <rect x="3.5" y="7" width="9" height="6.5" rx="1.2" fill="none" stroke="currentColor" stroke-width="1.3" />
                    <path d="M5 7V5a3 3 0 0 1 6 0v2" fill="none" stroke="currentColor" stroke-width="1.3" />
                  </svg></IconButton
                >
              </div>
              <textarea bind:value={displays[selectedDisplay].details} bind:this={detailsTextareaEl}
              ></textarea>
            </label>
          </div>
        </div>
      {/if}
      {#if displayViewMode === 'split-v' || displayViewMode === 'split-h'}
        <!-- svelte-ignore a11y-no-static-element-interactions -->
        <div
          class="resizer {displayViewMode === 'split-h' ? 'horizontal' : 'vertical'}"
          on:mousedown={dragDisplaySplit}
        ></div>
      {/if}
      {#if displayViewMode !== 'edit'}
        <div class="preview-pane-inline panel">
          <header class="panel-title"><span>{t('panel.preview')}</span></header>
          <div class="panel-body">
            {#if previewItemForDisplay < 0}
              <div class="empty">{t('empty.pickItemToPreview')}</div>
            {:else if displayPreview}
              {#if displayPreview.error}
                <div class="validation-issue validation-error">{displayPreview.error}</div>
              {/if}
              <p class="preview-label">{t('hint.listLabel')}<strong>{displayPreview.listLabel}</strong></p>
              {#if displayPreview.missingFields?.length}
                <p class="hint">{t('hint.missingFields', { fields: displayPreview.missingFields.join(', ') })}</p>
              {/if}
              <div class="details-preview">{@html displayPreview.detailsHtml}</div>
            {/if}
          </div>
        </div>
      {/if}
    </div>
  {:else}
    <div class="empty">{t('empty.selectDisplayOrAdd')}</div>
  {/if}
</div>

<style>
  /* Displays has no master-list sidebar (a combobox picks the display
     instead), so it doesn't use .master-detail/.detail at all. Its
     edit/preview split still needs a real, bounded height to resize
     within, so display-section fills the available height exactly and
     lets display-edit-preview's two panes scroll internally instead. */
  .display-section {
    display: flex;
    flex-direction: column;
    gap: 10px;
    height: 100%;
    min-height: 0;
    overflow: hidden;
  }

  /* .list-toolbar's own margin-bottom (global, shared with Items/Action
     Groups/Actions) would stack with this column's gap above — flex gap
     already spaces it consistently with every other child here. */
  .display-section > .list-toolbar {
    margin-bottom: 0;
  }

  .display-select-row {
    flex: none;
    display: flex;
    align-items: flex-end;
    gap: 12px;
  }

  .display-select-field {
    flex: 0 1 320px;
    margin-bottom: 0;
  }

  .display-name-field {
    flex: 1 1 200px;
    min-width: 0;
    margin-bottom: 0;
  }

  .display-toolbar {
    display: flex;
    align-items: flex-end;
    justify-content: space-between;
    gap: 12px;
    flex-wrap: wrap;
  }

  .view-mode-group {
    display: flex;
    gap: 4px;
  }

  /* .btn.active partially :global — that class now renders inside
     IconButton's own template, which Svelte's per-component CSS scoping
     wouldn't otherwise reach; .view-mode-group itself stays scoped since
     it's still this component's own element. */
  .view-mode-group :global(.btn.active) {
    background: var(--sm-bg-primary);
    border-color: var(--sm-bg-primary);
    color: var(--sm-text-primary);
    font-weight: 700;
  }

  .preview-item-picker {
    flex: 0 0 220px;
    margin-bottom: 0;
  }

  /* Row by default (also covers single-pane Edit-only/Preview-only modes,
     where whichever one pane is present just fills 100% via flex:1 below);
     .split-h switches to a column so the panes stack instead. */
  .display-edit-preview {
    display: flex;
    flex: 1 1 auto;
    min-height: 0;
    overflow: hidden;
  }

  .display-edit-preview.split-h {
    flex-direction: column;
  }

  /* Base flex-basis is 0, not auto: with auto, an unconstrained pane's basis
     is its max-content size — for the Preview pane that means the table's
     *unwrapped* natural width, which can be huge regardless of the
     word-wrap CSS below (max-content sizing ignores wrapping opportunities
     by definition). That huge implicit basis was swamping the edit pane's
     explicit pixel basis during flex-shrink, collapsing it to ~2px even
     though its own flex-basis said otherwise. flex-basis:0 makes both
     panes' share of space depend only on flex-grow/shrink and the explicit
     pixel size below, never on content. A single visible pane (Edit-only/
     Preview-only) still fills 100% via flex-grow regardless of basis. */
  .edit-pane,
  .preview-pane-inline {
    flex: 1 1 0;
    min-width: 0;
    min-height: 0;
  }

  .edit-pane-body {
    display: flex;
    flex-direction: column;
    gap: 8px;
    min-height: 0;
  }

  .list-template-field {
    flex: none;
  }

  .details-template-field {
    flex: 1 1 auto;
    min-height: 0;
    margin-bottom: 0;
  }

  .details-template-field textarea {
    flex: 1 1 auto;
    min-height: 60px;
    resize: none;
    font-family: "SF Mono", Consolas, monospace;
    font-size: 0.8rem;
  }

  .details-helper-toolbar {
    flex: none;
    display: flex;
    align-items: center;
    gap: 4px;
    margin-bottom: 4px;
  }

  .details-helper-toolbar :global(.icon-btn) {
    font-family: "SF Mono", Consolas, monospace;
    font-size: 0.8rem;
    line-height: 1;
  }

  .env-insert-select {
    max-width: 160px;
  }
</style>
