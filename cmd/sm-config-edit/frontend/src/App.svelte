<script lang="ts">
  import { onMount } from 'svelte'
  import Toast from '@shared/components/Toast.svelte'
  import StringListEditor from './components/StringListEditor.svelte'
  import FieldGrid from './components/FieldGrid.svelte'
  import ActionForm from './components/ActionForm.svelte'
  import {
    InitialState,
    NewBlank,
    BrowseOpen,
    BrowseSaveAs,
    Save,
    PreviewItem,
    PreviewAction,
    ValidateConfig,
    ValidateField,
    KnownTerminals,
  } from '../wailsjs/go/configedit/App.js'
  import type { configedit } from '../wailsjs/go/models'

  function emptyConfig(): configedit.ConfigDTO {
    return {
      shell: [],
      display: [],
      titles: { items: '', actions: '', details: '', command: '' },
      terminal: { mode: 'auto', name: '', argv: [] },
      envFields: [],
      items: [],
      actions: [],
    } as unknown as configedit.ConfigDTO
  }

  let cfg: configedit.ConfigDTO = emptyConfig()
  let path = ''
  let toast = ''
  let toastTimer: ReturnType<typeof setTimeout>
  let knownTerminals: string[] = []
  let validation: configedit.ValidationIssueDTO[] = []
  let initialized = false

  type Section = 'items' | 'actions' | 'display' | 'env' | 'shell' | 'titles' | 'terminal'
  const sections: { key: Section; label: string }[] = [
    { key: 'items', label: 'Items' },
    { key: 'actions', label: 'Actions' },
    { key: 'display', label: 'Displays' },
    { key: 'env', label: 'Environment' },
    { key: 'shell', label: 'Shell' },
    { key: 'titles', label: 'Titles' },
    { key: 'terminal', label: 'Terminal' },
  ]
  let section: Section = 'items'
  $: sectionTitle = sections.find((s) => s.key === section)?.label ?? ''

  let selectedItem = -1
  let selectedAction = -1
  let selectedDisplay = -1

  let preview: configedit.PreviewDTO | null = null
  let previewDisplayName = ''
  let previewActionIdx = -1
  let actionPreview: configedit.ActionPreviewDTO | null = null

  // Displays section: preview an arbitrary item against the display being
  // edited (a display isn't tied to one item the way an item's own preview
  // is), with a layout toggle for how much space editing vs. previewing
  // gets. Kept across display switches (not reset in resetSelection) so you
  // can flip through displays while comparing the same item.
  type DisplayViewMode = 'edit' | 'preview' | 'split-v' | 'split-h'
  let previewItemForDisplay = -1
  let displayViewMode: DisplayViewMode = 'split-v'
  let displayMasterCollapsed = false
  let displayPreview: configedit.PreviewDTO | null = null

  // Split-v sizes the edit pane by width, split-h by height; the other pane
  // always gets flex:1 to soak up whatever's left, so together they fill
  // the full available width (split-v) or height (split-h) — same
  // fixed-primary-pane-plus-flex:1-remainder pattern script-manager-gui's
  // own resizer uses (leftWidth/itemsHeight there).
  const DISPLAY_MIN_PANE = 160
  const DISPLAY_RESIZER = 6
  let displayEditWidth = 480
  let displayEditHeight = 260
  let displaySplitEl: HTMLElement

  const DISPLAY_LAYOUT_KEY = 'sm-config-edit:displayLayout'

  onMount(async () => {
    const state = await InitialState()
    applyState(state)
    knownTerminals = await KnownTerminals()
    initialized = true
  })

  onMount(() => {
    try {
      const saved = JSON.parse(localStorage.getItem(DISPLAY_LAYOUT_KEY) ?? '{}')
      if (saved.viewMode) displayViewMode = saved.viewMode
      displayMasterCollapsed = !!saved.masterCollapsed
      if (typeof saved.editWidth === 'number') displayEditWidth = saved.editWidth
      if (typeof saved.editHeight === 'number') displayEditHeight = saved.editHeight
    } catch {
      // ignore corrupt/missing layout, defaults already set
    }
  })

  function saveDisplayLayout() {
    localStorage.setItem(
      DISPLAY_LAYOUT_KEY,
      JSON.stringify({
        viewMode: displayViewMode,
        masterCollapsed: displayMasterCollapsed,
        editWidth: displayEditWidth,
        editHeight: displayEditHeight,
      }),
    )
  }
  function setDisplayViewMode(mode: DisplayViewMode) {
    displayViewMode = mode
    saveDisplayLayout()
  }
  function toggleDisplayMaster() {
    displayMasterCollapsed = !displayMasterCollapsed
    saveDisplayLayout()
  }

  function dragDisplaySplit(e: MouseEvent) {
    e.preventDefault()
    const horizontal = displayViewMode === 'split-h'
    const startPos = horizontal ? e.clientY : e.clientX
    const startSize = horizontal ? displayEditHeight : displayEditWidth
    function onMove(ev: MouseEvent) {
      const rect = displaySplitEl.getBoundingClientRect()
      const total = horizontal ? rect.height : rect.width
      const max = total - DISPLAY_MIN_PANE - DISPLAY_RESIZER
      const pos = horizontal ? ev.clientY : ev.clientX
      const next = Math.min(max, Math.max(DISPLAY_MIN_PANE, startSize + (pos - startPos)))
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

  // The "clean" snapshot dirty is compared against, taken every time cfg is
  // set programmatically (load/save) rather than by the user editing a
  // field. A boolean toggled by "cfg was reassigned" doesn't work here: cfg
  // is reassigned by applyState too (loading is itself a reassignment), and
  // initialized flips true in a *later* tick than applyState's — so an
  // edge-triggered flag fires once, spuriously, right after every load. A
  // value comparison instead of an edge trigger sidesteps that entirely.
  let cleanSnapshot = ''
  function markClean() {
    cleanSnapshot = JSON.stringify(cfg)
  }

  function applyState(state: configedit.StateDTO) {
    cfg = state.config
    path = state.path
    markClean()
    if (state.warning) flash(`Config load warning: ${state.warning}`)
  }

  function resetSelection() {
    selectedItem = -1
    selectedAction = -1
    selectedDisplay = -1
    previewActionIdx = -1
    preview = null
    actionPreview = null
    // A brand new/loaded config invalidates item indices entirely, unlike
    // switching between displays within the same config (where keeping the
    // same preview item is the point).
    previewItemForDisplay = -1
    displayPreview = null
  }

  function flash(msg: string) {
    toast = msg
    clearTimeout(toastTimer)
    toastTimer = setTimeout(() => (toast = ''), 3000)
  }

  let validateTimer: ReturnType<typeof setTimeout>
  function scheduleValidate() {
    clearTimeout(validateTimer)
    validateTimer = setTimeout(async () => {
      validation = await ValidateConfig(cfg)
    }, 300)
  }

  // dirty is a pure derived comparison against the last clean snapshot, not
  // an edge-triggered flag — see markClean's comment for why.
  $: dirty = initialized && JSON.stringify(cfg) !== cleanSnapshot

  // Validation re-runs on every nested edit too: Svelte's bind: chains
  // (StringListEditor, FieldGrid, ActionForm) all invalidate cfg up to this
  // root, which this statement is watching. Re-validating once extra right
  // after a load (before cleanSnapshot is compared) is harmless.
  $: if (initialized && cfg) scheduleValidate()

  $: hasBlockingError = validation.some((v) => v.severity === 'error')

  async function confirmDiscard(): Promise<boolean> {
    if (!dirty) return true
    return confirm('Discard unsaved changes?')
  }

  async function newConfig() {
    if (!(await confirmDiscard())) return
    applyState(await NewBlank())
    resetSelection()
  }

  async function openConfig() {
    if (!(await confirmDiscard())) return
    try {
      const state = await BrowseOpen()
      applyState(state)
      resetSelection()
    } catch (err) {
      flash(`Open failed: ${err}`)
    }
  }

  async function doSave(target: string) {
    try {
      const result = await Save(cfg, target)
      path = result.path
      markClean()
      flash('Saved')
    } catch (err) {
      flash(`Save failed: ${err}`)
    }
  }

  async function saveConfig() {
    if (hasBlockingError) {
      flash('Fix blocking errors before saving')
      return
    }
    if (path) {
      await doSave(path)
      return
    }
    const target = await BrowseSaveAs()
    if (target) await doSave(target)
  }

  async function saveAsConfig() {
    if (hasBlockingError) {
      flash('Fix blocking errors before saving')
      return
    }
    const target = await BrowseSaveAs()
    if (target) await doSave(target)
  }

  // The generated DTO classes for nested-object fields (ItemDTO, ConfigDTO)
  // carry a convertValues method, so a plain object literal isn't
  // structurally assignable — cast new entries the same way the rest of this
  // file's initial state does.
  function newItem(): configedit.ItemDTO {
    return { name: '', display: '', actions: [], actionGroups: [], customActions: [], fields: [] } as unknown as configedit.ItemDTO
  }
  function newAction(): configedit.ActionDTO {
    return { id: '', title: '', description: '', cmd: '', groups: [], noWait: false } as unknown as configedit.ActionDTO
  }
  function newDisplay(): configedit.DisplayDTO {
    return { name: '', list: '{{.name}}', details: '' } as unknown as configedit.DisplayDTO
  }

  function addItem() {
    cfg.items = [...cfg.items, newItem()]
    selectedItem = cfg.items.length - 1
    previewActionIdx = -1
  }
  function removeItem(i: number) {
    cfg.items = cfg.items.filter((_, idx) => idx !== i)
    if (selectedItem === i) selectedItem = -1
    else if (selectedItem > i) selectedItem -= 1
  }

  function addAction() {
    cfg.actions = [...cfg.actions, newAction()]
    selectedAction = cfg.actions.length - 1
  }
  function removeAction(i: number) {
    cfg.actions = cfg.actions.filter((_, idx) => idx !== i)
    if (selectedAction === i) selectedAction = -1
    else if (selectedAction > i) selectedAction -= 1
  }

  function addDisplay() {
    cfg.display = [...cfg.display, newDisplay()]
    selectedDisplay = cfg.display.length - 1
  }
  function removeDisplay(i: number) {
    cfg.display = cfg.display.filter((_, idx) => idx !== i)
    if (selectedDisplay === i) selectedDisplay = -1
    else if (selectedDisplay > i) selectedDisplay -= 1
  }
  function confirmRemoveDisplay(i: number) {
    const name = cfg.display[i]?.name || '(unnamed)'
    if (confirm(`Remove display "${name}"? This can't be undone.`)) removeDisplay(i)
  }

  function addCustomAction(itemIdx: number) {
    cfg.items[itemIdx].customActions = [...cfg.items[itemIdx].customActions, newAction()]
  }
  function removeCustomAction(itemIdx: number, i: number) {
    cfg.items[itemIdx].customActions = cfg.items[itemIdx].customActions.filter((_, idx) => idx !== i)
  }

  function toggleInList(list: string[], value: string): string[] {
    return list.includes(value) ? list.filter((v) => v !== value) : [...list, value]
  }

  $: allActionIds = cfg.actions.map((a) => a.id).filter((id) => id)
  $: allActionGroups = (() => {
    const seen = new Set<string>()
    for (const a of cfg.actions) for (const g of a.groups ?? []) seen.add(g)
    return [...seen]
  })()

  let previewTimer: ReturnType<typeof setTimeout>
  $: if (initialized && section === 'items' && selectedItem >= 0 && cfg.items[selectedItem]) schedulePreview()
  function schedulePreview() {
    clearTimeout(previewTimer)
    previewTimer = setTimeout(async () => {
      const item = cfg.items[selectedItem]
      if (!item) return
      if (!previewDisplayName && cfg.display.length > 0) previewDisplayName = item.display || cfg.display[0].name
      preview = await PreviewItem(item, cfg.envFields, cfg.display, previewDisplayName || item.display)
    }, 250)
  }

  async function previewSelectedAction() {
    if (selectedItem < 0 || previewActionIdx < 0) {
      actionPreview = null
      return
    }
    const act = cfg.actions[previewActionIdx]
    if (!act) return
    actionPreview = await PreviewAction(cfg.items[selectedItem], cfg.envFields, act)
  }

  let displayPreviewTimer: ReturnType<typeof setTimeout>
  // previewItemForDisplay >= -1 is always true — it's just there so Svelte
  // tracks it as a dependency of this statement (picking a different
  // preview item must re-trigger this the same way editing the template
  // does), not a real condition.
  $: if (
    initialized &&
    section === 'display' &&
    selectedDisplay >= 0 &&
    cfg.display[selectedDisplay] &&
    previewItemForDisplay >= -1
  )
    scheduleDisplayPreview()
  function scheduleDisplayPreview() {
    clearTimeout(displayPreviewTimer)
    displayPreviewTimer = setTimeout(async () => {
      const d = cfg.display[selectedDisplay]
      const item = cfg.items[previewItemForDisplay]
      if (!d || !item) {
        displayPreview = null
        return
      }
      displayPreview = await PreviewItem(item, cfg.envFields, cfg.display, d.name)
    }, 250)
  }
</script>

<main class="app-shell">
  <header class="toolbar">
    <button class="btn" type="button" on:click={newConfig}>New</button>
    <button class="btn" type="button" on:click={openConfig}>Open</button>
    <button class="btn btn-primary" type="button" disabled={hasBlockingError} on:click={saveConfig}>Save</button>
    <button class="btn" type="button" disabled={hasBlockingError} on:click={saveAsConfig}>Save As</button>
    <span class="toolbar-path">{path || '(unsaved)'}{dirty ? ' *' : ''}</span>
  </header>

  {#if validation.length > 0}
    <div class="validation-banner">
      {#each validation as issue}
        <div class="validation-issue" class:validation-error={issue.severity === 'error'}>
          {issue.severity === 'error' ? '⛔' : '⚠'}
          {issue.message}
        </div>
      {/each}
    </div>
  {/if}

  <div class="body">
    <nav class="section-nav list">
      {#each sections as s (s.key)}
        <button class="row" class:selected={section === s.key} on:click={() => (section = s.key)}>{s.label}</button>
      {/each}
    </nav>

    <section class="panel main-panel">
      <header class="panel-title"><span>{sectionTitle}</span></header>
      <div class="panel-body">
        {#if section === 'shell'}
          <p class="hint">The command used to launch actions, e.g. <code>pwsh -NoLogo -Command</code>.</p>
          <StringListEditor bind:items={cfg.shell} placeholder="e.g. pwsh" />
        {:else if section === 'titles'}
          <label class="field">
            <span>Items pane title</span>
            <input type="text" bind:value={cfg.titles.items} placeholder="Items" />
          </label>
          <label class="field">
            <span>Actions pane title</span>
            <input type="text" bind:value={cfg.titles.actions} placeholder="Actions" />
          </label>
          <label class="field">
            <span>Details pane title</span>
            <input type="text" bind:value={cfg.titles.details} placeholder="Details" />
          </label>
          <label class="field">
            <span>Command pane title</span>
            <input type="text" bind:value={cfg.titles.command} placeholder="Command" />
          </label>
        {:else if section === 'terminal'}
          <div class="radio-group">
            <label><input type="radio" bind:group={cfg.terminal.mode} value="auto" /> Auto-detect</label>
            <label><input type="radio" bind:group={cfg.terminal.mode} value="name" /> Named</label>
            <label><input type="radio" bind:group={cfg.terminal.mode} value="argv" /> Custom command</label>
          </div>
          {#if cfg.terminal.mode === 'name'}
            <label class="field">
              <span>Terminal name</span>
              <input type="text" list="known-terminals" bind:value={cfg.terminal.name} placeholder="e.g. wt" />
              <datalist id="known-terminals">
                {#each knownTerminals as name}<option value={name} />{/each}
              </datalist>
            </label>
          {:else if cfg.terminal.mode === 'argv'}
            <p class="hint">
              First entry is the terminal binary; the rest are its flags. Use <code>{'{{title}}'}</code>/<code
                >{'{{dir}}'}</code
              > as placeholders.
            </p>
            <StringListEditor bind:items={cfg.terminal.argv} placeholder="e.g. --title" />
          {/if}
        {:else if section === 'env'}
          <p class="hint">Available to every item's templates and subprocess environment.</p>
          <FieldGrid bind:fields={cfg.envFields} validateField={ValidateField} />
        {:else if section === 'display'}
          <div class="master-detail">
            {#if !displayMasterCollapsed}
              <div class="master list">
                <div class="master-header">
                  <span class="master-header-label">Displays</span>
                  <button class="collapse-btn" type="button" on:click={toggleDisplayMaster} title="Collapse list"
                    >◂</button
                  >
                </div>
                {#each cfg.display as d, i (i)}
                  <button class="row" class:selected={selectedDisplay === i} on:click={() => (selectedDisplay = i)}
                    >{d.name || '(unnamed)'}</button
                  >
                {/each}
                <button class="btn" type="button" on:click={addDisplay}>+ Add display</button>
              </div>
            {:else}
              <button
                class="master-collapsed-toggle"
                type="button"
                on:click={toggleDisplayMaster}
                title="Expand display list">▸</button
              >
            {/if}
            <div class="detail display-detail">
              {#if selectedDisplay >= 0 && cfg.display[selectedDisplay]}
                <div class="display-top">
                  <label class="field">
                    <span>Name</span>
                    <input type="text" bind:value={cfg.display[selectedDisplay].name} />
                  </label>

                  <div class="display-toolbar">
                    <div class="view-mode-group">
                      <button
                        class="btn"
                        class:active={displayViewMode === 'edit'}
                        type="button"
                        title="Edit only"
                        on:click={() => setDisplayViewMode('edit')}>Edit</button
                      >
                      <button
                        class="btn"
                        class:active={displayViewMode === 'preview'}
                        type="button"
                        title="Preview only"
                        on:click={() => setDisplayViewMode('preview')}>Preview</button
                      >
                      <button
                        class="btn"
                        class:active={displayViewMode === 'split-v'}
                        type="button"
                        title="Side by side"
                        on:click={() => setDisplayViewMode('split-v')}>Split ↔</button
                      >
                      <button
                        class="btn"
                        class:active={displayViewMode === 'split-h'}
                        type="button"
                        title="Stacked"
                        on:click={() => setDisplayViewMode('split-h')}>Split ↕</button
                      >
                    </div>
                    <label class="field preview-item-picker">
                      <span>Preview item</span>
                      <select bind:value={previewItemForDisplay} on:change={scheduleDisplayPreview}>
                        <option value={-1}>(none)</option>
                        {#each cfg.items as it, i}<option value={i}>{it.name || `(unnamed item ${i + 1})`}</option
                          >{/each}
                      </select>
                    </label>
                    <button
                      class="btn btn-danger"
                      type="button"
                      on:click={() => confirmRemoveDisplay(selectedDisplay)}>Remove display</button
                    >
                  </div>
                </div>

                <div
                  class="display-edit-preview"
                  class:split-v={displayViewMode === 'split-v'}
                  class:split-h={displayViewMode === 'split-h'}
                  bind:this={displaySplitEl}
                >
                  {#if displayViewMode !== 'preview'}
                    <div
                      class="edit-pane"
                      style={displayViewMode === 'split-v'
                        ? `flex: 0 1 ${displayEditWidth}px`
                        : displayViewMode === 'split-h'
                          ? `flex: 0 1 ${displayEditHeight}px`
                          : ''}
                    >
                      <label class="field list-template-field">
                        <span>List template</span>
                        <textarea rows="2" bind:value={cfg.display[selectedDisplay].list}></textarea>
                      </label>
                      <label class="field details-template-field">
                        <span>Details template</span>
                        <textarea bind:value={cfg.display[selectedDisplay].details}></textarea>
                      </label>
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
                      <header class="panel-title"><span>Preview</span></header>
                      <div class="panel-body">
                        {#if previewItemForDisplay < 0}
                          <div class="empty">Pick an item above to preview against.</div>
                        {:else if displayPreview}
                          {#if displayPreview.error}
                            <div class="validation-issue validation-error">{displayPreview.error}</div>
                          {/if}
                          <p class="preview-label">List label: <strong>{displayPreview.listLabel}</strong></p>
                          {#if displayPreview.missingFields?.length}
                            <p class="hint">⚠ missing: {displayPreview.missingFields.join(', ')}</p>
                          {/if}
                          <div class="details-preview">{@html displayPreview.detailsHtml}</div>
                        {/if}
                      </div>
                    </div>
                  {/if}
                </div>
              {:else}
                <div class="empty">Select a display, or add one.</div>
              {/if}
            </div>
          </div>
        {:else if section === 'actions'}
          <div class="master-detail">
            <div class="master list">
              {#each cfg.actions as a, i (i)}
                <button class="row" class:selected={selectedAction === i} on:click={() => (selectedAction = i)}
                  >{a.title || a.id || '(untitled)'}</button
                >
              {/each}
              <button class="btn" type="button" on:click={addAction}>+ Add action</button>
            </div>
            <div class="detail">
              {#if selectedAction >= 0 && cfg.actions[selectedAction]}
                <ActionForm bind:action={cfg.actions[selectedAction]} />
                <button class="btn" type="button" on:click={() => removeAction(selectedAction)}
                  >Remove action</button
                >
              {:else}
                <div class="empty">Select an action, or add one.</div>
              {/if}
            </div>
          </div>
        {:else if section === 'items'}
          <div class="master-detail">
            <div class="master list">
              {#each cfg.items as it, i (i)}
                <button
                  class="row"
                  class:selected={selectedItem === i}
                  on:click={() => {
                    selectedItem = i
                    previewActionIdx = -1
                    actionPreview = null
                  }}>{it.name || '(unnamed)'}</button
                >
              {/each}
              <button class="btn" type="button" on:click={addItem}>+ Add item</button>
            </div>
            <div class="detail">
              {#if selectedItem >= 0 && cfg.items[selectedItem]}
                <label class="field">
                  <span>Name</span>
                  <input type="text" bind:value={cfg.items[selectedItem].name} />
                </label>
                <label class="field">
                  <span>Display</span>
                  <select bind:value={cfg.items[selectedItem].display}>
                    <option value="">(default)</option>
                    {#each cfg.display as d}<option value={d.name}>{d.name}</option>{/each}
                  </select>
                </label>

                {#if allActionIds.length > 0}
                  <div class="field">
                    <span>Actions</span>
                    <div class="checkbox-list">
                      {#each allActionIds as id}
                        <label class="checkbox-chip">
                          <input
                            type="checkbox"
                            checked={cfg.items[selectedItem].actions.includes(id)}
                            on:change={() =>
                              (cfg.items[selectedItem].actions = toggleInList(cfg.items[selectedItem].actions, id))}
                          />
                          {id}
                        </label>
                      {/each}
                    </div>
                  </div>
                {/if}

                {#if allActionGroups.length > 0}
                  <div class="field">
                    <span>Action groups</span>
                    <div class="checkbox-list">
                      {#each allActionGroups as g}
                        <label class="checkbox-chip">
                          <input
                            type="checkbox"
                            checked={cfg.items[selectedItem].actionGroups.includes(g)}
                            on:change={() =>
                              (cfg.items[selectedItem].actionGroups = toggleInList(
                                cfg.items[selectedItem].actionGroups,
                                g,
                              ))}
                          />
                          {g}
                        </label>
                      {/each}
                    </div>
                  </div>
                {/if}

                <div class="field">
                  <span>Custom actions</span>
                  {#each cfg.items[selectedItem].customActions as _, j (j)}
                    <div class="nested-action">
                      <ActionForm bind:action={cfg.items[selectedItem].customActions[j]} showId={false} />
                      <button class="btn" type="button" on:click={() => removeCustomAction(selectedItem, j)}
                        >Remove custom action</button
                      >
                    </div>
                  {/each}
                  <button class="btn" type="button" on:click={() => addCustomAction(selectedItem)}
                    >+ Add custom action</button
                  >
                </div>

                <div class="field">
                  <span>Additional fields</span>
                  <FieldGrid bind:fields={cfg.items[selectedItem].fields} validateField={ValidateField} />
                </div>

                <button class="btn" type="button" on:click={() => removeItem(selectedItem)}>Remove item</button>

                <div class="preview-pane panel">
                  <header class="panel-title"><span>Preview</span></header>
                  <div class="panel-body">
                    {#if cfg.display.length > 1}
                      <label class="field">
                        <span>Preview display</span>
                        <select bind:value={previewDisplayName} on:change={schedulePreview}>
                          {#each cfg.display as d}<option value={d.name}>{d.name}</option>{/each}
                        </select>
                      </label>
                    {/if}
                    {#if preview}
                      {#if preview.error}
                        <div class="validation-issue validation-error">{preview.error}</div>
                      {/if}
                      <p class="preview-label">List label: <strong>{preview.listLabel}</strong></p>
                      {#if preview.missingFields?.length}
                        <p class="hint">⚠ missing: {preview.missingFields.join(', ')}</p>
                      {/if}
                      <div class="details-preview">{@html preview.detailsHtml}</div>
                    {/if}

                    {#if cfg.actions.length > 0}
                      <label class="field">
                        <span>Preview action</span>
                        <select bind:value={previewActionIdx} on:change={previewSelectedAction}>
                          <option value={-1}>(none)</option>
                          {#each cfg.actions as a, i}<option value={i}>{a.title || a.id}</option>{/each}
                        </select>
                      </label>
                      {#if actionPreview}
                        {#if actionPreview.error}
                          <div class="validation-issue validation-error">{actionPreview.error}</div>
                        {/if}
                        <p class="preview-label">Command:</p>
                        <pre class="cmd-preview">{actionPreview.cmd}</pre>
                        {#if actionPreview.description}<p class="hint">{actionPreview.description}</p>{/if}
                      {/if}
                    {/if}
                  </div>
                </div>
              {:else}
                <div class="empty">Select an item, or add one.</div>
              {/if}
            </div>
          </div>
        {/if}
      </div>
    </section>
  </div>

  <Toast message={toast} />
</main>

<style>
  .app-shell {
    display: flex;
    flex-direction: column;
    height: 100vh;
    box-sizing: border-box;
    padding: 8px;
    gap: 8px;
    text-align: left;
  }

  .toolbar {
    flex: none;
    display: flex;
    align-items: center;
    gap: 8px;
  }

  .toolbar-path {
    margin-left: 8px;
    color: var(--sm-text-muted);
    font-size: 0.85rem;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .validation-banner {
    flex: none;
    display: flex;
    flex-direction: column;
    gap: 2px;
    max-height: 120px;
    overflow-y: auto;
    background: rgba(232, 163, 61, 0.1);
    border: 1px solid var(--sm-border);
    border-radius: 6px;
    padding: 6px 10px;
  }

  .validation-issue {
    font-size: 0.8rem;
    color: var(--sm-text-muted);
  }

  .validation-issue.validation-error {
    color: var(--sm-accent-warm);
    font-weight: 700;
  }

  .body {
    flex: 1 1 auto;
    display: flex;
    gap: 8px;
    min-height: 0;
  }

  .section-nav {
    flex: 0 0 160px;
  }

  .main-panel {
    flex: 1 1 auto;
    min-width: 0;
  }

  .hint {
    color: var(--sm-text-muted);
    font-size: 0.8rem;
    margin: 0 0 8px;
  }

  .hint code {
    background: var(--sm-bg-deep);
    padding: 1px 4px;
    border-radius: 3px;
  }

  .field {
    display: flex;
    flex-direction: column;
    gap: 2px;
    font-size: 0.8rem;
    color: var(--sm-text-muted);
    margin-bottom: 10px;
  }

  .field input,
  .field select,
  .field textarea {
    background: var(--sm-bg-deep);
    color: var(--sm-text);
    border: 1px solid var(--sm-border);
    border-radius: 4px;
    padding: 5px 7px;
    font-family: inherit;
    font-size: 0.85rem;
  }

  .radio-group {
    display: flex;
    gap: 16px;
    margin-bottom: 10px;
    font-size: 0.85rem;
  }

  .master-detail {
    display: flex;
    gap: 10px;
    height: 100%;
    min-height: 0;
  }

  .master {
    flex: 0 0 200px;
    overflow-y: auto;
  }

  .master-header {
    flex: none;
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding: 0 2px 4px;
  }

  .master-header-label {
    font-size: 0.72rem;
    font-weight: 700;
    letter-spacing: 0.03em;
    text-transform: uppercase;
    color: var(--sm-text-faint);
  }

  .master-collapsed-toggle {
    flex: 0 0 22px;
    background: var(--sm-bg-alt);
    border: 1px solid var(--sm-border);
    border-radius: 4px;
    color: var(--sm-accent);
    cursor: pointer;
    font-size: 0.8rem;
  }

  .detail {
    flex: 1 1 auto;
    min-width: 0;
    overflow-y: auto;
    padding-right: 4px;
  }

  /* The Displays detail pane doesn't scroll as a whole (unlike .detail
     elsewhere): its edit/preview split needs a real, bounded height to
     resize within, so display-detail fills the available height exactly
     and lets display-edit-preview's two panes scroll internally instead. */
  .display-detail {
    display: flex;
    flex-direction: column;
    min-height: 0;
    overflow: hidden;
  }

  .display-top {
    flex: none;
  }

  .display-toolbar {
    display: flex;
    align-items: flex-end;
    justify-content: space-between;
    gap: 12px;
    flex-wrap: wrap;
    margin-bottom: 10px;
  }

  .view-mode-group {
    display: flex;
    gap: 4px;
  }

  .view-mode-group .btn.active {
    background: var(--sm-accent-warm);
    border-color: var(--sm-accent-warm);
    color: var(--sm-bg);
    font-weight: 700;
  }

  .preview-item-picker {
    flex: 0 0 220px;
    margin-bottom: 0;
  }

  .btn-danger {
    background: var(--sm-bg-deep);
    border-color: #b23b3b;
    color: #ff8a8a;
  }

  .btn-danger:hover {
    background: #b23b3b;
    color: var(--sm-text);
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

  /* Base size for both panes is flex:1, so a single visible pane (Edit-only/
     Preview-only) always fills the full width and height. In split modes,
     the template above overrides edit-pane's flex-basis to a fixed pixel
     size (dragDisplaySplit); the other pane keeps flex:1 to soak up
     whatever's left, so together they always fill the full available
     width (split-v) or height (split-h), however the window is sized. */
  .edit-pane {
    display: flex;
    flex-direction: column;
    gap: 8px;
    flex: 1 1 auto;
    min-width: 0;
    min-height: 0;
    overflow: hidden;
  }

  .preview-pane-inline {
    flex: 1 1 auto;
    min-width: 0;
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
  }

  .checkbox-list {
    display: flex;
    flex-wrap: wrap;
    gap: 6px;
  }

  .checkbox-chip {
    display: flex;
    align-items: center;
    gap: 4px;
    background: var(--sm-bg-deep);
    border: 1px solid var(--sm-border);
    border-radius: 999px;
    padding: 2px 9px;
    font-size: 0.75rem;
    cursor: pointer;
  }

  .nested-action {
    border: 1px solid var(--sm-border);
    border-radius: 6px;
    padding: 8px;
    margin-bottom: 8px;
    background: var(--sm-bg-deep);
  }

  .preview-pane {
    margin-top: 16px;
  }

  .preview-label {
    margin: 4px 0;
    font-size: 0.85rem;
  }

  .details-preview {
    font-size: 0.85rem;
    line-height: 1.5;
    margin-bottom: 8px;
  }

  .cmd-preview {
    background: var(--sm-bg-deep);
    border-radius: 4px;
    padding: 8px;
    font-family: "SF Mono", Consolas, monospace;
    font-size: 0.8rem;
    white-space: pre-wrap;
    word-break: break-word;
  }
</style>
