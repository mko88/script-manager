<script lang="ts">
  import { onMount } from 'svelte'
  import Toast from '@shared/components/Toast.svelte'
  import {
    GetTitles,
    GetItems,
    GetActions,
    GetActionDetail,
    GetItemDetails,
    GetActionGroups,
    CopyToClipboard,
    ReloadConfig,
    RunAction,
    LoadError,
  } from '../wailsjs/go/gui/App.js'
  import type { gui } from '../wailsjs/go/models'

  let titles: gui.TitlesDTO = { items: 'Items', actions: 'Actions', details: 'Details', command: 'Command' }
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

  $: groupSummary = selectedGroups.size === 0 ? 'All' : actionGroups.filter((g) => selectedGroups.has(g)).join(', ')

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

  $: alphaSortLabel = alphaDir === 'desc' ? 'Z-A' : 'A-Z'
  $: alphaSortTitle =
    sortMode !== 'alpha'
      ? 'Sort by name'
      : alphaDir === 'asc'
        ? 'Sorted A-Z — click for Z-A'
        : 'Sorted Z-A — click for A-Z'
  $: countSortLabel = countDir === 'desc' ? '# ↓' : '# ↑'
  $: countSortTitle =
    sortMode !== 'count'
      ? 'Sort by action count'
      : countDir === 'desc'
        ? 'Sorted by action count, descending — click for ascending'
        : 'Sorted by action count, ascending — click for descending'

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
    if (loadErr) flash(`Config load failed: ${loadErr}`)
    titles = await GetTitles()
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

  async function copyToClipboard(value: string, successMsg: string) {
    try {
      await CopyToClipboard(value)
      flash(successMsg)
    } catch (err) {
      flash(`Clipboard unavailable: ${err}`)
    }
  }

  function copyValue(idx: number) {
    if (!details) return
    const value = details.copyValues[idx]
    if (value === undefined) return
    copyToClipboard(value, details.copyMasked[idx] ? 'Copied masked value to clipboard' : `Copied: ${value}`)
  }

  function onDetailsClick(e: MouseEvent) {
    const target = (e.target as HTMLElement).closest('[data-copy-idx]') as HTMLElement | null
    if (!target) return
    copyValue(Number(target.dataset.copyIdx))
  }

  function copyCmd() {
    if (!actionDetail?.cmd) return
    copyToClipboard(actionDetail.cmd, 'Command copied to clipboard')
  }

  async function runAction() {
    if (selectedItem < 0 || selectedActionIndex < 0) return
    try {
      await RunAction(selectedItem, selectedActionIndex)
      flash('Running in terminal window…')
    } catch (err) {
      flash(`Run failed: ${err}`)
    }
  }

  async function reloadConfig() {
    let warning = ''
    try {
      warning = await ReloadConfig()
    } catch (err) {
      flash(`Reload failed: ${err}`)
      return
    }
    titles = await GetTitles()
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
    flash(warning ? `Config reloaded with a warning: ${warning}` : 'Config reloaded')
  }

  function onKeyDown(e: KeyboardEvent) {
    if (e.key === 'F5') {
      e.preventDefault()
      reloadConfig()
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

<main class="app-shell" bind:this={shellEl}>
  <div class="col col-left" style="flex: 0 0 {leftWidth}px" bind:this={colLeftEl}>
    <section class="panel panel-items" style={topStyle(itemsCollapsed, actionsCollapsed, itemsHeight, true)}>
      <header class="panel-title">
        <span class="panel-title-text" class:wrap={itemsCollapsed}>
          {titles.items}{#if itemsCollapsed && selectedItemLabel}<span class="panel-title-selected"> · {selectedItemLabel}</span>{/if}
        </span>
        <button class="collapse-btn" on:click={() => toggleCollapse('items')} title={itemsCollapsed ? 'Expand' : 'Collapse'}>
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
            >{item.label}</button>
          {/each}
          {#if items.length === 0}
            <div class="empty">No items configured</div>
          {/if}
        </div>
      {/if}
    </section>

    <!-- svelte-ignore a11y-no-static-element-interactions -->
    <div class="resizer horizontal" class:disabled={itemsCollapsed || actionsCollapsed} on:mousedown={dragItemsRow}></div>

    <section class="panel panel-actions" style={bottomStyle(actionsCollapsed, true)}>
      <header class="panel-title">
        <span class="panel-title-text" class:wrap={actionsCollapsed}>
          {titles.actions}{#if actionsCollapsed && selectedActionLabel}<span class="panel-title-selected"> · {selectedActionLabel}</span>{/if}
        </span>
        <button class="collapse-btn" on:click={() => toggleCollapse('actions')} title={actionsCollapsed ? 'Expand' : 'Collapse'}>
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
                title={groupChipsCollapsed ? 'Expand group filter' : 'Collapse group filter'}
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
                <button class="chip" class:active={selectedGroups.size === 0} on:click={selectAllGroups}>All</button>
                {#each visibleGroups as group (group)}
                  <button
                    class="chip"
                    class:active={selectedGroups.has(group)}
                    style={groupChipStyle(group, selectedGroups.has(group))}
                    on:click={() => toggleGroup(group)}
                    >{group}<span class="chip-count">({groupCounts[group] ?? 0})</span></button
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
            >{action.title}</button>
          {/each}
          {#if selectedItem >= 0 && filteredActions.length === 0}
            <div class="empty">
              {selectedGroups.size > 0 ? `No actions in the selected group${selectedGroups.size > 1 ? 's' : ''}` : 'No actions for this item'}
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
        <span>{titles.details}</span>
        <button class="collapse-btn" on:click={() => toggleCollapse('details')} title={detailsCollapsed ? 'Expand' : 'Collapse'}>
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
                title={detailsWarningCollapsed ? 'Expand missing-values warning' : 'Collapse missing-values warning'}
              >
                {detailsWarningCollapsed ? '▸' : '▾'}
              </button>
              <span class="warning-summary">
                ⚠ {missingFields.length} missing value{missingFields.length > 1 ? 's' : ''} (shown as &lt;nil&gt;)
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
            <div class="empty">No item selected</div>
          {/if}
        </div>
      {/if}
    </section>

    <!-- svelte-ignore a11y-no-static-element-interactions -->
    <div class="resizer horizontal" class:disabled={detailsCollapsed || commandCollapsed} on:mousedown={dragDetailsRow}></div>

    <section class="panel panel-command" style={bottomStyle(commandCollapsed)}>
      <header class="panel-title">
        <span>{titles.command}</span>
        <button class="collapse-btn" on:click={() => toggleCollapse('command')} title={commandCollapsed ? 'Expand' : 'Collapse'}>
          {commandCollapsed ? '▸' : '▾'}
        </button>
      </header>
      {#if !commandCollapsed}
        <div class="panel-body command-content">
          {#if actionDetail}
            {#if actionDetail.cmd}
              <div class="cmd-actions">
                <button class="run-cmd-btn" on:click={runAction}>Run</button>
                <button class="copy-cmd-btn" on:click={copyCmd}>Copy command</button>
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
                {#each actionDetail.cmd.replace(/\n+$/, '').split('\n') as line, i (i)}
                  <div class="cmd-line-row">
                    <span class="cmd-line-no">{i + 1}</span>
                    <span class="cmd-line-text">{line}</span>
                  </div>
                {/each}
              </div>
            {/if}
          {:else}
            <div class="empty">Select an action to preview its command</div>
          {/if}
        </div>
      {/if}
    </section>
  </div>

  <Toast message={toast} />
</main>

<style>
  .app-shell {
    display: flex;
    flex-direction: row;
    height: 100vh;
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
    color: #a9b6c8;
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
    border: 1px solid #3a4a63;
    color: #a9b6c8;
    border-radius: 4px;
    padding: 2px 6px;
    font-size: 0.68rem;
    line-height: 1.2;
    cursor: pointer;
    font-family: inherit;
  }

  .sort-btn:hover {
    background: #2b3b52;
    color: #d7dee8;
  }

  .sort-btn.active {
    background: #e8a33d;
    border-color: #e8a33d;
    color: #1b2636;
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
    border-bottom: 1px solid #3a4a63;
  }

  .details-warning-header {
    display: flex;
    align-items: flex-start;
    gap: 4px;
  }

  .warning-toggle {
    flex: none;
    padding: 2px 4px;
    color: #e8a33d;
  }

  .warning-summary {
    color: #e8a33d;
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
    border-color: #e8a33d;
    color: #e8a33d;
  }

  .details-content {
    font-size: 0.9rem;
    line-height: 1.5;
  }

  .details-content :global(h1),
  .details-content :global(h2),
  .details-content :global(h3) {
    color: #7fd4ff;
    margin: 0.6em 0 0.3em;
  }

  .details-content :global(table) {
    border-collapse: collapse;
    width: 100%;
  }

  .details-content :global(td),
  .details-content :global(th) {
    border: 1px solid #3a4a63;
    padding: 4px 8px;
    text-align: left;
  }

  .details-content :global(code) {
    background: #14202f;
    color: #6ee7d8;
    padding: 1px 5px;
    border-radius: 3px;
    font-family: "SF Mono", Consolas, monospace;
  }

  .details-content :global(code.copy-value) {
    cursor: pointer;
  }

  .details-content :global(code.copy-value:hover) {
    background: #1f3346;
    outline: 1px solid #6ee7d8;
  }

  .details-content :global(code.copy-value-masked) {
    color: #b9c3d1;
  }

  .command-content {
    font-size: 0.85rem;
  }

  .cmd-desc {
    margin: 0 0 8px;
    color: #a9b6c8;
    white-space: pre-wrap;
  }

  .cmd-groups {
    display: flex;
    flex-wrap: wrap;
    gap: 4px;
    margin: 0 0 8px;
  }

  .cmd-line {
    background: #14202f;
    border-radius: 4px;
    padding: 8px 0;
    margin: 0 0 8px;
    color: #d7dee8;
    font-family: "SF Mono", Consolas, monospace;
    font-size: 0.85rem;
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
    color: #4a5b74;
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
    background: #1f2c3d;
    box-shadow: 0 4px 6px -4px rgba(0, 0, 0, 0.5);
  }

</style>
