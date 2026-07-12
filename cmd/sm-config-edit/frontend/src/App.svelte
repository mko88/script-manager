<script lang="ts">
  import { onMount } from 'svelte'
  import type { DndEvent } from 'svelte-dnd-action'
  import { wrap, sortableList, type DndEntry } from './lib/sortable'
  import Toast from '@shared/components/Toast.svelte'
  import { flash } from '@shared/toast'
  import { getTheme, getThemes, type Theme, type CustomPalette } from '@shared/theme'
  import StringListEditor from './components/StringListEditor.svelte'
  import FieldGrid from './components/FieldGrid.svelte'
  import ActionForm from './components/ActionForm.svelte'
  import ThemeEditor from './components/ThemeEditor.svelte'
  import MessagesEditor from './components/MessagesEditor.svelte'
  import DisplaysEditor from './components/DisplaysEditor.svelte'
  import ActionGroupsEditor from './components/ActionGroupsEditor.svelte'
  import ActionsEditor from './components/ActionsEditor.svelte'
  import ListActionIcon from './components/ListActionIcon.svelte'
  import ToolbarIcon from './components/ToolbarIcon.svelte'
  import { t } from './messages'
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
    GetEditableMessages,
    GetDefaultMessages,
    SaveMessages,
    SetTheme,
    SaveTheme,
    DeleteTheme,
  } from '../wailsjs/go/configedit/App.js'
  import type { configedit } from '../wailsjs/go/models'

  function emptyConfig(): configedit.ConfigDTO {
    return {
      shell: [],
      display: [],
      terminal: { mode: 'auto', name: '', argv: [] },
      envFields: [],
      items: [],
      actionGroups: [],
      actions: [],
    } as unknown as configedit.ConfigDTO
  }

  let cfg: configedit.ConfigDTO = emptyConfig()
  let path = ''

  // No toolbar switcher here anymore — theme/themes just seed ThemeEditor's
  // picker panel and receive its two-way-bound updates as themes are
  // switched, saved, or deleted there.
  let theme: Theme = getTheme()
  let themes: Record<string, CustomPalette> | null = getThemes()

  let knownTerminals: string[] = []
  let validation: configedit.ValidationIssueDTO[] = []
  let initialized = false

  type Section =
    | 'items'
    | 'actionGroups'
    | 'actions'
    | 'display'
    | 'env'
    | 'shell'
    | 'terminal'
    | 'theme'
    | 'messages'
  const sections: { key: Section; label: string }[] = [
    { key: 'items', label: t('nav.items') },
    { key: 'actionGroups', label: t('nav.actionGroups') },
    { key: 'actions', label: t('nav.actions') },
    { key: 'display', label: t('nav.displays') },
    { key: 'env', label: t('nav.environment') },
    { key: 'shell', label: t('nav.shell') },
    { key: 'terminal', label: t('nav.terminal') },
    { key: 'theme', label: t('nav.theme') },
    { key: 'messages', label: t('nav.messages') },
  ]
  let section: Section = 'items'
  $: sectionTitle = sections.find((s) => s.key === section)?.label ?? ''

  let selectedItem = -1
  let selectedActionGroup = -1
  let selectedAction = -1
  let selectedDisplay = -1

  let preview: configedit.PreviewDTO | null = null
  let previewDisplayName = ''
  let previewActionIdx = -1
  let actionPreview: configedit.ActionPreviewDTO | null = null

  onMount(async () => {
    const state = await InitialState()
    applyState(state)
    knownTerminals = await KnownTerminals()
    initialized = true
  })

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
    if (state.warning) flash(t('toast.configLoadWarning', { warning: state.warning }))
  }

  function resetSelection() {
    selectedItem = -1
    selectedActionGroup = -1
    selectedAction = -1
    selectedDisplay = -1
    previewActionIdx = -1
    preview = null
    actionPreview = null
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
    return confirm(t('confirm.discardUnsaved'))
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
      flash(t('toast.openFailed', { error: String(err) }))
    }
  }

  async function doSave(target: string) {
    try {
      const result = await Save(cfg, target)
      path = result.path
      markClean()
      flash(t('toast.saved'))
    } catch (err) {
      flash(t('toast.saveFailed', { error: String(err) }))
    }
  }

  async function saveConfig() {
    if (hasBlockingError) {
      flash(t('toast.fixBlockingErrors'))
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
      flash(t('toast.fixBlockingErrors'))
      return
    }
    const target = await BrowseSaveAs()
    if (target) await doSave(target)
  }

  // Standard New/Open/Save/Save As shortcuts — Ctrl on Windows/Linux, Cmd on
  // Mac. These are modifier combos, not text a focused input would ever
  // insert, so it's safe to handle them regardless of what's focused.
  function handleGlobalKeydown(e: KeyboardEvent) {
    if (!(e.ctrlKey || e.metaKey)) return
    switch (e.key.toLowerCase()) {
      case 'n':
        e.preventDefault()
        newConfig()
        break
      case 'o':
        e.preventDefault()
        openConfig()
        break
      case 's':
        e.preventDefault()
        if (e.shiftKey) saveAsConfig()
        else saveConfig()
        break
    }
  }

  // The generated DTO classes for nested-object fields (ItemDTO, ConfigDTO)
  // carry a convertValues method, so a plain object literal isn't
  // structurally assignable — cast new entries the same way the rest of this
  // file's initial state does.
  function newItem(): configedit.ItemDTO {
    return { name: '', display: '', actions: [], actionGroups: [], customActions: [], fields: [] } as unknown as configedit.ItemDTO
  }
  function newAction(): configedit.ActionDTO {
    return {
      id: '',
      title: '',
      description: '',
      cmd: '',
      groups: [],
      noWait: false,
      interactive: true,
    } as unknown as configedit.ActionDTO
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
  function confirmRemoveItem(i: number) {
    const name = cfg.items[i]?.name || t('fallback.unnamed')
    if (confirm(t('confirm.removeItem', { name }))) removeItem(i)
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

  // Reordering is opt-in: off by default, toggled per-visit via the
  // reorder-mode button in each list's toolbar (not persisted — reopening
  // a section, or the app, starts back in "reordering off"). Without this
  // gate, a plain click-to-select on a row is only one accidental pixel
  // of movement away from silently reordering the list.
  let reorderMode = false
  // Turning reorder mode on clears whatever's selected — keeping a
  // selection alive through a reorder means tracking its index through
  // every live-shifting consider event, which isn't worth the
  // complication. Turning it back off deliberately does *not* try to
  // restore the old selection; it's simply gone, same as if you'd
  // clicked away.
  function toggleReorderMode() {
    reorderMode = !reorderMode
    if (reorderMode) selectedItem = -1
  }

  // Re-derived from cfg.* on any change EXCEPT while a drag is active —
  // during the drag, dndzone owns itemEntries via consider (below), and
  // reactively overwriting it out from under it too (with freshly
  // recreated wrapper objects on every one of our own cfg.items writes)
  // corrupted its internal drag tracking: the dragged entry vanished
  // entirely on drop instead of moving. Outside a drag this still stays
  // correct for free with no manual sync needed at every add/remove/load
  // call site.
  let dragging = false
  let itemEntries: DndEntry<configedit.ItemDTO>[] = wrap(cfg.items)
  $: if (!dragging) itemEntries = wrap(cfg.items)

  // consider fires continuously during the drag (giving the live-shifting
  // preview via dndzone's own flip animation); finalize fires once,
  // settled, on drop or cancel. Only finalize commits to the real cfg.*
  // data — consider only updates what's rendered, exactly the pattern
  // svelte-dnd-action's own examples use. No selection to track through
  // the reorder here: entering reorder mode clears it and blocks
  // reselecting until reorder mode is off again (see toggleReorderMode),
  // so selectedX is always -1 for the whole reorder, nothing to keep in
  // sync with a live-changing index.
  function syncItems(e: CustomEvent<DndEvent<DndEntry<configedit.ItemDTO>>>, final: boolean) {
    itemEntries = e.detail.items
    dragging = !final
    if (final) cfg.items = itemEntries.filter((w) => w.ref).map((w) => w.ref)
  }
  $: allActionIds = cfg.actions.map((a) => a.id).filter((id) => id)
  $: allActionGroups = cfg.actionGroups.map((g) => g.id).filter((id) => id)

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

  // Messages section: extracted into MessagesEditor.svelte.
  // Displays section: extracted into DisplaysEditor.svelte.
</script>

<svelte:window on:keydown={handleGlobalKeydown} />

<main class="app-shell">
  <header class="toolbar">
    <button class="btn icon-btn" type="button" title={t('tooltip.newTitle')} aria-label={t('tooltip.newAria')} on:click={newConfig}
      ><ToolbarIcon mode="new" /></button
    >
    <button class="btn icon-btn" type="button" title={t('tooltip.openTitle')} aria-label={t('tooltip.openAria')} on:click={openConfig}
      ><ToolbarIcon mode="open" /></button
    >
    <button
      class="btn btn-primary icon-btn"
      type="button"
      title={t('tooltip.saveTitle')}
      aria-label={t('tooltip.saveAria')}
      disabled={hasBlockingError}
      on:click={saveConfig}><ToolbarIcon mode="save" /></button
    >
    <button
      class="btn icon-btn"
      type="button"
      title={t('tooltip.saveAsTitle')}
      aria-label={t('tooltip.saveAsAria')}
      disabled={hasBlockingError}
      on:click={saveAsConfig}><ToolbarIcon mode="save-as" /></button
    >
    <span class="toolbar-path">{path || t('text.unsaved')}{dirty ? t('text.dirtyMarker') : ''}</span>
  </header>

  {#if validation.length > 0}
    <div class="validation-banner">
      {#each validation as issue}
        <div class="validation-issue" class:validation-error={issue.severity === 'error'}>
          {issue.severity === 'error' ? t('text.errorIcon') : t('text.warningIcon')}
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
      <div
        class="panel-body"
        class:list-body={section === 'items' || section === 'actionGroups' || section === 'actions' || section === 'theme' || section === 'messages'}
      >
        {#if section === 'shell'}
          <p class="hint">{t('hint.shellCommandPrefix')}<code>pwsh -NoLogo -Command</code>.</p>
          <StringListEditor bind:items={cfg.shell} placeholder={t('placeholder.shellCommand')} />
        {:else if section === 'terminal'}
          <div class="radio-group">
            <label><input type="radio" bind:group={cfg.terminal.mode} value="auto" /> {t('radio.autoDetect')}</label>
            <label><input type="radio" bind:group={cfg.terminal.mode} value="name" /> {t('radio.named')}</label>
            <label><input type="radio" bind:group={cfg.terminal.mode} value="argv" /> {t('radio.customCommand')}</label>
          </div>
          {#if cfg.terminal.mode === 'name'}
            <label class="field">
              <span>{t('field.terminalName')}</span>
              <input type="text" list="known-terminals" bind:value={cfg.terminal.name} placeholder={t('placeholder.terminalName')} />
              <datalist id="known-terminals">
                {#each knownTerminals as name}<option value={name} />{/each}
              </datalist>
            </label>
          {:else if cfg.terminal.mode === 'argv'}
            <p class="hint">
              {t('hint.terminalArgvPrefix')}<code>{'{{title}}'}</code>/<code
                >{'{{dir}}'}</code
              >{t('hint.terminalArgvSuffix')}
            </p>
            <StringListEditor bind:items={cfg.terminal.argv} placeholder={t('placeholder.terminalArgv')} />
          {/if}
        {:else if section === 'env'}
          <p class="hint">{t('hint.envGlobal')}</p>
          <FieldGrid bind:fields={cfg.envFields} validateField={ValidateField} />
        {:else if section === 'display'}
          <DisplaysEditor
            bind:displays={cfg.display}
            bind:selectedDisplay
            items={cfg.items}
            envFields={cfg.envFields}
            previewItem={PreviewItem}
          />
        {:else if section === 'actionGroups'}
          <ActionGroupsEditor
            bind:actionGroups={cfg.actionGroups}
            bind:items={cfg.items}
            bind:actions={cfg.actions}
            bind:selectedActionGroup
          />
        {:else if section === 'actions'}
          <ActionsEditor bind:actions={cfg.actions} bind:selectedAction {allActionGroups} />
        {:else if section === 'items'}
          <div class="list-toolbar">
            <button class="btn icon-btn" type="button" title={t('tooltip.addItem')} aria-label={t('tooltip.addItem')} on:click={addItem}
              ><ListActionIcon mode="add" /></button
            >
            <button
              class="btn icon-btn"
              type="button"
              title={t('tooltip.removeItem')}
              aria-label={t('tooltip.removeItem')}
              disabled={selectedItem < 0}
              on:click={() => confirmRemoveItem(selectedItem)}><ListActionIcon mode="remove" /></button
            >
            <button
              class="btn icon-btn"
              class:active={reorderMode}
              type="button"
              title={reorderMode ? t('tooltip.exitReorderMode') : t('tooltip.enterReorderMode')}
              aria-label={reorderMode ? t('tooltip.exitReorderMode') : t('tooltip.enterReorderMode')}
              on:click={toggleReorderMode}><ListActionIcon mode="reorder" /></button
            >
          </div>
          <div class="master-detail">
            <div
              class="master list"
              class:reorder-mode={reorderMode}
              use:sortableList={{ items: itemEntries, onSync: syncItems, dragDisabled: !reorderMode }}
            >
              {#each itemEntries as entry, i (entry.id)}
                <button
                  class="row"
                  class:selected={selectedItem === i}
                  on:click={() => {
                    if (reorderMode) return
                    selectedItem = i
                    previewActionIdx = -1
                    actionPreview = null
                  }}>{entry.ref.name || t('fallback.unnamed')}</button
                >
              {/each}
            </div>
            <div class="detail">
              {#if selectedItem >= 0 && cfg.items[selectedItem]}
                <label class="field">
                  <span>{t('field.name')}</span>
                  <input type="text" bind:value={cfg.items[selectedItem].name} />
                </label>
                <label class="field">
                  <span>{t('field.display')}</span>
                  <select bind:value={cfg.items[selectedItem].display}>
                    <option value="">{t('option.defaultDisplay')}</option>
                    {#each cfg.display as d}<option value={d.name}>{d.name}</option>{/each}
                  </select>
                </label>

                {#if allActionIds.length > 0}
                  <div class="field">
                    <span>{t('nav.actions')}</span>
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
                    <span>{t('field.itemActionGroupsList')}</span>
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
                  <span>{t('field.customActions')}</span>
                  {#each cfg.items[selectedItem].customActions as _, j (j)}
                    <div class="nested-action">
                      <ActionForm bind:action={cfg.items[selectedItem].customActions[j]} showId={false} {allActionGroups} />
                      <button class="btn" type="button" on:click={() => removeCustomAction(selectedItem, j)}
                        >{t('button.removeCustomAction')}</button
                      >
                    </div>
                  {/each}
                  <button class="btn" type="button" on:click={() => addCustomAction(selectedItem)}
                    >{t('button.addCustomAction')}</button
                  >
                </div>

                <div class="field">
                  <span>{t('nav.environment')}</span>
                  <p class="hint">{t('hint.envItem')}</p>
                  <FieldGrid bind:fields={cfg.items[selectedItem].fields} validateField={ValidateField} />
                </div>

                <div class="preview-pane panel">
                  <header class="panel-title"><span>{t('panel.preview')}</span></header>
                  <div class="panel-body">
                    {#if cfg.display.length > 1}
                      <label class="field">
                        <span>{t('field.previewDisplay')}</span>
                        <select bind:value={previewDisplayName} on:change={schedulePreview}>
                          {#each cfg.display as d}<option value={d.name}>{d.name}</option>{/each}
                        </select>
                      </label>
                    {/if}
                    {#if preview}
                      {#if preview.error}
                        <div class="validation-issue validation-error">{preview.error}</div>
                      {/if}
                      <p class="preview-label">{t('hint.listLabel')}<strong>{preview.listLabel}</strong></p>
                      {#if preview.missingFields?.length}
                        <p class="hint">{t('hint.missingFields', { fields: preview.missingFields.join(', ') })}</p>
                      {/if}
                      <div class="details-preview">{@html preview.detailsHtml}</div>
                    {/if}

                    {#if cfg.actions.length > 0}
                      <label class="field">
                        <span>{t('field.previewAction')}</span>
                        <select bind:value={previewActionIdx} on:change={previewSelectedAction}>
                          <option value={-1}>{t('option.none')}</option>
                          {#each cfg.actions as a, i}<option value={i}>{a.title || a.id}</option>{/each}
                        </select>
                      </label>
                      {#if actionPreview}
                        {#if actionPreview.error}
                          <div class="validation-issue validation-error">{actionPreview.error}</div>
                        {/if}
                        <p class="preview-label">{t('hint.commandLabel')}</p>
                        <pre class="cmd-preview">{actionPreview.cmd}</pre>
                        {#if actionPreview.description}<p class="hint action-desc-preview">{actionPreview.description}</p>{/if}
                      {/if}
                    {/if}
                  </div>
                </div>
              {:else}
                <div class="empty">{t('empty.selectItemOrAdd')}</div>
              {/if}
            </div>
          </div>
        {:else if section === 'theme'}
          <ThemeEditor
            bind:theme
            bind:themes
            saveTheme={SaveTheme}
            deleteTheme={DeleteTheme}
            setActiveTheme={SetTheme}
            {flash}
          />
        {:else if section === 'messages'}
          <MessagesEditor
            getEditableMessages={GetEditableMessages}
            getDefaultMessages={GetDefaultMessages}
            saveMessages={SaveMessages}
          />
        {/if}
      </div>
    </section>
  </div>

  <Toast />
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
    flex: 1 1 auto;
    min-width: 0;
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

  /* Items/Action Groups/Actions: .master-detail's height:100% only works
     out if it's the sole child filling panel-body — with .list-toolbar as
     a sibling above it, "100%" of the same box overflows by the toolbar's
     own height, forcing a scrollbar on panel-body that a user could never
     actually need (master/detail already scroll internally). Making
     panel-body a column flex here — toolbar fixed-height, master-detail
     filling exactly what's left — removes that spurious overflow and, as a
     side effect, keeps the toolbar permanently visible above the list
     without needing script-manager-gui's position:sticky trick (nothing
     here scrolls at the panel-body level to begin with). The extra
     specificity over the plain .panel-body rule is deliberate so this
     doesn't depend on CSS source order between the shared theme and this
     component's scoped styles. */
  .panel-body.list-body {
    display: flex;
    flex-direction: column;
    overflow-y: hidden;
  }

  .hint {
    color: var(--sm-text-muted);
    font-size: 0.8rem;
    margin: 0 0 8px;
  }

  .action-desc-preview {
    white-space: pre-wrap;
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
  .field select {
    background: var(--sm-bg-deep);
    color: var(--sm-text);
    border: 1px solid var(--sm-border);
    border-radius: 4px;
    padding: 5px 7px;
    font-family: inherit;
    font-size: 0.85rem;
  }

  /* Without appearance: none, <select> keeps its native dropdown-arrow
     chrome in Chromium/WebView2 regardless of the background/color set
     above, showing as a jarring light box behind the arrow against this
     dark theme. appearance: none removes that entirely (including the
     arrow itself), so a plain custom chevron replaces it here instead. */
  .field select {
    appearance: none;
    -webkit-appearance: none;
    -moz-appearance: none;
    background-image: url("data:image/svg+xml,%3Csvg xmlns='http://www.w3.org/2000/svg' width='10' height='6' viewBox='0 0 10 6'%3E%3Cpath d='M1 1l4 4 4-4' fill='none' stroke='%23a9b6c8' stroke-width='1.4' stroke-linecap='round' stroke-linejoin='round'/%3E%3C/svg%3E");
    background-repeat: no-repeat;
    background-position: right 10px center;
    padding-right: 28px;
  }

  /* The arrow color is baked into the SVG's URL-encoded string, so it can't
     reference --sm-text-muted via var() — swap the whole background-image
     for the light-theme equivalent (%2355647a = --sm-text-muted's light
     value) instead. */
  :global([data-theme="light"]) .field select {
    background-image: url("data:image/svg+xml,%3Csvg xmlns='http://www.w3.org/2000/svg' width='10' height='6' viewBox='0 0 10 6'%3E%3Cpath d='M1 1l4 4 4-4' fill='none' stroke='%2355647a' stroke-width='1.4' stroke-linecap='round' stroke-linejoin='round'/%3E%3C/svg%3E");
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
    flex: 1 1 auto;
    min-height: 0;
  }

  .master {
    flex: 0 0 200px;
    overflow-y: auto;
  }

  .detail {
    flex: 1 1 auto;
    min-width: 0;
    overflow-y: auto;
    padding-right: 4px;
  }

  .master.reorder-mode .row {
    cursor: grab;
  }

  .icon-btn {
    display: flex;
    align-items: center;
    justify-content: center;
    padding: 6px 9px;
  }

  .icon-btn:disabled {
    opacity: 0.35;
    cursor: default;
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

  /* Mirrors script-manager-gui's .details-content rules for the same
     goldmark-rendered HTML. table-layout: fixed (gui's own table rule
     doesn't need this — it only ever resizes panes by height, never width)
     is the important addition here: without it, an unstyled <table> sizes
     itself to its widest cell and refuses to shrink below that, fighting
     the drag-to-resize divider in split-v mode — this pane, unlike gui's,
     genuinely needs to shrink to arbitrary widths. */
  .details-preview {
    font-size: 0.85rem;
    line-height: 1.5;
    margin-bottom: 8px;
    min-width: 0;
    overflow-wrap: break-word;
  }

  .details-preview :global(h1),
  .details-preview :global(h2),
  .details-preview :global(h3) {
    color: var(--sm-accent);
    margin: 0.6em 0 0.3em;
  }

  .details-preview :global(table) {
    table-layout: fixed;
    border-collapse: collapse;
    width: 100%;
  }

  .details-preview :global(td),
  .details-preview :global(th) {
    border: 1px solid var(--sm-border);
    padding: 4px 8px;
    text-align: left;
    overflow-wrap: break-word;
  }

  .details-preview :global(code) {
    background: var(--sm-bg-deep);
    color: var(--sm-code);
    padding: 1px 5px;
    border-radius: 3px;
    font-family: "SF Mono", Consolas, monospace;
    overflow-wrap: break-word;
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

  /* .messages-* styling now lives in MessagesEditor.svelte. */
</style>
