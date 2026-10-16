<script lang="ts">
  import { t } from '../messages'

  export let open = false
  export let initial = ''
  export let existing: string[] = []
  export let onSubmit: (name: string) => void
  export let onCancel: () => void

  let name = ''
  let inputEl: HTMLInputElement | undefined

  $: if (open) name = initial
  $: if (open && inputEl) inputEl.focus()
  $: trimmed = name.trim()
  $: replaces = existing.includes(trimmed)

  function submit() {
    if (trimmed) onSubmit(trimmed)
  }

  function onKeydown(e: KeyboardEvent) {
    if (!open) return
    if (e.key === 'Escape') {
      e.preventDefault()
      onCancel()
    } else if (e.key === 'Enter') {
      e.preventDefault()
      submit()
    }
  }
</script>

<svelte:window on:keydown={onKeydown} />

{#if open}
  <div class="name-backdrop">
    <div class="name-dialog" role="dialog" aria-modal="true" aria-label={t('params.presetNameTitle')}>
      <div class="name-title">{t('params.presetNameTitle')}</div>
      <label class="field">
        <span>{t('params.presetNameLabel')}</span>
        <input type="text" bind:value={name} bind:this={inputEl} autocomplete="off" />
      </label>
      {#if replaces}<p class="hint">{t('params.presetReplaceNote')}</p>{/if}
      <div class="name-actions">
        <button class="btn" type="button" on:click={onCancel}>{t('params.presetNameCancel')}</button>
        <button class="btn btn-primary" type="button" disabled={!trimmed} on:click={submit}>{t('params.presetNameSave')}</button>
      </div>
    </div>
  </div>
{/if}

<style>
  /* Absolute against .app-root, like the PIN dialog: the root is zoomed for
     the interface-scale setting. */
  .name-backdrop {
    position: absolute;
    inset: 0;
    z-index: 200;
    display: flex;
    align-items: center;
    justify-content: center;
    background: var(--sm-scrim);
  }

  .name-dialog {
    width: 320px;
    max-width: calc(100% - 32px);
    padding: 16px;
    border: 1px solid var(--sm-border);
    border-radius: 8px;
    background: var(--sm-panel-header);
    box-shadow: 0 8px 24px var(--sm-shadow);
  }

  .name-title {
    margin-bottom: 8px;
    font-weight: 700;
    color: var(--sm-text-heading);
  }

  .name-actions {
    display: flex;
    justify-content: flex-end;
    gap: 8px;
    margin-top: 12px;
  }

  .name-actions .btn:disabled {
    opacity: 0.5;
    cursor: default;
  }
</style>
