<script lang="ts">
  export let open = false
  export let title: string
  export let message = ''
  export let confirmPrompt = ''
  export let pinLabel: string
  export let confirmLabel: string
  export let cancelLabel: string
  export let requireConfirm = false
  export let mismatchLabel = ''
  export let error = ''
  export let onSubmit: (pin: string) => void
  export let onCancel: () => void

  let pin = ''
  let confirmPin = ''
  let inputEl: HTMLInputElement | undefined

  $: if (open && inputEl) inputEl.focus()
  $: if (!open) {
    pin = ''
    confirmPin = ''
  }

  $: mismatch = requireConfirm && confirmPin !== '' && pin !== confirmPin
  $: canSubmit = pin.length > 0 && (!requireConfirm || (confirmPin.length > 0 && !mismatch))

  function submit() {
    if (!canSubmit) return
    onSubmit(pin)
  }

  function cancel() {
    onCancel()
  }

  function onKeydown(e: KeyboardEvent) {
    if (!open) return
    if (e.key === 'Escape') {
      e.preventDefault()
      cancel()
    } else if (e.key === 'Enter') {
      e.preventDefault()
      submit()
    }
  }
</script>

<svelte:window on:keydown={onKeydown} />

{#if open}
  <div class="pin-backdrop">
    <div class="pin-dialog" role="dialog" aria-modal="true" aria-label={title}>
      <div class="pin-title">{title}</div>
      {#if message}<p class="pin-message">{message}</p>{/if}
      <label class="pin-field">
        <span>{pinLabel}</span>
        <input type="password" bind:value={pin} bind:this={inputEl} autocomplete="off" />
      </label>
      {#if requireConfirm}
        <label class="pin-field">
          <span>{confirmPrompt}</span>
          <input type="password" bind:value={confirmPin} autocomplete="off" />
        </label>
      {/if}
      {#if mismatch}<div class="pin-error">{mismatchLabel}</div>{/if}
      {#if error}<div class="pin-error">{error}</div>{/if}
      <div class="pin-actions">
        <button class="pin-btn" type="button" on:click={cancel}>{cancelLabel}</button>
        <button class="pin-btn pin-primary" type="button" disabled={!canSubmit} on:click={submit}>{confirmLabel}</button>
      </div>
    </div>
  </div>
{/if}

<style>
  .pin-backdrop {
    position: fixed;
    inset: 0;
    z-index: 200;
    display: flex;
    align-items: center;
    justify-content: center;
    background: var(--sm-scrim);
  }

  .pin-dialog {
    width: 320px;
    padding: 16px;
    border: 1px solid var(--sm-border);
    border-radius: 8px;
    background: var(--sm-panel-header);
    box-shadow: 0 8px 24px var(--sm-shadow);
  }

  .pin-title {
    margin-bottom: 6px;
    font-weight: 700;
    color: var(--sm-text-heading);
  }

  .pin-message {
    margin: 0 0 10px;
    font-size: var(--sm-type-sm);
    line-height: 1.4;
    color: var(--sm-text);
  }

  .pin-field {
    display: flex;
    flex-direction: column;
    gap: 4px;
    margin-bottom: 8px;
    font-size: var(--sm-type-sm);
    color: var(--sm-text-muted);
  }

  .pin-field input {
    padding: 6px 8px;
    border: 1px solid var(--sm-border);
    border-radius: 4px;
    background: var(--sm-bg-deep);
    color: var(--sm-text);
    font-family: inherit;
    font-size: var(--sm-type-lg);
  }

  .pin-error {
    margin-bottom: 8px;
    font-size: var(--sm-type-sm);
    color: var(--sm-error);
  }

  .pin-actions {
    display: flex;
    justify-content: flex-end;
    gap: 8px;
    margin-top: 12px;
  }

  .pin-btn {
    padding: 5px 12px;
    border: 1px solid var(--sm-border);
    border-radius: 4px;
    background: var(--sm-bg-deep);
    color: var(--sm-text);
    font-family: inherit;
    font-size: var(--sm-type-sm);
    cursor: pointer;
  }

  .pin-btn:hover:not(:disabled) {
    background: var(--sm-tint-hover);
  }

  .pin-btn:disabled {
    opacity: 0.5;
    cursor: default;
  }

  .pin-primary {
    border-color: var(--sm-bg-primary);
    background: var(--sm-bg-primary);
    color: var(--sm-text-primary);
  }

  .pin-primary:hover:not(:disabled) {
    background: var(--sm-primary-hover);
  }
</style>
