<script lang="ts">
  import { createEventDispatcher } from 'svelte'
  import IconButton from '@shared/components/IconButton.svelte'
  import Icon from '@shared/components/Icon.svelte'

  // The Add/Remove/Reorder-toggle row above a reorderable master list —
  // identical in shape across the Actions/Action Groups/Items sections, so
  // a future change (a new button, a tweak to the reorder-toggle behavior)
  // touches this one file instead of all three editors.
  export let addLabel: string
  export let removeLabel: string
  export let removeDisabled: boolean
  export let reorderMode: boolean
  export let reorderEnterLabel: string
  export let reorderExitLabel: string

  const dispatch = createEventDispatcher<{ add: void; remove: void; toggleReorder: void }>()
</script>

<div class="list-toolbar">
  <IconButton title={addLabel} on:click={() => dispatch('add')}><Icon name="add" /></IconButton>
  <IconButton title={removeLabel} disabled={removeDisabled} on:click={() => dispatch('remove')}><Icon name="remove" /></IconButton>
  <IconButton
    active={reorderMode}
    title={reorderMode ? reorderExitLabel : reorderEnterLabel}
    on:click={() => dispatch('toggleReorder')}><Icon name="reorder" /></IconButton
  >
</div>
