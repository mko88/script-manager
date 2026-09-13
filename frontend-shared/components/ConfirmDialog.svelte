<script lang="ts">
  export let open = false
  export let title: string
  export let message = ''
  export let confirmLabel: string
  export let cancelLabel: string
  export let onConfirm: () => void
  export let onCancel: () => void

  let confirmEl: HTMLButtonElement | undefined

  $: if (open && confirmEl) confirmEl.focus()

  function onKeydown(e: KeyboardEvent) {
    if (!open) return
    if (e.key === 'Escape') {
      e.preventDefault()
      onCancel()
    } else if (e.key === 'Enter') {
      e.preventDefault()
      onConfirm()
    }
  }
</script>

<svelte:window on:keydown={onKeydown} />

{#if open}
  <div class="confirm-backdrop">
    <div class="confirm-dialog" role="dialog" aria-modal="true" aria-label={title}>
      <div class="confirm-title">{title}</div>
      {#if message}<p class="confirm-message">{message}</p>{/if}
      <div class="confirm-actions">
        <button class="confirm-btn" type="button" on:click={onCancel}>{cancelLabel}</button>
        <button class="confirm-btn confirm-primary" type="button" bind:this={confirmEl} on:click={onConfirm}
          >{confirmLabel}</button
        >
      </div>
    </div>
  </div>
{/if}

<style>
  /* Absolute against .app-root, not fixed against the viewport: the root is
     zoomed for the interface-scale setting, and a fixed inset would be
     measured in unzoomed pixels and then scaled past the window edge. */
  .confirm-backdrop {
    position: absolute;
    inset: 0;
    z-index: 200;
    display: flex;
    align-items: center;
    justify-content: center;
    background: var(--sm-scrim);
  }

  .confirm-dialog {
    width: 420px;
    max-width: calc(100% - 32px);
    padding: 16px;
    border: 1px solid var(--sm-border);
    border-radius: 8px;
    background: var(--sm-panel-header);
    box-shadow: 0 8px 24px var(--sm-shadow);
  }

  .confirm-title {
    margin-bottom: 6px;
    font-weight: 700;
    color: var(--sm-text-heading);
  }

  .confirm-message {
    /* The message is prose plus file paths, laid out with its own newlines. */
    margin: 0;
    white-space: pre-line;
    overflow-wrap: anywhere;
    font-size: var(--sm-type-sm);
    line-height: 1.4;
    color: var(--sm-text);
  }

  .confirm-actions {
    display: flex;
    justify-content: flex-end;
    gap: 8px;
    margin-top: 12px;
  }

  .confirm-btn {
    padding: 5px 12px;
    border: 1px solid var(--sm-border);
    border-radius: 4px;
    background: var(--sm-bg-deep);
    color: var(--sm-text);
    font-family: inherit;
    font-size: var(--sm-type-sm);
    cursor: pointer;
  }

  .confirm-btn:hover {
    background: var(--sm-tint-hover);
  }

  .confirm-primary {
    border-color: var(--sm-bg-primary);
    background: var(--sm-bg-primary);
    color: var(--sm-text-primary);
  }

  .confirm-primary:hover {
    background: var(--sm-primary-hover);
  }
</style>
