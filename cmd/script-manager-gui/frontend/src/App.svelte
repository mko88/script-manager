<script lang="ts">
  import { onMount } from 'svelte'
  import {
    GetTitles,
    GetItems,
    GetActions,
    GetActionDetail,
    GetItemDetails,
    CopyToClipboard,
  } from '../wailsjs/go/main/App.js'
  import type { main } from '../wailsjs/go/models'

  let titles: main.TitlesDTO = { items: 'Items', actions: 'Actions', details: 'Details' }
  let items: main.ItemDTO[] = []
  let actions: main.ActionDTO[] = []
  let details: main.DetailsDTO | null = null
  let actionDetail: main.ActionDetailDTO | null = null

  let selectedItem = -1
  let selectedActionId: string | null = null
  let toast = ''
  let toastTimer: ReturnType<typeof setTimeout>

  onMount(async () => {
    titles = await GetTitles()
    items = await GetItems()
    if (items.length > 0) selectItem(0)
  })

  async function selectItem(index: number) {
    selectedItem = index
    selectedActionId = null
    actionDetail = null
    actions = await GetActions(index)
    details = await GetItemDetails(index)
  }

  async function selectAction(id: string) {
    if (selectedItem < 0) return
    selectedActionId = id
    actionDetail = await GetActionDetail(selectedItem, id)
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
</script>

<main class="app-shell">
  <div class="col col-left">
    <section class="panel panel-items">
      <header class="panel-title">{titles.items}</header>
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
    </section>

    <section class="panel panel-actions">
      <header class="panel-title">{titles.actions}</header>
      <div class="panel-body list">
        {#each actions as action (action.id || action.title)}
          <button
            class="row"
            class:selected={action.id === selectedActionId}
            on:click={() => selectAction(action.id)}
          >{action.title}</button>
        {/each}
        {#if selectedItem >= 0 && actions.length === 0}
          <div class="empty">No actions for this item</div>
        {/if}
      </div>
    </section>
  </div>

  <div class="col col-right">
    <section class="panel panel-details">
      <header class="panel-title">{titles.details}</header>
      <!-- svelte-ignore a11y-click-events-have-key-events -->
      <!-- svelte-ignore a11y-no-static-element-interactions -->
      <div class="panel-body details-content" on:click={onDetailsClick}>
        {#if details?.html}
          {@html details.html}
        {:else}
          <div class="empty">No item selected</div>
        {/if}
      </div>
    </section>

    <section class="panel panel-command">
      <header class="panel-title">Command</header>
      <div class="panel-body command-content">
        {#if actionDetail}
          {#if actionDetail.description}
            <p class="cmd-desc">{actionDetail.description}</p>
          {/if}
          {#if actionDetail.cmd}
            <pre class="cmd-line">$ {actionDetail.cmd}</pre>
            <button class="copy-cmd-btn" on:click={copyCmd}>Copy command</button>
          {/if}
        {:else}
          <div class="empty">Select an action to preview its command</div>
        {/if}
      </div>
    </section>
  </div>

  {#if toast}
    <div class="toast">{toast}</div>
  {/if}
</main>

<style>
  .app-shell {
    display: grid;
    grid-template-columns: 320px 1fr;
    gap: 8px;
    height: 100vh;
    box-sizing: border-box;
    padding: 8px;
    text-align: left;
  }

  .col {
    display: flex;
    flex-direction: column;
    gap: 8px;
    min-height: 0;
  }

  .col-left {
    flex: 1;
  }

  .col-right {
    flex: 2;
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

  .panel-items,
  .panel-details {
    flex: 2;
  }

  .panel-actions,
  .panel-command {
    flex: 1;
  }

  .panel-title {
    flex: none;
    padding: 6px 10px;
    font-size: 0.8rem;
    font-weight: 700;
    letter-spacing: 0.03em;
    text-transform: uppercase;
    color: #7fd4ff;
    background: #253449;
    border-bottom: 1px solid #3a4a63;
  }

  .panel-body {
    flex: 1;
    overflow-y: auto;
    padding: 6px;
  }

  .list {
    display: flex;
    flex-direction: column;
    gap: 2px;
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
    white-space: pre-wrap;
    word-break: break-word;
    background: #14202f;
    border-radius: 4px;
    padding: 8px;
    margin: 0 0 8px;
    color: #d7dee8;
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
