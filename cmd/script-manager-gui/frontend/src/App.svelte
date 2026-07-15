<script lang="ts">
  import { onMount, tick } from 'svelte'
  import Toast from '@shared/components/Toast.svelte'
  import { flash } from '@shared/toast'
  import { loadPersisted, savePersisted } from '@shared/persist'
  import { watchTheme } from '@shared/theme'
  import Icon from '@shared/components/Icon.svelte'
  import CollapseToggle from '@shared/components/CollapseToggle.svelte'
  import IconButton from '@shared/components/IconButton.svelte'
  import ScriptSource from '@shared/components/ScriptSource.svelte'
  import Panel from './components/Panel.svelte'
  import GroupFilter from './components/GroupFilter.svelte'
  import { t } from './messages'
  import { buildGroupColors, groupChipStyle } from './lib/groupColors'
  import { inlineStates, inlineKey, startInlineRun, cancelInlineRun } from './lib/inlineRuns'
  import { dragColumn, dragRow, topStyle, bottomStyle } from './lib/panelLayout'
  import { EventsOn } from '../wailsjs/runtime'
  import {
    GetItems,
    GetActions,
    GetActionDetail,
    GetItemDetails,
    GetActionGroups,
    CopyToClipboard,
    ReloadConfig,
    BrowseConfig,
    LaunchConfigEditor,
    RunAction,
    LoadError,
  } from '../wailsjs/go/gui/App.js'
  import type { gui } from '../wailsjs/go/models'

  let items: gui.ItemDTO[] = []
  let actions: gui.ActionDTO[] = []
  let actionGroupCatalog: gui.ActionGroupDTO[] = []
  let details: gui.DetailsDTO | null = null
  let actionDetail: gui.ActionDetailDTO | null = null

  let selectedItem = -1
  let selectedActionIndex = -1
  // Two-way bound into GroupFilter; empty set means "All" — no filter.
  let selectedGroups = new Set<string>()

  // Live reload: internal/gui/themewatch.go watches sm-theme.json and
  // pushes a Wails event whenever sm-config-edit changes it, so a theme
  // switched or saved there shows up here without needing to relaunch.
  // This app never switches themes itself, so there's no reactive state to
  // update here — watchTheme applies the change internally regardless.
  onMount(() => watchTheme(EventsOn, () => {}))

  let inlineOutputEl: HTMLElement | undefined

  // Autoscrolls the inline output box to the newest line as it streams in.
  // Called directly from the poll loop below when it's updating the
  // currently-viewed action, not from a `$:` reactive statement watching
  // output/inlineOutputEl together — that shape was tried first and
  // reliably broke Wails' own bound-method delivery (and, separately, an
  // EventSource-based version of this feature) in WebKitGTK:
  // inlineOutputEl is only bound once the <pre> below actually renders
  // (there must already be output for that), so the reactive statement's
  // own dependency on both variables together, right as new output
  // arrived, was the trigger. Root-caused by bisection, not fully
  // understood at the WebKitGTK level.
  async function scrollInlineOutputToEnd() {
    await tick()
    if (inlineOutputEl) inlineOutputEl.scrollTop = inlineOutputEl.scrollHeight
  }

  // What the Command pane actually displays — derived from the shared
  // inlineRuns store (see lib/inlineRuns) for whatever's currently selected,
  // defaulting to "never run" (blank, not running) when there's no entry yet.
  $: currentInline = selectedItem >= 0 && selectedActionIndex >= 0 ? $inlineStates[inlineKey(selectedItem, selectedActionIndex)] : undefined
  $: inlineRunning = currentInline?.running ?? false
  $: inlineOutput = currentInline?.output ?? ''
  $: inlineExitCode = currentInline?.exitCode ?? null

  // Which items/actions to show a running indicator for — every entry
  // still running, cross-referenced by itemIndex for the Items list and by
  // actionIndex (within the selected item) for the Actions list. Pure data
  // derivations, no DOM access — safe alongside the bug described above,
  // which was specifically about a reactive statement that touched the DOM.
  $: runningItemIndices = new Set(Object.values($inlineStates).filter((s) => s.running).map((s) => s.itemIndex))
  $: runningActionIndicesForSelectedItem = new Set(
    Object.values($inlineStates)
      .filter((s) => s.running && s.itemIndex === selectedItem)
      .map((s) => s.actionIndex),
  )

  $: filteredActions =
    selectedGroups.size === 0
      ? actions
      : actions.filter((a) => [...selectedGroups].every((g) => (a.groups ?? []).includes(g)))

  $: missingFields = details?.missingFields ?? []

  $: selectedItemLabel = items.find((i) => i.index === selectedItem)?.label ?? ''
  $: selectedActionLabel = actions.find((a) => a.index === selectedActionIndex)?.title ?? ''
  $: selectedActionGroups = actions.find((a) => a.index === selectedActionIndex)?.groups ?? []

  $: groupColors = buildGroupColors(actionGroupCatalog)

  onMount(async () => {
    const loadErr = await LoadError()
    if (loadErr) flash(t('toast.configLoadFailed', { error: loadErr }))
    items = await GetItems()
    actionGroupCatalog = await GetActionGroups()
    if (items.length > 0) selectItem(0)
  })

  async function selectItem(index: number) {
    selectedItem = index
    selectedActionIndex = -1
    selectedGroups = new Set()
    actionDetail = null
    actions = await GetActions(index)
    details = await GetItemDetails(index)
  }

  // A group-filter change can hide the selected action, so the selection is
  // always reset alongside it.
  function onGroupFilterChange() {
    selectedActionIndex = -1
    actionDetail = null
  }

  async function selectAction(index: number) {
    if (selectedItem < 0) return
    selectedActionIndex = index
    actionDetail = await GetActionDetail(selectedItem, index)
  }

  async function copyToClipboard(value: string) {
    try {
      await CopyToClipboard(value)
      flash(t('toast.copiedToClipboard'))
    } catch (err) {
      flash(t('toast.clipboardUnavailable', { error: String(err) }))
    }
  }

  function copyValue(idx: number) {
    if (!details) return
    const value = details.copyValues[idx]
    if (value === undefined) return
    copyToClipboard(value)
  }

  function onDetailsClick(e: MouseEvent) {
    const target = (e.target as HTMLElement).closest('[data-copy-idx]') as HTMLElement | null
    if (!target) return
    copyValue(Number(target.dataset.copyIdx))
  }

  function copyCmd() {
    const value = actionDetail?.cmd || actionDetail?.script
    if (!value) return
    copyToClipboard(value)
  }

  async function runAction() {
    if (selectedItem < 0 || selectedActionIndex < 0) return
    try {
      await RunAction(selectedItem, selectedActionIndex)
      flash(t('toast.runningInTerminal'))
    } catch (err) {
      flash(t('toast.runFailed', { error: String(err) }))
    }
  }

  // The run/poll mechanics live in lib/inlineRuns — these wrappers just tie
  // them to the current selection, plus the scroll side effect for whichever
  // run is on screen right now (a DOM concern that stays in this component;
  // see scrollInlineOutputToEnd's doc comment above).
  function runActionInline() {
    if (selectedItem < 0 || selectedActionIndex < 0) return
    startInlineRun(selectedItem, selectedActionIndex, (itemIndex, actionIndex) => {
      if (selectedItem === itemIndex && selectedActionIndex === actionIndex) {
        scrollInlineOutputToEnd()
      }
    })
  }

  function cancelInlineAction() {
    if (selectedItem < 0 || selectedActionIndex < 0) return
    cancelInlineRun(selectedItem, selectedActionIndex)
  }

  // Shared by reloadConfig (F5 / Refresh config) and browseConfig (Load
  // config) — both swap the backend's in-memory config out from under the
  // frontend, so both need the same items/actions/details re-fetch and
  // reselect-something-sane dance afterward.
  async function refreshAfterConfigChange() {
    actionGroupCatalog = await GetActionGroups()
    const newItems = await GetItems()
    items = newItems
    if (newItems.length === 0) {
      selectedItem = -1
      actions = []
      details = null
      actionDetail = null
    } else {
      await selectItem(Math.min(selectedItem < 0 ? 0 : selectedItem, newItems.length - 1))
    }
  }

  async function reloadConfig() {
    let warning = ''
    try {
      warning = await ReloadConfig()
    } catch (err) {
      flash(t('toast.reloadFailed', { error: String(err) }))
      return
    }
    await refreshAfterConfigChange()
    flash(warning ? t('toast.configReloadedWithWarning', { warning }) : t('toast.configReloaded'))
  }

  async function browseConfig() {
    let path = ''
    try {
      path = await BrowseConfig()
    } catch (err) {
      flash(t('toast.loadFailed', { error: String(err) }))
      return
    }
    if (!path) return // dialog cancelled
    await refreshAfterConfigChange()
    flash(t('toast.loaded', { path }))
  }

  async function launchConfigEditor() {
    try {
      const alreadyRunning = await LaunchConfigEditor()
      if (alreadyRunning) {
        flash(t('toast.configEditorAlreadyOpen'))
      }
    } catch (err) {
      flash(t('toast.openConfigEditorFailed', { error: String(err) }))
    }
  }

  function onKeyDown(e: KeyboardEvent) {
    if (e.key === 'F5') {
      e.preventDefault()
      reloadConfig()
    } else if ((e.ctrlKey || e.metaKey) && e.key.toLowerCase() === 'e') {
      e.preventDefault()
      launchConfigEditor()
    }
  }

  // --- Resizable / collapsible panel layout (geometry: lib/panelLayout) ---
  const LAYOUT_KEY = 'script-manager-gui:layout'

  let shellEl: HTMLElement
  let colLeftEl: HTMLElement
  let colRightEl: HTMLElement

  let leftWidth = 320
  let itemsHeight = 340
  let detailsHeight = 420

  let itemsCollapsed = false
  let actionsCollapsed = false
  let detailsCollapsed = false
  let commandCollapsed = false
  let groupChipsCollapsed = true
  let detailsWarningCollapsed = true
  let cmdSectionCollapsed = false
  let outputSectionCollapsed = false

  onMount(() => {
    // Defaults here are the effective first-run values, not necessarily the
    // `let` initializers above (e.g. group chips start expanded on a fresh
    // profile, matching the pre-loadPersisted `!!saved.groupChipsCollapsed`
    // coercion this replaced).
    ;({
      leftWidth,
      itemsHeight,
      detailsHeight,
      itemsCollapsed,
      actionsCollapsed,
      detailsCollapsed,
      commandCollapsed,
      groupChipsCollapsed,
      detailsWarningCollapsed,
      cmdSectionCollapsed,
      outputSectionCollapsed,
    } = loadPersisted(LAYOUT_KEY, {
      leftWidth: 320,
      itemsHeight: 340,
      detailsHeight: 420,
      itemsCollapsed: false,
      actionsCollapsed: false,
      detailsCollapsed: false,
      commandCollapsed: false,
      groupChipsCollapsed: false,
      detailsWarningCollapsed: true,
      cmdSectionCollapsed: false,
      outputSectionCollapsed: false,
    }))
  })

  function saveLayout() {
    savePersisted(LAYOUT_KEY, {
      leftWidth,
      itemsHeight,
      detailsHeight,
      itemsCollapsed,
      actionsCollapsed,
      detailsCollapsed,
      commandCollapsed,
      groupChipsCollapsed,
      detailsWarningCollapsed,
      cmdSectionCollapsed,
      outputSectionCollapsed,
    })
  }

  // The drag/flex geometry lives in lib/panelLayout — these wrappers just
  // bind it to this window's panels and persist the result.
  function dragLeftColumn(e: MouseEvent) {
    dragColumn(e, {
      getTotal: () => shellEl.getBoundingClientRect().width,
      get: () => leftWidth,
      set: (v) => (leftWidth = v),
      onDone: saveLayout,
    })
  }

  function dragItemsRow(e: MouseEvent) {
    if (itemsCollapsed || actionsCollapsed) return
    dragRow(e, {
      getTotal: () => colLeftEl.getBoundingClientRect().height,
      get: () => itemsHeight,
      set: (v) => (itemsHeight = v),
      onDone: saveLayout,
    })
  }

  function dragDetailsRow(e: MouseEvent) {
    if (detailsCollapsed || commandCollapsed) return
    dragRow(e, {
      getTotal: () => colRightEl.getBoundingClientRect().height,
      get: () => detailsHeight,
      set: (v) => (detailsHeight = v),
      onDone: saveLayout,
    })
  }

</script>

<svelte:window on:keydown={onKeyDown} />

<div class="app-root">
  <header class="toolbar">
    <IconButton title={t('tooltip.loadConfig')} on:click={browseConfig}><Icon name="load" /></IconButton>
    <IconButton title={t('tooltip.refreshConfigTitle')} aria={t('tooltip.refreshConfigAria')} on:click={reloadConfig}><Icon name="refresh" /></IconButton>
    <IconButton
      class="btn icon-btn settings-btn"
      title={t('tooltip.openConfigEditorTitle')}
      aria={t('tooltip.openConfigEditorAria')}
      on:click={launchConfigEditor}><Icon name="settings" /></IconButton
    >
  </header>
  <main class="app-shell" bind:this={shellEl}>
  <div class="col col-left" style="flex: 0 0 {leftWidth}px" bind:this={colLeftEl}>
    <Panel
      bind:collapsed={itemsCollapsed}
      title={t('panel.items')}
      titleWrap={itemsCollapsed}
      expandTitle={t('tooltip.expand')}
      collapseTitle={t('tooltip.collapse')}
      onToggle={saveLayout}
      style={topStyle(itemsCollapsed, actionsCollapsed, itemsHeight, true)}
      class="panel-items"
    >
      <svelte:fragment slot="title-extra">
        {#if itemsCollapsed && selectedItemLabel}<span class="panel-title-selected">{t('text.separator')}{selectedItemLabel}</span>{/if}
      </svelte:fragment>
      <div class="panel-body list">
        {#each items as item (item.index)}
          <button
            class="row"
            class:selected={item.index === selectedItem}
            on:click={() => selectItem(item.index)}
          >{item.label}{#if runningItemIndices.has(item.index)}<span class="running-indicator" title={t('tooltip.actionRunningItem')}>●</span>{/if}</button>
        {/each}
        {#if items.length === 0}
          <div class="empty">{t('empty.noItems')}</div>
        {/if}
      </div>
    </Panel>

    <!-- svelte-ignore a11y-no-static-element-interactions -->
    <div class="resizer horizontal" class:disabled={itemsCollapsed || actionsCollapsed} on:mousedown={dragItemsRow}></div>

    <Panel
      bind:collapsed={actionsCollapsed}
      title={t('panel.actions')}
      titleWrap={actionsCollapsed}
      expandTitle={t('tooltip.expand')}
      collapseTitle={t('tooltip.collapse')}
      onToggle={saveLayout}
      style={bottomStyle(actionsCollapsed, true)}
      class="panel-actions"
    >
      <svelte:fragment slot="title-extra">
        {#if actionsCollapsed && selectedActionLabel}<span class="panel-title-selected">{t('text.separator')}{selectedActionLabel}</span>{/if}
      </svelte:fragment>
      <GroupFilter
        {actions}
        {groupColors}
        bind:selectedGroups
        bind:collapsed={groupChipsCollapsed}
        onCollapseChange={saveLayout}
        onSelectionChange={onGroupFilterChange}
      />
      <div class="panel-body list">
        {#each filteredActions as action (action.index)}
          <button
            class="row"
            class:selected={action.index === selectedActionIndex}
            on:click={() => selectAction(action.index)}
          >{action.title}{#if runningActionIndicesForSelectedItem.has(action.index)}<span class="running-indicator" title={t('tooltip.actionRunningAction')}>●</span>{/if}</button>
        {/each}
        {#if selectedItem >= 0 && filteredActions.length === 0}
          <div class="empty">
            {selectedGroups.size > 0
              ? t('empty.noActionsForGroups', { plural: selectedGroups.size > 1 ? 's' : '' })
              : t('empty.noActionsForItem')}
          </div>
        {/if}
      </div>
    </Panel>
  </div>

  <!-- svelte-ignore a11y-no-static-element-interactions -->
  <div class="resizer vertical" on:mousedown={dragLeftColumn}></div>

  <div class="col col-right" bind:this={colRightEl}>
    <Panel
      bind:collapsed={detailsCollapsed}
      title={t('panel.details')}
      expandTitle={t('tooltip.expand')}
      collapseTitle={t('tooltip.collapse')}
      onToggle={saveLayout}
      style={topStyle(detailsCollapsed, commandCollapsed, detailsHeight)}
      class="panel-details"
    >
      {#if missingFields.length > 0}
        <div class="details-warning">
          <div class="details-warning-header">
            <CollapseToggle
              bind:collapsed={detailsWarningCollapsed}
              onToggle={saveLayout}
              class="warning-toggle"
              expandTitle={t('tooltip.expandMissingWarning')}
              collapseTitle={t('tooltip.collapseMissingWarning')}
            />
            <span class="warning-summary">
              {t('warning.missingFields', { count: missingFields.length, plural: missingFields.length > 1 ? 's' : '' })}
            </span>
          </div>
          {#if !detailsWarningCollapsed}
            <div class="warning-chips">
              {#each missingFields as field (field)}
                <span class="chip chip-static warning-chip">{field}</span>
              {/each}
            </div>
          {/if}
        </div>
      {/if}
      <!-- svelte-ignore a11y-click-events-have-key-events -->
      <!-- svelte-ignore a11y-no-static-element-interactions -->
      <div class="panel-body details-content" on:click={onDetailsClick}>
        {#if details?.html}
          {@html details.html}
        {:else}
          <div class="empty">{t('empty.noItemSelected')}</div>
        {/if}
      </div>
    </Panel>

    <!-- svelte-ignore a11y-no-static-element-interactions -->
    <div class="resizer horizontal" class:disabled={detailsCollapsed || commandCollapsed} on:mousedown={dragDetailsRow}></div>

    <Panel
      bind:collapsed={commandCollapsed}
      title={t('panel.command')}
      expandTitle={t('tooltip.expand')}
      collapseTitle={t('tooltip.collapse')}
      onToggle={saveLayout}
      style={bottomStyle(commandCollapsed)}
      class="panel-command"
    >
        <div class="panel-body command-content">
          {#if actionDetail}
            {#if actionDetail.cmd || actionDetail.script}
              <div class="cmd-actions">
                {#if !actionDetail.interactive}
                  <IconButton
                    class="run-cmd-btn icon-btn"
                    title={t('tooltip.runHere')}
                    disabled={inlineRunning}
                    on:click={runActionInline}><Icon name="run-here" /></IconButton
                  >
                {/if}
                <IconButton class="run-cmd-btn icon-btn" title={t('tooltip.run')} on:click={runAction}><Icon name="run" /></IconButton>
                {#if inlineRunning}
                  <IconButton class="copy-cmd-btn icon-btn" title={t('tooltip.cancel')} on:click={cancelInlineAction}><Icon name="cancel" /></IconButton>
                {/if}
              </div>
            {/if}
            {#if inlineRunning || inlineOutput || inlineExitCode !== null}
              <div class="messages-group">
                <button class="messages-group-header" type="button" on:click={() => { outputSectionCollapsed = !outputSectionCollapsed; saveLayout() }}>
                  <span class="messages-group-title">{t('section.output')}</span>
                  <span class="output-status">
                    {#if inlineRunning}
                      {t('text.running')}<span class="status-dot status-running">●</span>
                    {:else if inlineExitCode !== null}
                      {t('text.exitCode', { code: String(inlineExitCode) })}<span
                        class="status-dot"
                        class:status-ok={inlineExitCode === 0}
                        class:status-fail={inlineExitCode !== 0}>●</span
                      >
                    {/if}
                  </span>
                  <span class="collapse-glyph">{outputSectionCollapsed ? '▸' : '▾'}</span>
                </button>
                {#if !outputSectionCollapsed && inlineOutput}
                  <div class="cmd-output">
                    <IconButton
                      class="cmd-copy-btn cmd-output-copy-btn"
                      title={t('tooltip.copyOutput')}
                      on:click={() => copyToClipboard(inlineOutput)}><Icon name="copy" /></IconButton
                    >
                    <pre class="cmd-output-body" bind:this={inlineOutputEl}>{inlineOutput}</pre>
                  </div>
                {/if}
              </div>
            {/if}
            <div class="messages-group">
              <button class="messages-group-header" type="button" on:click={() => { cmdSectionCollapsed = !cmdSectionCollapsed; saveLayout() }}>
                <span class="messages-group-title">{t('section.command')}</span>
                <span class="collapse-glyph">{cmdSectionCollapsed ? '▸' : '▾'}</span>
              </button>
              {#if !cmdSectionCollapsed}
                {#if actionDetail.description}
                  <p class="cmd-desc">{actionDetail.description}</p>
                {/if}
                {#if selectedActionGroups.length > 0}
                  <div class="cmd-groups">
                    {#each selectedActionGroups as group (group)}
                      <span class="chip chip-static" style={groupChipStyle(groupColors, group, false)}>{group}</span>
                    {/each}
                  </div>
                {/if}
                {#if actionDetail.script}
                  <p class="cmd-desc">{t('text.scriptLabel')}{actionDetail.script}</p>
                  {#if actionDetail.scriptError}
                    <p class="cmd-error">{actionDetail.scriptError}</p>
                  {:else}
                    <ScriptSource content={actionDetail.scriptContent}>
                      <IconButton class="cmd-copy-btn cmd-line-copy-btn" title={t('tooltip.copyCommand')} on:click={copyCmd}><Icon name="copy" /></IconButton>
                    </ScriptSource>
                  {/if}
                {:else if actionDetail.cmd}
                  <ScriptSource content={actionDetail.cmd}>
                    <IconButton class="cmd-copy-btn cmd-line-copy-btn" title={t('tooltip.copyCommand')} on:click={copyCmd}><Icon name="copy" /></IconButton>
                  </ScriptSource>
                {/if}
              {/if}
            </div>
          {:else}
            <div class="empty">{t('empty.selectActionToPreview')}</div>
          {/if}
        </div>
    </Panel>
  </div>

    <Toast />
  </main>
</div>

<style>
  .app-root {
    display: flex;
    flex-direction: column;
    height: 100vh;
  }

  .toolbar {
    flex: none;
    display: flex;
    align-items: center;
    gap: 6px;
    padding: 6px 8px;
    background: var(--sm-panel-header);
    border-bottom: 1px solid var(--sm-border);
  }

  /* .icon-btn comes from the shared design system (@shared/theme.css),
     same as .btn. */

  /* Takes over the slot the theme dropdown used to occupy at the far
     right of the toolbar. */
  /* :global — this class now renders inside IconButton's own template
     (via its class prop), which Svelte's per-component CSS scoping
     wouldn't otherwise reach. */
  :global(.settings-btn) {
    margin-left: auto;
  }

  .app-shell {
    display: flex;
    flex-direction: row;
    flex: 1 1 auto;
    min-height: 0;
    box-sizing: border-box;
    padding: 8px;
    text-align: left;
  }

  .col {
    display: flex;
    flex-direction: column;
    min-height: 0;
    min-width: 0;
  }

  .col-right {
    flex: 1 1 auto;
  }

  /* .resizer(.vertical/.horizontal/.disabled), .panel, .panel-title(-text/-selected), .collapse-btn, .panel-body, .list,
     .row, .chip, .empty, .toast, .copy-cmd-btn, .run-cmd-btn come from the
     shared design system (@shared/theme.css, imported via style.css) — not
     redefined here. */

  .details-warning {
    flex: none;
    display: flex;
    flex-direction: column;
    gap: 4px;
    padding: 4px 6px;
    background: rgba(232, 163, 61, 0.1);
    border-bottom: 1px solid var(--sm-border);
  }

  .details-warning-header {
    display: flex;
    align-items: flex-start;
    gap: 4px;
  }

  /* :global, not scoped — the button this styles now renders inside
     CollapseToggle's own template (a different component), which
     Svelte's per-component CSS scoping wouldn't otherwise reach. */
  :global(.warning-toggle) {
    flex: none;
    padding: 2px 4px;
    color: var(--sm-warning);
  }

  .warning-summary {
    color: var(--sm-warning);
    font-size: 0.78rem;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .warning-chips {
    display: flex;
    flex-wrap: wrap;
    gap: 4px;
    padding-bottom: 2px;
  }

  .warning-chip {
    border-color: var(--sm-warning);
    color: var(--sm-warning);
  }

  .details-content {
    font-size: 0.9rem;
    line-height: 1.5;
  }

  .details-content :global(h1),
  .details-content :global(h2),
  .details-content :global(h3) {
    color: var(--sm-text-heading);
    margin: 0.6em 0 0.3em;
  }

  .details-content :global(table) {
    border-collapse: collapse;
    width: 100%;
  }

  .details-content :global(td),
  .details-content :global(th) {
    border: 1px solid var(--sm-border);
    padding: 4px 8px;
    text-align: left;
  }

  .details-content :global(code) {
    background: var(--sm-bg-deep);
    color: var(--sm-text-highlight);
    padding: 1px 5px;
    border-radius: 3px;
    font-family: "SF Mono", Consolas, monospace;
  }

  .details-content :global(code.copy-value) {
    cursor: pointer;
  }

  .details-content :global(code.copy-value:hover) {
    background: var(--sm-tint-hover);
    outline: 1px solid var(--sm-text-highlight);
  }

  .details-content :global(code.copy-value-masked) {
    color: var(--sm-masked);
  }

  .command-content {
    font-size: 0.85rem;
  }

  .cmd-desc {
    margin: 0 0 8px;
    color: var(--sm-text-muted);
    white-space: pre-wrap;
  }

  .cmd-error {
    margin: 0 0 8px;
    color: var(--sm-error);
  }

  .cmd-output {
    position: relative;
    background: var(--sm-bg-deep);
    border-radius: 4px;
    margin: 0 0 8px;
  }
  /* :global — these buttons now render inside IconButton's own template
     (via its class prop), which Svelte's per-component CSS scoping
     wouldn't otherwise reach. */
  :global(.cmd-output-copy-btn) {
    position: absolute;
    top: 4px;
    right: 4px;
  }
  .running-indicator {
    display: inline-block;
    margin-left: 6px;
    color: var(--sm-run-active);
    animation: running-pulse 1.5s ease-in-out infinite;
  }
  /* .output-status, .status-dot, .status-running/.status-ok/.status-fail
     (the OUTPUT section header's run status) come from the shared design
     system (@shared/theme.css) — also reused by the config editor's theme
     preview. */
  @keyframes running-pulse {
    0%, 100% { opacity: 1; }
    50% { opacity: 0.3; }
  }
  .cmd-output-body {
    margin: 0;
    padding: 10px;
    font-family: "SF Mono", Consolas, monospace;
    font-size: 0.8rem;
    white-space: pre-wrap;
    word-break: break-all;
    max-height: 260px;
    overflow-y: auto;
  }

  .cmd-groups {
    display: flex;
    flex-wrap: wrap;
    gap: 4px;
    margin: 0 0 8px;
  }

  /* Minimal, borderless copy button meant to sit inside a code block —
     .cmd-line-copy-btn and .cmd-output-copy-btn both float it in the
     top-right corner, the placement docs sites commonly use for a code
     block's copy action. .cmd-line-copy-btn positions against
     ScriptSource's own position:relative card (@shared/components/
     ScriptSource.svelte) — it's passed into that component's default slot,
     not rendered as a sibling here, but stays :global() since Svelte scopes
     slotted content to the component that fills the slot (this one), not
     the one that declares it. */
  :global(.cmd-copy-btn) {
    display: flex;
    align-items: center;
    justify-content: center;
    background: transparent;
    border: none;
    padding: 3px 5px;
    border-radius: 4px;
    color: var(--sm-text-muted);
    cursor: pointer;
  }
  :global(.cmd-copy-btn:hover) {
    background: var(--sm-overlay-soft);
    color: var(--sm-text);
  }
  :global(.cmd-line-copy-btn) {
    position: absolute;
    top: 4px;
    right: 4px;
  }

  .cmd-actions {
    position: sticky;
    top: 0;
    z-index: 2;
    display: flex;
    gap: 8px;
    margin: -6px -6px 8px;
    padding: 6px 6px 8px;
    background: var(--sm-bg-alt);
    box-shadow: 0 4px 6px -4px var(--sm-shadow);
  }

</style>
