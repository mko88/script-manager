<script lang="ts">
  import { onMount, tick } from 'svelte'
  import Toast from '@shared/components/Toast.svelte'
  import { getTheme, setTheme, type Theme } from '@shared/theme'
  import Icon from './components/Icon.svelte'
  import { t } from './messages'
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
    RunActionInline,
    CancelInlineAction,
    GetInlineStatus,
    LoadError,
    SetTheme,
  } from '../wailsjs/go/gui/App.js'
  import type { gui } from '../wailsjs/go/models'

  let items: gui.ItemDTO[] = []
  let actions: gui.ActionDTO[] = []
  let actionGroupCatalog: gui.ActionGroupDTO[] = []
  let details: gui.DetailsDTO | null = null
  let actionDetail: gui.ActionDetailDTO | null = null

  let selectedItem = -1
  let selectedActionIndex = -1
  // Empty set means "All" — no filter, show everything. Otherwise an action
  // matches if it belongs to any selected group (OR semantics).
  let selectedGroups = new Set<string>()
  // Group chips are sorted by exactly one key at a time: the active button
  // (name or count) owns the order. Clicking the active button flips its
  // direction; clicking the inactive one switches to that key, keeping the
  // direction it last had.
  let sortMode: 'alpha' | 'count' = 'alpha'
  let alphaDir: 'asc' | 'desc' = 'asc'
  let countDir: 'asc' | 'desc' = 'desc'
  let toast = ''
  let toastTimer: ReturnType<typeof setTimeout>

  let theme: Theme = getTheme()
  function toggleTheme() {
    theme = theme === 'dark' ? 'light' : 'dark'
    setTheme(theme)
    // Best-effort — the theme is already applied locally regardless of
    // whether this persists; see internal/theme for why it's shared.
    SetTheme(theme).catch(() => {})
  }

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

  // One entry per item/action pair that's ever been run inline in this
  // session, keyed by inlineKey(itemIndex, actionIndex) — not just the
  // single currently-viewed one, so a run started on one action keeps going
  // (and stays pollable) after switching to a different action, and
  // switching back shows however far it's gotten (or its finished result)
  // instead of losing track of it.
  type InlineState = {
    itemIndex: number
    actionIndex: number
    output: string
    running: boolean
    exitCode: number | null
  }
  let inlineStates: Record<string, InlineState> = {}

  function inlineKey(itemIndex: number, actionIndex: number): string {
    return `${itemIndex}:${actionIndex}`
  }

  function setInlineState(itemIndex: number, actionIndex: number, state: Omit<InlineState, 'itemIndex' | 'actionIndex'>) {
    inlineStates = { ...inlineStates, [inlineKey(itemIndex, actionIndex)]: { itemIndex, actionIndex, ...state } }
  }

  // What the Command pane actually displays — derived from inlineStates for
  // whatever's currently selected, defaulting to "never run" (blank, not
  // running) when there's no entry yet.
  $: currentInline = selectedItem >= 0 && selectedActionIndex >= 0 ? inlineStates[inlineKey(selectedItem, selectedActionIndex)] : undefined
  $: inlineRunning = currentInline?.running ?? false
  $: inlineOutput = currentInline?.output ?? ''
  $: inlineExitCode = currentInline?.exitCode ?? null

  // Which items/actions to show a running indicator for — every entry
  // still running, cross-referenced by itemIndex for the Items list and by
  // actionIndex (within the selected item) for the Actions list. Pure data
  // derivations, no DOM access — safe alongside the bug described above,
  // which was specifically about a reactive statement that touched the DOM.
  $: runningItemIndices = new Set(Object.values(inlineStates).filter((s) => s.running).map((s) => s.itemIndex))
  $: runningActionIndicesForSelectedItem = new Set(
    Object.values(inlineStates)
      .filter((s) => s.running && s.itemIndex === selectedItem)
      .map((s) => s.actionIndex),
  )

  // Unique groups across the current item's actions, in order of first
  // appearance — same set the TUI's [ / ] cycling walks.
  $: actionGroups = (() => {
    const seen = new Set<string>()
    const list: string[] = []
    for (const a of actions) {
      for (const g of a.groups ?? []) {
        if (!seen.has(g)) {
          seen.add(g)
          list.push(g)
        }
      }
    }
    return list
  })()

  $: filteredActions =
    selectedGroups.size === 0
      ? actions
      : actions.filter((a) => [...selectedGroups].every((g) => (a.groups ?? []).includes(g)))

  $: groupSummary = selectedGroups.size === 0 ? t('text.allGroupsChip') : actionGroups.filter((g) => selectedGroups.has(g)).join(', ')

  // For each group, how many actions would match if that group were added to
  // the current filter (AND semantics, same rule filteredActions applies) —
  // for an already-selected group this is just the current filtered count.
  $: groupCounts = (() => {
    const counts: Record<string, number> = {}
    for (const g of actionGroups) {
      const otherSelected = [...selectedGroups].filter((x) => x !== g)
      counts[g] = actions.filter(
        (a) => otherSelected.every((og) => (a.groups ?? []).includes(og)) && (a.groups ?? []).includes(g),
      ).length
    }
    return counts
  })()

  $: sortedGroups = [...actionGroups].sort((a, b) => {
    if (sortMode === 'alpha') {
      return a.localeCompare(b) * (alphaDir === 'asc' ? 1 : -1)
    }
    const countCmp = ((groupCounts[a] ?? 0) - (groupCounts[b] ?? 0)) * (countDir === 'asc' ? 1 : -1)
    if (countCmp !== 0) return countCmp
    // Equal counts need a deterministic order; plain A-Z, unaffected by the
    // (inactive) name button.
    return a.localeCompare(b)
  })

  // Hide chips that would narrow the filter to nothing — but never hide an
  // already-selected group, or there'd be no way left to deselect it short of
  // hitting "All".
  $: visibleGroups = sortedGroups.filter((g) => selectedGroups.has(g) || (groupCounts[g] ?? 0) > 0)

  $: alphaSortLabel = alphaDir === 'desc' ? t('sort.alphaDesc') : t('sort.alphaAsc')
  $: alphaSortTitle =
    sortMode !== 'alpha'
      ? t('sort.alphaTitleDefault')
      : alphaDir === 'asc'
        ? t('sort.alphaTitleAsc')
        : t('sort.alphaTitleDesc')
  $: countSortLabel = countDir === 'desc' ? t('sort.countDesc') : t('sort.countAsc')
  $: countSortTitle =
    sortMode !== 'count'
      ? t('sort.countTitleDefault')
      : countDir === 'desc'
        ? t('sort.countTitleDesc')
        : t('sort.countTitleAsc')

  $: missingFields = details?.missingFields ?? []

  $: selectedItemLabel = items.find((i) => i.index === selectedItem)?.label ?? ''
  $: selectedActionLabel = actions.find((a) => a.index === selectedActionIndex)?.title ?? ''
  $: selectedActionGroups = actions.find((a) => a.index === selectedActionIndex)?.groups ?? []

  // Only groups with a configured color get one here — a group with no
  // catalog entry (or an entry with no color set) just keeps the default
  // chip styling, so this feature is fully opt-in per group.
  $: groupColors = Object.fromEntries(
    actionGroupCatalog.filter((g) => /^#[0-9a-fA-F]{6}$/.test(g.color)).map((g) => [g.id, g.color]),
  ) as Record<string, string>

  function readableTextColor(hex: string): string {
    const r = parseInt(hex.slice(1, 3), 16)
    const g = parseInt(hex.slice(3, 5), 16)
    const b = parseInt(hex.slice(5, 7), 16)
    const brightness = (r * 299 + g * 587 + b * 114) / 1000
    return brightness > 128 ? '#1b2636' : '#d7dee8'
  }

  // Active/selected chips keep the existing accent-warm highlight regardless
  // of the group's own color, so "this filter is active" stays unambiguous.
  function groupChipStyle(group: string, active: boolean): string {
    const color = groupColors[group]
    if (active || !color) return ''
    return `background: ${color}; border-color: ${color}; color: ${readableTextColor(color)};`
  }

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

  function selectAllGroups() {
    selectedGroups = new Set()
    selectedActionIndex = -1
    actionDetail = null
  }

  function toggleGroup(group: string) {
    const next = new Set(selectedGroups)
    if (next.has(group)) next.delete(group)
    else next.add(group)
    selectedGroups = next
    selectedActionIndex = -1
    actionDetail = null
  }

  function toggleSort(axis: 'alpha' | 'count') {
    if (sortMode !== axis) {
      sortMode = axis
      return
    }
    if (axis === 'alpha') alphaDir = alphaDir === 'asc' ? 'desc' : 'asc'
    else countDir = countDir === 'asc' ? 'desc' : 'asc'
  }

  async function selectAction(index: number) {
    if (selectedItem < 0) return
    selectedActionIndex = index
    actionDetail = await GetActionDetail(selectedItem, index)
  }

  function flash(msg: string) {
    toast = msg
    clearTimeout(toastTimer)
    toastTimer = setTimeout(() => (toast = ''), 2000)
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
    if (!actionDetail?.cmd) return
    copyToClipboard(actionDetail.cmd)
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

  // How the frontend gets a live-updating view of an inline run: polling
  // GetInlineStatus on a short timer, not a pushed event — this app's other
  // bound methods are all plain request/response calls, and that's the
  // shape that's held up reliably here (see scrollInlineOutputToEnd's doc
  // comment above for the actual bug that made earlier streaming attempts
  // look unreliable — it wasn't Wails or this call shape at all).
  //
  // A poll loop keeps going until its own run finishes, regardless of
  // whether the user is still looking at that action — that's what makes
  // "switch away, switch back" work: inlineStates already has whatever
  // this loop has captured by the time the user returns, instead of the
  // loop having given up and stopped tracking it. It only skips the
  // scroll-into-view side effect while its action isn't the one on screen.
  const INLINE_POLL_INTERVAL_MS = 300

  async function pollInlineStatus(itemIndex: number, actionIndex: number) {
    for (;;) {
      const status = await GetInlineStatus(itemIndex, actionIndex)
      setInlineState(itemIndex, actionIndex, { output: status.output, running: status.running, exitCode: status.exitCode })
      if (selectedItem === itemIndex && selectedActionIndex === actionIndex) {
        scrollInlineOutputToEnd()
      }
      if (status.running) {
        await new Promise((resolve) => setTimeout(resolve, INLINE_POLL_INTERVAL_MS))
        continue
      }
      if (status.errMsg) flash(t('toast.runFailed', { error: status.errMsg }))
      return
    }
  }

  async function runActionInline() {
    if (selectedItem < 0 || selectedActionIndex < 0) return
    const itemIndex = selectedItem
    const actionIndex = selectedActionIndex
    if (inlineStates[inlineKey(itemIndex, actionIndex)]?.running) return
    setInlineState(itemIndex, actionIndex, { output: '', running: true, exitCode: null })
    try {
      await RunActionInline(itemIndex, actionIndex)
      pollInlineStatus(itemIndex, actionIndex)
    } catch (err) {
      setInlineState(itemIndex, actionIndex, { output: '', running: false, exitCode: null })
      flash(t('toast.runFailed', { error: String(err) }))
    }
  }

  async function cancelInlineAction() {
    if (selectedItem < 0 || selectedActionIndex < 0) return
    try {
      await CancelInlineAction(selectedItem, selectedActionIndex)
    } catch (err) {
      flash(t('toast.cancelFailed', { error: String(err) }))
    }
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

  // --- Resizable / collapsible panel layout ---
  const LAYOUT_KEY = 'script-manager-gui:layout'
  const HEADER_H = 33
  const MIN_PANEL = 60
  const MIN_COL = 180
  const RESIZER = 6

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

  onMount(() => {
    try {
      const saved = JSON.parse(localStorage.getItem(LAYOUT_KEY) ?? '{}')
      if (typeof saved.leftWidth === 'number') leftWidth = saved.leftWidth
      if (typeof saved.itemsHeight === 'number') itemsHeight = saved.itemsHeight
      if (typeof saved.detailsHeight === 'number') detailsHeight = saved.detailsHeight
      itemsCollapsed = !!saved.itemsCollapsed
      actionsCollapsed = !!saved.actionsCollapsed
      detailsCollapsed = !!saved.detailsCollapsed
      commandCollapsed = !!saved.commandCollapsed
      groupChipsCollapsed = !!saved.groupChipsCollapsed
      detailsWarningCollapsed = saved.detailsWarningCollapsed ?? true
    } catch {
      // ignore corrupt/missing layout, defaults already set
    }
  })

  function saveLayout() {
    localStorage.setItem(
      LAYOUT_KEY,
      JSON.stringify({
        leftWidth,
        itemsHeight,
        detailsHeight,
        itemsCollapsed,
        actionsCollapsed,
        detailsCollapsed,
        commandCollapsed,
        groupChipsCollapsed,
        detailsWarningCollapsed,
      }),
    )
  }

  function toggleGroupChips() {
    groupChipsCollapsed = !groupChipsCollapsed
    saveLayout()
  }

  function toggleDetailsWarning() {
    detailsWarningCollapsed = !detailsWarningCollapsed
    saveLayout()
  }

  function dragColumn(e: MouseEvent) {
    e.preventDefault()
    const startX = e.clientX
    const startWidth = leftWidth
    function onMove(ev: MouseEvent) {
      const total = shellEl.getBoundingClientRect().width
      const max = total - MIN_COL - RESIZER
      leftWidth = Math.min(max, Math.max(MIN_COL, startWidth + (ev.clientX - startX)))
    }
    function onUp() {
      window.removeEventListener('mousemove', onMove)
      window.removeEventListener('mouseup', onUp)
      saveLayout()
    }
    window.addEventListener('mousemove', onMove)
    window.addEventListener('mouseup', onUp)
  }

  function dragRow(e: MouseEvent, get: () => number, set: (v: number) => void, containerEl: () => HTMLElement) {
    e.preventDefault()
    const startY = e.clientY
    const startH = get()
    function onMove(ev: MouseEvent) {
      const total = containerEl().getBoundingClientRect().height
      const max = total - MIN_PANEL - RESIZER - HEADER_H
      set(Math.min(max, Math.max(MIN_PANEL, startH + (ev.clientY - startY))))
    }
    function onUp() {
      window.removeEventListener('mousemove', onMove)
      window.removeEventListener('mouseup', onUp)
      saveLayout()
    }
    window.addEventListener('mousemove', onMove)
    window.addEventListener('mouseup', onUp)
  }

  function dragItemsRow(e: MouseEvent) {
    if (itemsCollapsed || actionsCollapsed) return
    dragRow(e, () => itemsHeight, (v) => (itemsHeight = v), () => colLeftEl)
  }
  function dragDetailsRow(e: MouseEvent) {
    if (detailsCollapsed || commandCollapsed) return
    dragRow(e, () => detailsHeight, (v) => (detailsHeight = v), () => colRightEl)
  }

  function toggleCollapse(which: 'items' | 'actions' | 'details' | 'command') {
    if (which === 'items') itemsCollapsed = !itemsCollapsed
    else if (which === 'actions') actionsCollapsed = !actionsCollapsed
    else if (which === 'details') detailsCollapsed = !detailsCollapsed
    else commandCollapsed = !commandCollapsed
    saveLayout()
  }

  // The "top" panel in a pair (Items/Details) gets an explicit height; the
  // "bottom" panel (Actions/Command) fills whatever space is left. Collapsing
  // either one just swaps who gets the fixed header-only height. Panels whose
  // collapsed header shows a selected-item label that can wrap onto multiple
  // lines (Items, Actions) get an auto flex-basis instead of the fixed
  // HEADER_H so the wrapped text isn't clipped.
  function topStyle(topCollapsed: boolean, bottomCollapsed: boolean, size: number, autoCollapse = false) {
    if (topCollapsed) return autoCollapse ? `flex: 0 0 auto;` : `flex: 0 0 ${HEADER_H}px;`
    if (bottomCollapsed) return `flex: 1 1 auto;`
    return `flex: 0 0 ${size}px;`
  }
  function bottomStyle(bottomCollapsed: boolean, autoCollapse = false) {
    if (bottomCollapsed) return autoCollapse ? `flex: 0 0 auto;` : `flex: 0 0 ${HEADER_H}px;`
    return `flex: 1 1 auto; min-height: 0;`
  }
</script>

<svelte:window on:keydown={onKeyDown} />

<div class="app-root">
  <header class="toolbar">
    <button class="btn icon-btn" type="button" title={t('tooltip.loadConfig')} aria-label={t('tooltip.loadConfig')} on:click={browseConfig}
      ><Icon name="load" /></button
    >
    <button
      class="btn icon-btn"
      type="button"
      title={t('tooltip.refreshConfigTitle')}
      aria-label={t('tooltip.refreshConfigAria')}
      on:click={reloadConfig}><Icon name="refresh" /></button
    >
    <button
      class="btn icon-btn"
      type="button"
      title={t('tooltip.openConfigEditorTitle')}
      aria-label={t('tooltip.openConfigEditorAria')}
      on:click={launchConfigEditor}><Icon name="config-edit" /></button
    >
    <button
      class="btn icon-btn theme-toggle-btn"
      type="button"
      title={theme === 'dark' ? t('tooltip.themeLight') : t('tooltip.themeDark')}
      aria-label={theme === 'dark' ? t('tooltip.themeLight') : t('tooltip.themeDark')}
      on:click={toggleTheme}><Icon name={theme === 'dark' ? 'theme-dark' : 'theme-light'} /></button
    >
  </header>
  <main class="app-shell" bind:this={shellEl}>
  <div class="col col-left" style="flex: 0 0 {leftWidth}px" bind:this={colLeftEl}>
    <section class="panel panel-items" style={topStyle(itemsCollapsed, actionsCollapsed, itemsHeight, true)}>
      <header class="panel-title">
        <span class="panel-title-text" class:wrap={itemsCollapsed}>
          {t('panel.items')}{#if itemsCollapsed && selectedItemLabel}<span class="panel-title-selected">{t('text.separator')}{selectedItemLabel}</span>{/if}
        </span>
        <button class="collapse-btn" on:click={() => toggleCollapse('items')} title={itemsCollapsed ? t('tooltip.expand') : t('tooltip.collapse')}>
          {itemsCollapsed ? '▸' : '▾'}
        </button>
      </header>
      {#if !itemsCollapsed}
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
      {/if}
    </section>

    <!-- svelte-ignore a11y-no-static-element-interactions -->
    <div class="resizer horizontal" class:disabled={itemsCollapsed || actionsCollapsed} on:mousedown={dragItemsRow}></div>

    <section class="panel panel-actions" style={bottomStyle(actionsCollapsed, true)}>
      <header class="panel-title">
        <span class="panel-title-text" class:wrap={actionsCollapsed}>
          {t('panel.actions')}{#if actionsCollapsed && selectedActionLabel}<span class="panel-title-selected">{t('text.separator')}{selectedActionLabel}</span>{/if}
        </span>
        <button class="collapse-btn" on:click={() => toggleCollapse('actions')} title={actionsCollapsed ? t('tooltip.expand') : t('tooltip.collapse')}>
          {actionsCollapsed ? '▸' : '▾'}
        </button>
      </header>
      {#if !actionsCollapsed}
        {#if actionGroups.length > 0}
          <div class="group-filter">
            <div class="group-filter-header">
              <button
                class="collapse-btn group-filter-toggle"
                on:click={toggleGroupChips}
                title={groupChipsCollapsed ? t('tooltip.expandGroupFilter') : t('tooltip.collapseGroupFilter')}
              >
                {groupChipsCollapsed ? '▸' : '▾'}
              </button>
              {#if groupChipsCollapsed}
                <span class="group-summary">{groupSummary}</span>
              {:else}
                <div class="group-sort">
                  <button
                    class="sort-btn"
                    class:active={sortMode === 'alpha'}
                    on:click={() => toggleSort('alpha')}
                    title={alphaSortTitle}
                  >
                    {alphaSortLabel}
                  </button>
                  <button
                    class="sort-btn"
                    class:active={sortMode === 'count'}
                    on:click={() => toggleSort('count')}
                    title={countSortTitle}
                  >
                    {countSortLabel}
                  </button>
                </div>
              {/if}
            </div>
            {#if !groupChipsCollapsed}
              <div class="group-chips">
                <button class="chip" class:active={selectedGroups.size === 0} on:click={selectAllGroups}>{t('text.allGroupsChip')}</button>
                {#each visibleGroups as group (group)}
                  <button
                    class="chip"
                    class:active={selectedGroups.has(group)}
                    style={groupChipStyle(group, selectedGroups.has(group))}
                    on:click={() => toggleGroup(group)}
                    >{group}<span class="chip-count">{t('text.groupCount', { count: groupCounts[group] ?? 0 })}</span></button
                  >
                {/each}
              </div>
            {/if}
          </div>
        {/if}
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
      {/if}
    </section>
  </div>

  <!-- svelte-ignore a11y-no-static-element-interactions -->
  <div class="resizer vertical" on:mousedown={dragColumn}></div>

  <div class="col col-right" bind:this={colRightEl}>
    <section class="panel panel-details" style={topStyle(detailsCollapsed, commandCollapsed, detailsHeight)}>
      <header class="panel-title">
        <span>{t('panel.details')}</span>
        <button class="collapse-btn" on:click={() => toggleCollapse('details')} title={detailsCollapsed ? t('tooltip.expand') : t('tooltip.collapse')}>
          {detailsCollapsed ? '▸' : '▾'}
        </button>
      </header>
      {#if !detailsCollapsed}
        {#if missingFields.length > 0}
          <div class="details-warning">
            <div class="details-warning-header">
              <button
                class="collapse-btn warning-toggle"
                on:click={toggleDetailsWarning}
                title={detailsWarningCollapsed ? t('tooltip.expandMissingWarning') : t('tooltip.collapseMissingWarning')}
              >
                {detailsWarningCollapsed ? '▸' : '▾'}
              </button>
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
      {/if}
    </section>

    <!-- svelte-ignore a11y-no-static-element-interactions -->
    <div class="resizer horizontal" class:disabled={detailsCollapsed || commandCollapsed} on:mousedown={dragDetailsRow}></div>

    <section class="panel panel-command" style={bottomStyle(commandCollapsed)}>
      <header class="panel-title">
        <span>{t('panel.command')}</span>
        <button class="collapse-btn" on:click={() => toggleCollapse('command')} title={commandCollapsed ? t('tooltip.expand') : t('tooltip.collapse')}>
          {commandCollapsed ? '▸' : '▾'}
        </button>
      </header>
      {#if !commandCollapsed}
        <div class="panel-body command-content">
          {#if actionDetail}
            {#if actionDetail.cmd}
              <div class="cmd-actions">
                {#if !actionDetail.interactive}
                  <button
                    class="run-cmd-btn icon-btn"
                    title={t('tooltip.runHere')}
                    aria-label={t('tooltip.runHere')}
                    disabled={inlineRunning}
                    on:click={runActionInline}><Icon name="run-here" /></button
                  >
                {/if}
                <button class="run-cmd-btn icon-btn" title={t('tooltip.run')} aria-label={t('tooltip.run')} on:click={runAction}
                  ><Icon name="run" /></button
                >
                {#if inlineRunning}
                  <button class="copy-cmd-btn icon-btn" title={t('tooltip.cancel')} aria-label={t('tooltip.cancel')} on:click={cancelInlineAction}
                    ><Icon name="cancel" /></button
                  >
                {/if}
              </div>
            {/if}
            {#if inlineRunning || inlineOutput}
              <div class="cmd-output">
                <div class="cmd-output-status" class:running={inlineRunning} class:error={inlineExitCode !== null && inlineExitCode !== 0}>
                  <span>{inlineRunning ? t('status.running') : t('status.exited', { code: String(inlineExitCode) })}</span>
                  {#if inlineOutput}
                    <button
                      class="cmd-copy-btn"
                      title={t('tooltip.copyOutput')}
                      aria-label={t('tooltip.copyOutput')}
                      on:click={() => copyToClipboard(inlineOutput)}><Icon name="copy" /></button
                    >
                  {/if}
                </div>
                {#if inlineOutput}
                  <pre class="cmd-output-body" bind:this={inlineOutputEl}>{inlineOutput}</pre>
                {/if}
              </div>
            {/if}
            {#if actionDetail.description}
              <p class="cmd-desc">{actionDetail.description}</p>
            {/if}
            {#if selectedActionGroups.length > 0}
              <div class="cmd-groups">
                {#each selectedActionGroups as group (group)}
                  <span class="chip chip-static" style={groupChipStyle(group, false)}>{group}</span>
                {/each}
              </div>
            {/if}
            {#if actionDetail.cmd}
              <div class="cmd-line">
                <button class="cmd-copy-btn cmd-line-copy-btn" title={t('tooltip.copyCommand')} aria-label={t('tooltip.copyCommand')} on:click={copyCmd}
                  ><Icon name="copy" /></button
                >
                {#each actionDetail.cmd.replace(/\n+$/, '').split('\n') as line, i (i)}
                  <div class="cmd-line-row">
                    <span class="cmd-line-no">{i + 1}</span>
                    <span class="cmd-line-text">{line}</span>
                  </div>
                {/each}
              </div>
            {/if}
          {:else}
            <div class="empty">{t('empty.selectActionToPreview')}</div>
          {/if}
        </div>
      {/if}
    </section>
  </div>

    <Toast message={toast} />
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

  /* Square padding for an icon-only .btn — theme.css's own .btn/.btn-primary
     assume a text label; not shared since sm-config-edit scopes the same
     rule locally rather than in theme.css. */
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

  .theme-toggle-btn {
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

  .group-filter {
    flex: none;
    display: flex;
    flex-direction: column;
    gap: 4px;
    padding: 4px 6px 0;
  }

  .group-filter-header {
    display: flex;
    align-items: flex-start;
    gap: 4px;
  }

  .group-filter-toggle {
    flex: none;
    padding: 2px 4px;
  }

  .group-summary {
    color: var(--sm-text-muted);
    font-size: 0.78rem;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .group-sort {
    flex: none;
    display: flex;
    gap: 2px;
  }

  .sort-btn {
    flex: none;
    background: none;
    border: 1px solid var(--sm-border);
    color: var(--sm-text-muted);
    border-radius: 4px;
    padding: 2px 6px;
    font-size: 0.68rem;
    line-height: 1.2;
    cursor: pointer;
    font-family: inherit;
  }

  .sort-btn:hover {
    background: var(--sm-hover);
    color: var(--sm-text);
  }

  .sort-btn.active {
    background: var(--sm-accent-warm);
    border-color: var(--sm-accent-warm);
    color: var(--sm-bg);
    font-weight: 700;
  }

  .group-chips {
    display: flex;
    flex-wrap: wrap;
    gap: 4px;
  }

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

  .warning-toggle {
    flex: none;
    padding: 2px 4px;
    color: var(--sm-accent-warm);
  }

  .warning-summary {
    color: var(--sm-accent-warm);
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
    border-color: var(--sm-accent-warm);
    color: var(--sm-accent-warm);
  }

  .details-content {
    font-size: 0.9rem;
    line-height: 1.5;
  }

  .details-content :global(h1),
  .details-content :global(h2),
  .details-content :global(h3) {
    color: var(--sm-accent);
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
    color: var(--sm-code);
    padding: 1px 5px;
    border-radius: 3px;
    font-family: "SF Mono", Consolas, monospace;
  }

  .details-content :global(code.copy-value) {
    cursor: pointer;
  }

  .details-content :global(code.copy-value:hover) {
    background: var(--sm-tint-hover);
    outline: 1px solid var(--sm-code);
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

  .cmd-output {
    background: var(--sm-bg-deep);
    border-radius: 4px;
    margin: 0 0 8px;
  }
  .cmd-output-status {
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding: 6px 10px;
    font-size: 0.85rem;
    color: var(--sm-text-muted);
  }
  .cmd-output-status.running {
    color: var(--sm-accent);
  }
  .cmd-output-status.error {
    color: var(--sm-error);
  }
  .running-indicator {
    display: inline-block;
    margin-left: 6px;
    color: var(--sm-accent);
    animation: running-pulse 1.5s ease-in-out infinite;
  }
  @keyframes running-pulse {
    0%, 100% { opacity: 1; }
    50% { opacity: 0.3; }
  }
  .cmd-output-body {
    margin: 0;
    padding: 0 10px 10px;
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

  .cmd-line {
    position: relative;
    background: var(--sm-bg-deep);
    border-radius: 4px;
    padding: 8px 0;
    margin: 0 0 8px;
    color: var(--sm-text);
    font-family: "SF Mono", Consolas, monospace;
    font-size: 0.85rem;
  }

  /* Minimal, borderless copy button meant to sit inside a code block —
     .cmd-line-copy-btn additionally floats it in the top-right corner,
     the placement docs sites commonly use for a code block's copy action;
     .cmd-output-status's copy button instead just sits at the end of that
     flex row (space-between above), no absolute positioning needed there. */
  .cmd-copy-btn {
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
  .cmd-copy-btn:hover {
    background: var(--sm-overlay-soft);
    color: var(--sm-text);
  }
  .cmd-line-copy-btn {
    position: absolute;
    top: 4px;
    right: 4px;
  }

  .cmd-line-row {
    display: flex;
    gap: 10px;
    padding: 0 8px;
  }

  .cmd-line-no {
    flex: none;
    width: 1.6em;
    text-align: right;
    color: var(--sm-line-number);
    user-select: none;
  }

  .cmd-line-text {
    flex: 1;
    min-width: 0;
    white-space: pre-wrap;
    word-break: break-word;
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
