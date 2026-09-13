<script lang="ts">
  import { t } from '../messages'
  import { confirmAsk } from '../lib/confirm'

  export let configured = false
  export let unlocked = false
  export let lockedCount = 0
  export let onSetPin: (pin: string) => Promise<string>
  export let onChangePin: (oldPin: string, newPin: string) => Promise<string>
  export let onRemovePin: (pin: string) => Promise<string>

  let pin = ''
  let confirmPin = ''
  let currentPin = ''
  let removePin = ''
  let error = ''
  let busy = false

  $: mismatch = confirmPin !== '' && pin !== confirmPin
  $: canSubmit =
    !busy && pin.length > 0 && confirmPin.length > 0 && !mismatch && (!configured || currentPin.length > 0)

  function reset() {
    pin = ''
    confirmPin = ''
    currentPin = ''
    removePin = ''
  }

  async function submit() {
    if (!canSubmit) return
    busy = true
    error = configured ? await onChangePin(currentPin, pin) : await onSetPin(pin)
    busy = false
    if (!error) reset()
  }

  async function remove() {
    if (busy || removePin.length === 0) return
    if (!(await confirmAsk(t('secrets.removeConfirm', { count: lockedCount })))) return
    busy = true
    error = await onRemovePin(removePin)
    busy = false
    if (!error) reset()
  }
</script>

<div class="secrets-editor">
  <div class="secrets-status" class:secrets-status-set={configured}>
    {#if configured}
      {t('secrets.statusSet', { count: lockedCount })}
      {unlocked ? t('secrets.statusUnlocked') : t('secrets.statusLocked')}
    {:else}
      {t('secrets.statusNone')}
    {/if}
  </div>

  <p class="hint">{configured ? t('secrets.changeHint') : t('secrets.setHint')}</p>

  <div class="secrets-form">
    {#if configured}
      <label class="field">
        <span>{t('secrets.currentPin')}</span>
        <input type="password" autocomplete="off" bind:value={currentPin} />
      </label>
    {/if}
    <label class="field">
      <span>{configured ? t('secrets.newPin') : t('secrets.pin')}</span>
      <input type="password" autocomplete="off" bind:value={pin} />
    </label>
    <label class="field">
      <span>{t('secrets.repeatPin')}</span>
      <input type="password" autocomplete="off" bind:value={confirmPin} />
    </label>

    {#if mismatch}<div class="validation-issue validation-error">{t('secrets.mismatch')}</div>{/if}
    {#if error}<div class="validation-issue validation-error">{error}</div>{/if}

    <div class="secrets-actions">
      <button class="btn" type="button" disabled={!canSubmit} on:click={submit}>
        {configured ? t('secrets.changeButton') : t('secrets.setButton')}
      </button>
    </div>
  </div>

  {#if configured}
    <div class="secrets-remove">
      <p class="hint">{t('secrets.removeHint')}</p>
      <label class="field">
        <span>{t('secrets.removeCurrentPin')}</span>
        <input type="password" autocomplete="off" bind:value={removePin} />
      </label>
      <div class="secrets-actions">
        <button class="btn" type="button" disabled={busy || removePin.length === 0} on:click={remove}>
          {t('secrets.removeButton')}
        </button>
      </div>
    </div>
  {/if}

  <p class="hint secrets-warning">{t('secrets.warning')}</p>
</div>

<style>
  .secrets-editor {
    max-width: 520px;
  }

  .secrets-status {
    margin-bottom: 10px;
    padding: 8px 10px;
    border: 1px solid var(--sm-border);
    border-radius: 6px;
    background: var(--sm-bg-deep);
    font-size: var(--sm-type-base);
  }

  .secrets-status-set {
    border-color: var(--sm-run-ok);
  }

  .secrets-form {
    display: flex;
    flex-direction: column;
    gap: 8px;
    margin-top: 12px;
  }

  .secrets-actions {
    margin-top: 4px;
  }

  .secrets-remove {
    margin-top: 20px;
    padding-top: 12px;
    border-top: 1px solid var(--sm-border);
  }

  .secrets-remove .field {
    margin-top: 8px;
  }

  .secrets-warning {
    margin-top: 16px;
    padding-top: 12px;
    border-top: 1px solid var(--sm-border);
  }
</style>
