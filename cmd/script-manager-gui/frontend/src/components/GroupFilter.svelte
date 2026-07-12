<script lang="ts">
  import { t } from '../messages'
  import { groupChipStyle } from '../lib/groupColors'
  import type { gui } from '../../wailsjs/go/models'

  // The current item's actions — the filter derives its group list and
  // counts from these.
  export let actions: gui.ActionDTO[] = []
  // Catalog colors from lib/groupColors.buildGroupColors, shared with the
  // Command pane's own group chips so both render identically.
  export let groupColors: Record<string, string> = {}
  // Two-way bound: empty set means "All" — no filter, show everything.
  // Otherwise an action matches if it belongs to every selected group (AND
  // semantics, applied by the parent's filteredActions).
  export let selectedGroups = new Set<string>()
  // Two-way bound so the parent can persist it with the rest of its layout.
  export let collapsed = true
  // Called after the collapse toggle / a selection change respectively —
  // the parent persists layout on the former and resets its action
  // selection on the latter.
  export let onCollapseChange: () => void = () => {}
  export let onSelectionChange: () => void = () => {}

  // Group chips are sorted by exactly one key at a time: the active button
  // (name or count) owns the order. Clicking the active button flips its
  // direction; clicking the inactive one switches to that key, keeping the
  // direction it last had.
  let sortMode: 'alpha' | 'count' = 'alpha'
  let alphaDir: 'asc' | 'desc' = 'asc'
  let countDir: 'asc' | 'desc' = 'desc'

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

  function toggleCollapsed() {
    collapsed = !collapsed
    onCollapseChange()
  }

  function selectAllGroups() {
    selectedGroups = new Set()
    onSelectionChange()
  }

  function toggleGroup(group: string) {
    const next = new Set(selectedGroups)
    if (next.has(group)) next.delete(group)
    else next.add(group)
    selectedGroups = next
    onSelectionChange()
  }

  function toggleSort(axis: 'alpha' | 'count') {
    if (sortMode !== axis) {
      sortMode = axis
      return
    }
    if (axis === 'alpha') alphaDir = alphaDir === 'asc' ? 'desc' : 'asc'
    else countDir = countDir === 'asc' ? 'desc' : 'asc'
  }
</script>

{#if actionGroups.length > 0}
  <div class="group-filter">
    <div class="group-filter-header">
      <button
        class="collapse-btn group-filter-toggle"
        on:click={toggleCollapsed}
        title={collapsed ? t('tooltip.expandGroupFilter') : t('tooltip.collapseGroupFilter')}
      >
        {collapsed ? '▸' : '▾'}
      </button>
      {#if collapsed}
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
    {#if !collapsed}
      <div class="group-chips">
        <button class="chip" class:active={selectedGroups.size === 0} on:click={selectAllGroups}>{t('text.allGroupsChip')}</button>
        {#each visibleGroups as group (group)}
          <button
            class="chip"
            class:active={selectedGroups.has(group)}
            style={groupChipStyle(groupColors, group, selectedGroups.has(group))}
            on:click={() => toggleGroup(group)}
            >{group}<span class="chip-count">{t('text.groupCount', { count: groupCounts[group] ?? 0 })}</span></button
          >
        {/each}
      </div>
    {/if}
  </div>
{/if}

<style>
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
</style>
