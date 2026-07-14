<script lang="ts">
  import { t } from '../messages'

  // A reusable ordered string-list editor: Shell, Terminal's custom-argv mode,
  // an Action's Groups, and an item's Actions/ActionGroups all edit a plain
  // string[] this way.
  export let items: string[] = []
  export let placeholder = ''
  // Opt-in per usage: most lists here (Terminal argv, an Action's Groups)
  // are reordered/pruned freely and a confirm would just be friction, but
  // Shell's entries are load-bearing enough to warrant one. Unset means no
  // confirmation, matching every other list here's existing behavior.
  export let confirmRemoveMessage: ((value: string) => string) | null = null

  function add() {
    items = [...items, '']
  }
  function remove(i: number) {
    if (confirmRemoveMessage && !confirm(confirmRemoveMessage(items[i]))) return
    items = items.filter((_, idx) => idx !== i)
  }
</script>

<div class="string-list">
  {#each items as _, i (i)}
    <div class="string-list-row">
      <input type="text" bind:value={items[i]} {placeholder} />
      <button class="btn" type="button" title={t('tooltip.remove')} on:click={() => remove(i)}>{t('text.removeGlyph')}</button>
    </div>
  {/each}
  <button class="btn" type="button" on:click={add}>{t('button.add')}</button>
</div>

<style>
  .string-list {
    display: flex;
    flex-direction: column;
    gap: 4px;
  }
  .string-list-row {
    display: flex;
    gap: 4px;
  }
  .string-list-row input {
    flex: 1;
    min-width: 0;
    background: var(--sm-bg-deep);
    color: var(--sm-text);
    border: 1px solid var(--sm-border);
    border-radius: 4px;
    padding: 4px 6px;
    font-family: inherit;
    font-size: 0.85rem;
  }
</style>
