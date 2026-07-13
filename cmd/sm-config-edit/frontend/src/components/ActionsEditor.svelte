<script lang="ts">
  import Icon from '@shared/components/Icon.svelte'
  import ActionForm from './ActionForm.svelte'
  import { t } from '../messages'
  import { wrap, sortableList, type DndEntry } from '../lib/sortable'
  import type { DndEvent } from 'svelte-dnd-action'
  import type { configedit } from '../../wailsjs/go/models'

  // The Actions section: a reorderable master list of the global actions,
  // each edited through the shared ActionForm.

  // Two-way bound slices of the parent's cfg.
  export let actions: configedit.ActionDTO[]
  export let selectedAction: number
  export let allActionGroups: string[] = []
  export let browseScriptFile: () => Promise<string>

  function newAction(): configedit.ActionDTO {
    return {
      id: '',
      title: '',
      description: '',
      cmd: '',
      script: '',
      groups: [],
      noWait: false,
      interactive: true,
    } as unknown as configedit.ActionDTO
  }

  function addAction() {
    actions = [...actions, newAction()]
    selectedAction = actions.length - 1
  }
  function removeAction(i: number) {
    actions = actions.filter((_, idx) => idx !== i)
    if (selectedAction === i) selectedAction = -1
    else if (selectedAction > i) selectedAction -= 1
  }
  function confirmRemoveAction(i: number) {
    const name = actions[i]?.title || actions[i]?.id || t('fallback.untitled')
    if (confirm(t('confirm.removeAction', { name }))) removeAction(i)
  }

  // See ItemsEditor for why reordering is an explicit opt-in mode and why
  // entries aren't re-derived mid-drag.
  let reorderMode = false
  function toggleReorderMode() {
    reorderMode = !reorderMode
    if (reorderMode) selectedAction = -1
  }

  let dragging = false
  let actionEntries: DndEntry<configedit.ActionDTO>[] = wrap(actions)
  $: if (!dragging) actionEntries = wrap(actions)

  function syncActions(e: CustomEvent<DndEvent<DndEntry<configedit.ActionDTO>>>, final: boolean) {
    actionEntries = e.detail.items
    dragging = !final
    if (final) actions = actionEntries.filter((w) => w.ref).map((w) => w.ref)
  }
</script>

<div class="list-toolbar">
  <button class="btn icon-btn" type="button" title={t('tooltip.addAction')} aria-label={t('tooltip.addAction')} on:click={addAction}
    ><Icon name="add" /></button
  >
  <button
    class="btn icon-btn"
    type="button"
    title={t('tooltip.removeAction')}
    aria-label={t('tooltip.removeAction')}
    disabled={selectedAction < 0}
    on:click={() => confirmRemoveAction(selectedAction)}><Icon name="remove" /></button
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
    use:sortableList={{ items: actionEntries, onSync: syncActions, dragDisabled: !reorderMode }}
  >
    {#each actionEntries as entry, i (entry.id)}
      <button
        class="row"
        class:selected={selectedAction === i}
        on:click={() => {
          if (!reorderMode) selectedAction = i
        }}>{entry.ref.title || entry.ref.id || t('fallback.untitled')}</button
      >
    {/each}
  </div>
  <div class="detail">
    {#if selectedAction >= 0 && actions[selectedAction]}
      <ActionForm bind:action={actions[selectedAction]} {allActionGroups} {browseScriptFile} />
    {:else}
      <div class="empty">{t('empty.selectActionOrAdd')}</div>
    {/if}
  </div>
</div>
