<script lang="ts">
  import IconButton from './IconButton.svelte'

  export let title: string
  export let aria: string | undefined = undefined
  export let recents: string[] = []
  export let currentPath = ''
  export let browseLabel: string
  export let clearLabel: string
  export let emptyLabel: string
  export let recentsHeading: string
  export let onOpen: (path: string) => void
  export let onBrowse: () => void
  export let onClear: () => void

  let open = false
  let rootEl: HTMLElement

  function toggle() {
    open = !open
  }

  function close() {
    open = false
  }

  function choose(path: string) {
    close()
    onOpen(path)
  }

  function browse() {
    close()
    onBrowse()
  }

  function clear() {
    close()
    onClear()
  }

  function onWindowClick(e: MouseEvent) {
    if (open && rootEl && !rootEl.contains(e.target as Node)) close()
  }

  function onWindowKeydown(e: KeyboardEvent) {
    if (open && e.key === 'Escape') {
      e.preventDefault()
      close()
    }
  }
</script>

<svelte:window on:click={onWindowClick} on:keydown={onWindowKeydown} />

<div class="recent-menu" bind:this={rootEl}>
  <IconButton {title} {aria} active={open} on:click={toggle}><slot /></IconButton>
  {#if open}
    <div class="recent-popover">
      <div class="recent-heading">{recentsHeading}</div>
      {#if recents.length === 0}
        <div class="recent-empty">{emptyLabel}</div>
      {:else}
        <ul class="recent-list">
          {#each recents as path (path)}
            <li>
              <button
                class="recent-item"
                class:current={path === currentPath}
                type="button"
                title={path}
                on:click={() => choose(path)}>{path}</button
              >
            </li>
          {/each}
        </ul>
      {/if}
      <div class="recent-actions">
        <button class="recent-action" type="button" disabled={recents.length === 0} on:click={clear}
          >{clearLabel}</button
        >
        <button class="recent-action" type="button" on:click={browse}>{browseLabel}</button>
      </div>
    </div>
  {/if}
</div>

<style>
  .recent-menu {
    position: relative;
    display: inline-flex;
  }

  .recent-popover {
    position: absolute;
    top: calc(100% + 4px);
    left: 0;
    z-index: 20;
    min-width: 280px;
    max-width: 620px;
    padding: 6px;
    border: 1px solid var(--sm-border);
    border-radius: 6px;
    background: var(--sm-panel-header);
    box-shadow: 0 4px 12px var(--sm-shadow);
  }

  .recent-heading {
    padding: 4px 8px;
    font-size: 0.7rem;
    text-transform: uppercase;
    letter-spacing: 0.04em;
    color: var(--sm-text-muted);
  }

  .recent-empty {
    padding: 6px 8px 8px;
    font-size: 0.8rem;
    color: var(--sm-text-muted);
  }

  .recent-list {
    max-height: 320px;
    overflow-y: auto;
    margin: 0;
    padding: 0;
    list-style: none;
  }

  .recent-item {
    display: block;
    width: 100%;
    padding: 5px 8px;
    border: 0;
    border-radius: 4px;
    background: none;
    color: var(--sm-text);
    font-family: inherit;
    font-size: 0.8rem;
    text-align: left;
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
    cursor: pointer;
  }

  .recent-item:hover {
    background: var(--sm-tint-hover);
  }

  .recent-item.current {
    color: var(--sm-text-highlight);
  }

  .recent-actions {
    display: flex;
    justify-content: space-between;
    gap: 6px;
    margin-top: 6px;
    padding-top: 6px;
    border-top: 1px solid var(--sm-border);
  }

  .recent-action {
    padding: 4px 10px;
    border: 1px solid var(--sm-border);
    border-radius: 4px;
    background: var(--sm-bg-deep);
    color: var(--sm-text);
    font-family: inherit;
    font-size: 0.8rem;
    cursor: pointer;
  }

  .recent-action:hover:not(:disabled) {
    background: var(--sm-tint-hover);
  }

  .recent-action:disabled {
    opacity: 0.5;
    cursor: default;
  }
</style>
