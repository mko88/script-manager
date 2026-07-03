<script lang="ts">
  import { onMount } from 'svelte'
  import {
    GetTitles,
    GetItems,
    GetActions,
    GetActionDetail,
    GetItemDetails,
    CopyToClipboard,
    ReloadConfig,
    RunAction,
  } from '../wailsjs/go/main/App.js'
  import type { main } from '../wailsjs/go/models'

  let titles: main.TitlesDTO = { items: 'Items', actions: 'Actions', details: 'Details' }
  let items: main.ItemDTO[] = []
  let actions: main.ActionDTO[] = []
  let details: main.DetailsDTO | null = null
  let actionDetail: main.ActionDetailDTO | null = null

  let selectedItem = -1
  let selectedActionIndex = -1
  // Empty set means "All" — no filter, show everything. Otherwise an action
  // matches if it belongs to any selected group (OR semantics).
  let selectedGroups = new Set<string>()
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

  $: selectedItemLabel = items.find((i) => i.index === selectedItem)?.label ?? ''
  $: selectedActionLabel = actions.find((a) => a.index === selectedActionIndex)?.title ?? ''

  onMount(async () => {
    titles = await GetTitles()
    items = await GetItems()
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
      flash('Running in Windows Terminal…')
    } catch (err) {
      flash(`Run failed: ${err}`)
    }
  }

  async function reloadConfig() {
    try {
      await ReloadConfig()
    } catch (err) {
      flash(`Reload failed: ${err}`)
      return
    }
    titles = await GetTitles()
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
    flash('Config reloaded')
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
      }),
    )
  }

  function toggleGroupChips() {
    groupChipsCollapsed = !groupChipsCollapsed
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
  // either one just swaps who gets the fixed header-only height.
  function topStyle(topCollapsed: boolean, bottomCollapsed: boolean, size: number) {
    if (topCollapsed) return `flex: 0 0 ${HEADER_H}px;`
    if (bottomCollapsed) return `flex: 1 1 auto;`
    return `flex: 0 0 ${size}px;`
  }
  function bottomStyle(bottomCollapsed: boolean) {
    return bottomCollapsed ? `flex: 0 0 ${HEADER_H}px;` : `flex: 1 1 auto; min-height: 0;`
  }
</script>

<svelte:window on:keydown={onKeyDown} />

<main class="app-shell" bind:this={shellEl}>
  <div class="col col-left" style="flex: 0 0 {leftWidth}px" bind:this={colLeftEl}>
    <section class="panel panel-items" style={topStyle(itemsCollapsed, actionsCollapsed, itemsHeight)}>
      <header class="panel-title">
        <span class="panel-title-text">
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

    <section class="panel panel-actions" style={bottomStyle(actionsCollapsed)}>
      <header class="panel-title">
        <span class="panel-title-text">
          {titles.actions}{#if actionsCollapsed && selectedActionLabel}<span class="panel-title-selected"> · {selectedActionLabel}</span>{/if}
        </span>
        <button class="collapse-btn" on:click={() => toggleCollapse('actions')} title={actionsCollapsed ? 'Expand' : 'Collapse'}>
          {actionsCollapsed ? '▸' : '▾'}
        </button>
      </header>
      {#if !actionsCollapsed}
        {#if actionGroups.length > 0}
          <div class="group-filter">
            <button
              class="collapse-btn group-filter-toggle"
              on:click={toggleGroupChips}
              title={groupChipsCollapsed ? 'Expand group filter' : 'Collapse group filter'}
            >
              {groupChipsCollapsed ? '▸' : '▾'}
            </button>
            {#if groupChipsCollapsed}
              <span class="group-summary">Groups: {groupSummary}</span>
            {:else}
              <div class="group-chips">
                <button class="chip" class:active={selectedGroups.size === 0} on:click={selectAllGroups}>All</button>
                {#each actionGroups as group (group)}
                  <button class="chip" class:active={selectedGroups.has(group)} on:click={() => toggleGroup(group)}>{group}</button>
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
        <span>Command</span>
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

  {#if toast}
    <div class="toast">{toast}</div>
  {/if}
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

  .resizer {
    flex: none;
    background: transparent;
  }

  .resizer:hover,
  .resizer:active {
    background: #3a4a63;
  }

  .resizer.disabled,
  .resizer.disabled:hover,
  .resizer.disabled:active {
    background: transparent;
    cursor: default;
  }

  .resizer.vertical {
    width: 6px;
    margin: 0 1px;
    cursor: col-resize;
  }

  .resizer.horizontal {
    height: 6px;
    margin: 1px 0;
    cursor: row-resize;
  }

  .panel {
    display: flex;
    flex-direction: column;
    min-height: 0;
    border: 1px solid #3a4a63;
    border-radius: 6px;
    background: #1f2c3d;
    overflow: hidden;
  }

  .panel-title {
    flex: none;
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding: 6px 6px 6px 10px;
    font-size: 0.8rem;
    font-weight: 700;
    letter-spacing: 0.03em;
    text-transform: uppercase;
    color: #7fd4ff;
    background: #253449;
    border-bottom: 1px solid #3a4a63;
  }

  .panel-title-text {
    flex: 1;
    min-width: 0;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .panel-title-selected {
    text-transform: none;
    font-weight: 400;
    color: #d7dee8;
  }

  .collapse-btn {
    flex: none;
    background: none;
    border: none;
    color: #7fd4ff;
    cursor: pointer;
    font-size: 0.8rem;
    line-height: 1;
    padding: 4px 6px;
    border-radius: 4px;
  }

  .collapse-btn:hover {
    background: #34455e;
  }

  .panel-body {
    flex: 1;
    overflow-y: auto;
    padding: 6px;
    scrollbar-width: thin;
    scrollbar-color: rgba(255, 255, 255, 0.14) transparent;
  }

  .panel-body::-webkit-scrollbar {
    width: 5px;
  }

  .panel-body::-webkit-scrollbar-track {
    background: transparent;
  }

  .panel-body::-webkit-scrollbar-thumb {
    background-color: rgba(255, 255, 255, 0.14);
    border-radius: 3px;
  }

  .panel-body::-webkit-scrollbar-thumb:hover {
    background-color: rgba(255, 255, 255, 0.28);
  }

  .list {
    display: flex;
    flex-direction: column;
    gap: 2px;
  }

  .group-filter {
    flex: none;
    display: flex;
    align-items: center;
    gap: 4px;
    padding: 4px 6px 0;
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

  .group-chips {
    flex: 1;
    display: flex;
    flex-wrap: wrap;
    gap: 4px;
    min-width: 0;
  }

  .chip {
    background: #14202f;
    color: #a9b6c8;
    border: 1px solid #3a4a63;
    border-radius: 999px;
    padding: 2px 9px;
    font-size: 0.72rem;
    cursor: pointer;
    font-family: inherit;
  }

  .chip:hover {
    background: #1f3346;
    color: #d7dee8;
  }

  .chip.active {
    background: #e8a33d;
    border-color: #e8a33d;
    color: #1b2636;
    font-weight: 700;
  }

  .row {
    display: block;
    width: 100%;
    text-align: left;
    background: none;
    border: none;
    border-radius: 4px;
    color: #d7dee8;
    padding: 6px 8px;
    font-size: 0.9rem;
    cursor: pointer;
    font-family: inherit;
  }

  .row:hover {
    background: #2b3b52;
  }

  .row.selected {
    background: #e8a33d;
    color: #1b2636;
    font-weight: 700;
  }

  .empty {
    color: #6b7a90;
    font-size: 0.85rem;
    padding: 8px;
    font-style: italic;
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

  .copy-cmd-btn {
    background: #2b3b52;
    color: #d7dee8;
    border: 1px solid #3a4a63;
    border-radius: 4px;
    padding: 5px 10px;
    cursor: pointer;
    font-size: 0.8rem;
  }

  .copy-cmd-btn:hover {
    background: #34455e;
  }

  .run-cmd-btn {
    background: #e8a33d;
    color: #1b2636;
    border: 1px solid #e8a33d;
    border-radius: 4px;
    padding: 5px 12px;
    cursor: pointer;
    font-size: 0.8rem;
    font-weight: 700;
  }

  .run-cmd-btn:hover {
    background: #f0b25a;
  }

  .toast {
    position: fixed;
    bottom: 16px;
    left: 50%;
    transform: translateX(-50%);
    background: #253449;
    color: #d7dee8;
    border: 1px solid #3a4a63;
    border-radius: 6px;
    padding: 8px 16px;
    font-size: 0.85rem;
    box-shadow: 0 4px 12px rgba(0, 0, 0, 0.4);
  }
</style>
