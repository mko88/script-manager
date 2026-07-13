<script lang="ts">
  import ActionForm from './ActionForm.svelte'
  import ListToolbar from './ListToolbar.svelte'
  import { t } from '../messages'
  import { wrap, sortableList, syncList, type DndEntry } from '../lib/sortable'
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

  const syncActions = syncList<configedit.ActionDTO>({
    setEntries: (v) => (actionEntries = v),
    setDragging: (v) => (dragging = v),
    setList: (v) => (actions = v),
  })
</script>

<ListToolbar
  addLabel={t('tooltip.addAction')}
  removeLabel={t('tooltip.removeAction')}
  removeDisabled={selectedAction < 0}
  {reorderMode}
  reorderEnterLabel={t('tooltip.enterReorderMode')}
  reorderExitLabel={t('tooltip.exitReorderMode')}
  on:add={addAction}
  on:remove={() => confirmRemoveAction(selectedAction)}
  on:toggleReorder={toggleReorderMode}
/>
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
