<script lang="ts">
  import { onMount } from 'svelte'
  import { loadPersisted, savePersisted } from '@shared/persist'
  import { flash } from '@shared/toast'
  import Icon from '@shared/components/Icon.svelte'
  import IconButton from '@shared/components/IconButton.svelte'
  import CodeMirror from '@shared/components/CodeMirror.svelte'
  import { t } from '../messages'
  import { deepCopy, copyLabel } from '../lib/duplicate'
  import { looksLikeSecretKey } from '../secretKey'
  import type { configedit } from '../../wailsjs/go/models'

  export let displays: configedit.DisplayDTO[]
  export let selectedDisplay: number
  export let items: configedit.ItemDTO[] = []
  export let envFields: configedit.FieldDTO[] = []
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

  const DISPLAY_MIN_WIDTH = 180
  const DISPLAY_MIN_HEIGHT = 60
  const DISPLAY_RESIZER = 6
  let displayEditWidth = 480
  let displayEditHeight = 260
  let displaySplitEl: HTMLElement

  let detailsEditor: CodeMirror | undefined

  // name is a reserved item key, held on the item itself rather than among
  // its fields, so it has to be named here — templates use it more than
  // anything else.
  const BUILT_IN_ITEM_KEYS = ['name']

  $: availableEnvKeys = Array.from(
    new Set([
      ...BUILT_IN_ITEM_KEYS,
      ...envFields.map((f) => f.key),
      ...(previewItemForDisplay >= 0 ? (items[previewItemForDisplay]?.fields ?? []).map((f) => f.key) : []),
    ]),
  ).filter((k) => k)

  // The same list the Insert env… dropdown offers, in the form a template
  // wants it. A key that looks like a secret also offers its masked form,
  // which is what the dropdown inserts for one.
  $: templateCompletions = availableEnvKeys.flatMap((key) =>
    looksLikeSecretKey(key)
      ? [
          {
            label: `mask .${key}`,
            apply: `mask .${key}`,
            applyStandalone: '`{{mask .' + key + '}}`',
            detail: 'masked',
          },
          { label: `.${key}`, apply: `.${key}`, applyStandalone: `{{.${key}}}`, detail: 'variable' },
        ]
      : [{ label: `.${key}`, apply: `.${key}`, applyStandalone: `{{.${key}}}`, detail: 'variable' }],
  )

  function insertEnvVar(key: string) {
    if (looksLikeSecretKey(key)) detailsEditor?.insertAtCursor('`{{mask .' + key + '}}`')
    else detailsEditor?.insertAtCursor(`{{.${key}}}`)
  }

  function onEnvSelectChange(e: Event) {
    const select = e.currentTarget as HTMLSelectElement
    const key = select.value
    if (key) insertEnvVar(key)
    select.selectedIndex = 0
  }

  const FIELD_REF_RE = /^\{\{\s*(\.[\w.]+)\s*\}\}$/

  function maskSelection() {
    const match = (detailsEditor?.selectedText() ?? '').match(FIELD_REF_RE)
    if (!match) {
      flash(t('toast.maskNeedsVariable'))
      return
    }
    detailsEditor?.replaceSelection('`{{mask ' + match[1] + '}}`')
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
    const dup = deepCopy(src)
    dup.name = copyLabel(src.name, displays.map((d) => d.name))
    displays = [...displays, dup]
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
            <div class="field details-template-field">
              <span>{t('field.detailsTemplate')}</span>
              <div class="details-helper-toolbar">
                <select class="env-insert-select" title={t('tooltip.insertEnvVar')} on:change={onEnvSelectChange}>
                  <option value="">{t('option.insertEnv')}</option>
                  {#each availableEnvKeys as key (key)}<option value={key}>{key}</option>{/each}
                </select>
                <IconButton title={t('tooltip.bold')} on:click={() => detailsEditor?.wrapSelection('**')}><strong>B</strong></IconButton>
                <IconButton title={t('tooltip.italic')} on:click={() => detailsEditor?.wrapSelection('_')}><em>I</em></IconButton>
                <IconButton title={t('tooltip.highlight')} on:click={() => detailsEditor?.wrapSelection('`')}><code>`</code></IconButton>
                <IconButton title={t('tooltip.mask')} on:click={maskSelection}
                  ><svg viewBox="0 0 16 16" width="13" height="13" aria-hidden="true">
                    <rect x="3.5" y="7" width="9" height="6.5" rx="1.2" fill="none" stroke="currentColor" stroke-width="1.3" />
                    <path d="M5 7V5a3 3 0 0 1 6 0v2" fill="none" stroke="currentColor" stroke-width="1.3" />
                  </svg></IconButton
                >
              </div>
              <CodeMirror
                bind:this={detailsEditor}
                bind:value={displays[selectedDisplay].details}
                language="markdown"
                completions={templateCompletions}
                minHeight="100%"
                maxHeight="100%"
              />
            </div>
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
  .display-section {
    display: flex;
    flex-direction: column;
    gap: 10px;
    height: 100%;
    min-height: 0;
    overflow: hidden;
  }

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

  .display-edit-preview {
    display: flex;
    flex: 1 1 auto;
    min-height: 0;
    overflow: hidden;
  }

  .display-edit-preview.split-h {
    flex-direction: column;
  }

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

  .details-template-field :global(.sm-code) {
    flex: 1 1 auto;
    min-height: 60px;
  }

  .details-template-field :global(.cm-editor) {
    height: 100%;
  }

  .details-helper-toolbar {
    flex: none;
    display: flex;
    align-items: center;
    gap: 4px;
    margin-bottom: 4px;
  }

  .details-helper-toolbar :global(.icon-btn) {
    font-family: var(--sm-font-mono);
    font-size: var(--sm-type-sm);
    line-height: 1;
  }

  .env-insert-select {
    max-width: 160px;
  }
</style>
