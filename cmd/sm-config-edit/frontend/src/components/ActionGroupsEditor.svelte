<script lang="ts">
  import Icon from '@shared/components/Icon.svelte'
  import { t } from '../messages'
  import { wrap, sortableList, type DndEntry } from '../lib/sortable'
  import type { DndEvent } from 'svelte-dnd-action'
  import type { configedit } from '../../wailsjs/go/models'

  // The Action Groups section: edits the id/title/color catalog entries.
  // Deleting a group also scrubs its id out of every action's and item's
  // group lists, so actions and items are bound too — the picker UIs
  // already hide unknown ids, but the underlying data would otherwise
  // silently keep the stale id forever.

  // Two-way bound slices of the parent's cfg.
  export let actionGroups: configedit.ActionGroupDTO[]
  export let actions: configedit.ActionDTO[]
  export let items: configedit.ItemDTO[]
  export let selectedActionGroup: number

  function newActionGroup(): configedit.ActionGroupDTO {
    return { id: '', title: '', color: '' } as unknown as configedit.ActionGroupDTO
  }

  function addActionGroup() {
    actionGroups = [...actionGroups, newActionGroup()]
    selectedActionGroup = actionGroups.length - 1
  }

  // How many actions/items/custom-actions currently reference a group id —
  // used to warn before deleting.
  function actionGroupRefCount(id: string): number {
    let count = 0
    for (const a of actions) if (a.groups.includes(id)) count++
    for (const it of items) {
      if (it.actionGroups.includes(id)) count++
      for (const ca of it.customActions) if (ca.groups.includes(id)) count++
    }
    return count
  }
  function removeActionGroupReferences(id: string) {
    actions = actions.map((a) => ({ ...a, groups: a.groups.filter((g) => g !== id) })) as unknown as configedit.ActionDTO[]
    items = items.map((it) => ({
      ...it,
      actionGroups: it.actionGroups.filter((g) => g !== id),
      customActions: it.customActions.map((ca) => ({ ...ca, groups: ca.groups.filter((g) => g !== id) })),
    })) as unknown as configedit.ItemDTO[]
  }
  function removeActionGroup(i: number) {
    const id = actionGroups[i]?.id
    actionGroups = actionGroups.filter((_, idx) => idx !== i)
    if (id) removeActionGroupReferences(id)
    if (selectedActionGroup === i) selectedActionGroup = -1
    else if (selectedActionGroup > i) selectedActionGroup -= 1
  }
  function confirmRemoveActionGroup(i: number) {
    const g = actionGroups[i]
    const name = g?.title || g?.id || t('fallback.unnamed')
    const refCount = g?.id ? actionGroupRefCount(g.id) : 0
    const refSuffix = refCount > 0 ? t('confirm.removeActionGroupRefSuffix', { count: refCount, plural: refCount > 1 ? 's' : '' }) : ''
    if (confirm(t('confirm.removeActionGroup', { name, refSuffix }))) removeActionGroup(i)
  }

  // See ItemsEditor for why reordering is an explicit opt-in mode and why
  // entries aren't re-derived mid-drag.
  let reorderMode = false
  function toggleReorderMode() {
    reorderMode = !reorderMode
    if (reorderMode) selectedActionGroup = -1
  }

  let dragging = false
  let actionGroupEntries: DndEntry<configedit.ActionGroupDTO>[] = wrap(actionGroups)
  $: if (!dragging) actionGroupEntries = wrap(actionGroups)

  function syncActionGroups(e: CustomEvent<DndEvent<DndEntry<configedit.ActionGroupDTO>>>, final: boolean) {
    actionGroupEntries = e.detail.items
    dragging = !final
    if (final) actionGroups = actionGroupEntries.filter((w) => w.ref).map((w) => w.ref)
  }
</script>

<div class="list-toolbar">
  <button
    class="btn icon-btn"
    type="button"
    title={t('tooltip.addActionGroup')}
    aria-label={t('tooltip.addActionGroup')}
    on:click={addActionGroup}><Icon name="add" /></button
  >
  <button
    class="btn icon-btn"
    type="button"
    title={t('tooltip.removeActionGroup')}
    aria-label={t('tooltip.removeActionGroup')}
    disabled={selectedActionGroup < 0}
    on:click={() => confirmRemoveActionGroup(selectedActionGroup)}><Icon name="remove" /></button
  >
  <button
    class="btn icon-btn"
    class:active={reorderMode}
    type="button"
    title={reorderMode ? t('tooltip.exitReorderMode') : t('tooltip.enterReorderMode')}
    aria-label={reorderMode ? t('tooltip.exitReorderMode') : t('tooltip.enterReorderMode')}
    on:click={toggleReorderMode}><Icon name="reorder" /></button
  >
</div>
<div class="master-detail">
  <div
    class="master list"
    class:reorder-mode={reorderMode}
    use:sortableList={{ items: actionGroupEntries, onSync: syncActionGroups, dragDisabled: !reorderMode }}
  >
    {#each actionGroupEntries as entry, i (entry.id)}
      <button
        class="row"
        class:selected={selectedActionGroup === i}
        on:click={() => {
          if (!reorderMode) selectedActionGroup = i
        }}
      >
        <span class="group-swatch" style="background: {entry.ref.color || 'var(--sm-border)'}"></span>
        {entry.ref.title || entry.ref.id || t('fallback.unnamed')}
      </button>
    {/each}
  </div>
  <div class="detail">
    {#if selectedActionGroup >= 0 && actionGroups[selectedActionGroup]}
      <label class="field">
        <span>{t('field.id')}</span>
        <input
          type="text"
          bind:value={actionGroups[selectedActionGroup].id}
          placeholder={t('placeholder.actionGroupId')}
        />
      </label>
      <label class="field">
        <span>{t('field.title')}</span>
        <input
          type="text"
          bind:value={actionGroups[selectedActionGroup].title}
          placeholder={t('placeholder.actionGroupTitle')}
        />
      </label>
      <div class="field">
        <span>{t('field.color')}</span>
        <div class="color-field">
          <input
            type="color"
            value={/^#[0-9a-fA-F]{6}$/.test(actionGroups[selectedActionGroup].color)
              ? actionGroups[selectedActionGroup].color
              : '#7fd4ff'}
            on:input={(e) => (actionGroups[selectedActionGroup].color = e.currentTarget.value)}
            title={t('tooltip.pickColor')}
          />
          <input
            type="text"
            bind:value={actionGroups[selectedActionGroup].color}
            placeholder={t('placeholder.actionGroupColor')}
          />
        </div>
      </div>
    {:else}
      <div class="empty">{t('empty.selectActionGroupOrAdd')}</div>
    {/if}
  </div>
</div>

<style>
  .group-swatch {
    display: inline-block;
    width: 10px;
    height: 10px;
    border-radius: 50%;
    margin-right: 6px;
    vertical-align: middle;
    border: 1px solid var(--sm-border);
  }
</style>
