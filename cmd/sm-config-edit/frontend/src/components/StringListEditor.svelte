<script lang="ts">
  import { t } from '../messages'
  import { confirmAsk } from '../lib/confirm'

  export let items: string[] = []
  export let placeholder = ''
  export let confirmRemoveMessage: ((value: string) => string) | null = null

  function add() {
    items = [...items, '']
  }
  async function remove(i: number) {
    if (confirmRemoveMessage && !(await confirmAsk(confirmRemoveMessage(items[i])))) return
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
    font-size: var(--sm-type-base);
  }
</style>
