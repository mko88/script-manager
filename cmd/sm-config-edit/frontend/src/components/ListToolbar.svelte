<script lang="ts">
  import { createEventDispatcher } from 'svelte'
  import IconButton from '@shared/components/IconButton.svelte'
  import Icon from '@shared/components/Icon.svelte'

  export let addLabel: string
  export let copyLabel: string
  export let copyDisabled: boolean
  export let removeLabel: string
  export let removeDisabled: boolean
  export let reorderMode: boolean
  export let reorderEnterLabel: string
  export let reorderExitLabel: string

  const dispatch = createEventDispatcher<{ add: void; copy: void; remove: void; toggleReorder: void }>()
</script>

<div class="list-toolbar">
  <IconButton title={addLabel} on:click={() => dispatch('add')}><Icon name="add" /></IconButton>
  <IconButton title={copyLabel} disabled={copyDisabled} on:click={() => dispatch('copy')}><Icon name="copy" /></IconButton>
  <IconButton title={removeLabel} disabled={removeDisabled} on:click={() => dispatch('remove')}><Icon name="remove" /></IconButton>
  <IconButton
    active={reorderMode}
    title={reorderMode ? reorderExitLabel : reorderEnterLabel}
    on:click={() => dispatch('toggleReorder')}><Icon name="reorder" /></IconButton
  >
</div>
