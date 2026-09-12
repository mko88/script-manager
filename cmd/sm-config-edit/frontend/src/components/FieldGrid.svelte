<script lang="ts">
  import { t } from '../messages'
  import { looksLikeSecretKey } from '../secretKey'
  import IconButton from '@shared/components/IconButton.svelte'
  import CodeMirror from '@shared/components/CodeMirror.svelte'

  export let fields: { key: string; kind: string; value: string; secret: boolean; locked?: boolean }[] = []
  export let validateField: (kind: string, value: string) => Promise<string> = async () => ''
  export let onToggleLock: ((field: { key: string; value: string; locked?: boolean }) => Promise<string | null>) | null =
    null

  const kinds = ['string', 'multiline', 'number', 'bool', 'yaml'] as const

  const secretPlaceholder = t('text.secretMask')
  const LOCK_PREFIX = 'sm-enc:v1:'

  let errors: Record<number, string> = {}
  let focused: Record<number, boolean> = {}

  function onValueInput(i: number, e: Event) {
    fields[i].value = (e.currentTarget as HTMLInputElement | HTMLTextAreaElement).value
    check(i)
  }

  function add() {
    fields = [...fields, { key: '', kind: 'string', value: '', secret: false, locked: false }]
  }

  async function toggleLock(i: number) {
    if (!onToggleLock) return
    const next = await onToggleLock(fields[i])
    if (next === null) return
    fields[i].value = next
    fields[i].locked = next.startsWith(LOCK_PREFIX)
    if (fields[i].locked) fields[i].secret = true
    fields = fields
  }
  function remove(i: number) {
    const key = fields[i]?.key || t('fallback.unnamed')
    if (!confirm(t('confirm.removeField', { key }))) return
    fields = fields.filter((_, idx) => idx !== i)
    errors = {}
  }
  async function check(i: number) {
    errors = { ...errors, [i]: await validateField(fields[i].kind, fields[i].value) }
  }
  async function onKindChange(i: number) {
    if (fields[i].kind === 'bool' && fields[i].value !== 'true' && fields[i].value !== 'false') {
      fields[i].value = 'false'
    }
    await check(i)
  }
  function toggleSecret(i: number) {
    fields[i].secret = !fields[i].secret
  }

  function onKeyInput(i: number) {
    if (!fields[i].secret && looksLikeSecretKey(fields[i].key)) {
      fields[i].secret = true
    }
  }
</script>

<div class="field-grid">
  {#each fields as _, i (i)}
    <div class="field-row">
      <input
        class="field-key"
        type="text"
        placeholder={t('placeholder.fieldKey')}
        bind:value={fields[i].key}
        on:input={() => onKeyInput(i)}
      />
      <select class="field-kind sm-select" bind:value={fields[i].kind} on:change={() => onKindChange(i)}>
        {#each kinds as k}<option value={k}>{k}</option>{/each}
      </select>
      {#if fields[i].kind === 'bool'}
        <select class="field-value sm-select" bind:value={fields[i].value} on:change={() => check(i)}>
          <option value="true">true</option>
          <option value="false">false</option>
        </select>
      {:else if fields[i].kind === 'yaml' && !fields[i].secret}
        <div class="field-value field-value-yaml">
          <CodeMirror
            bind:value={fields[i].value}
            language="yaml"
            showLineNumbers={false}
            minHeight="3.2em"
            maxHeight="220px"
            on:blur={() => check(i)}
          />
        </div>
      {:else if fields[i].kind === 'yaml'}
        <textarea
          class="field-value field-value-yaml"
          class:field-value-secret={fields[i].secret && focused[i]}
          rows="2"
          value={fields[i].secret && !focused[i] ? secretPlaceholder : fields[i].value}
          on:input={(e) => onValueInput(i, e)}
          on:focus={() => (focused = { ...focused, [i]: true })}
          on:blur={() => (focused = { ...focused, [i]: false })}
        ></textarea>
      {:else if fields[i].kind === 'multiline'}
        <textarea
          class="field-value"
          class:field-value-secret={fields[i].secret && focused[i]}
          rows="3"
          value={fields[i].secret && !focused[i] ? secretPlaceholder : fields[i].value}
          on:input={(e) => onValueInput(i, e)}
          on:focus={() => (focused = { ...focused, [i]: true })}
          on:blur={() => (focused = { ...focused, [i]: false })}
        ></textarea>
      {:else}
        <input
          class="field-value"
          class:field-value-secret={fields[i].secret && focused[i]}
          type="text"
          readonly={fields[i].locked}
          value={fields[i].locked || (fields[i].secret && !focused[i]) ? secretPlaceholder : fields[i].value}
          on:input={(e) => onValueInput(i, e)}
          on:focus={() => (focused = { ...focused, [i]: true })}
          on:blur={() => (focused = { ...focused, [i]: false })}
        />
      {/if}
      <IconButton
        class="btn icon-btn field-icon-btn"
        active={fields[i].secret}
        title={fields[i].secret ? t('tooltip.markedSecret') : t('tooltip.markSecret')}
        on:click={() => toggleSecret(i)}
      >
        {#if fields[i].secret}
          <svg viewBox="0 0 16 16" width="13" height="13" aria-hidden="true">
            <rect x="3.5" y="7" width="9" height="6.5" rx="1.2" fill="none" stroke="currentColor" stroke-width="1.3" />
            <path d="M5 7V5a3 3 0 0 1 6 0v2" fill="none" stroke="currentColor" stroke-width="1.3" />
          </svg>
        {:else}
          <svg viewBox="0 0 16 16" width="13" height="13" aria-hidden="true">
            <rect x="3.5" y="7" width="9" height="6.5" rx="1.2" fill="none" stroke="currentColor" stroke-width="1.3" />
            <path d="M5 7V5a3 3 0 0 1 5.7-1.3" fill="none" stroke="currentColor" stroke-width="1.3" stroke-linecap="round" />
          </svg>
        {/if}
      </IconButton>
      {#if onToggleLock}
        <IconButton
          class="btn icon-btn field-icon-btn"
          active={fields[i].locked}
          title={fields[i].locked ? t('tooltip.lockedField') : t('tooltip.lockField')}
          on:click={() => toggleLock(i)}
        >
          <svg viewBox="0 0 16 16" width="13" height="13" aria-hidden="true">
            <rect x="3" y="7" width="10" height="6.5" rx="1.2" fill={fields[i].locked ? 'currentColor' : 'none'} stroke="currentColor" stroke-width="1.3" />
            <path d="M5.2 7V5.2a2.8 2.8 0 0 1 5.6 0V7" fill="none" stroke="currentColor" stroke-width="1.3" />
            <circle cx="8" cy="10.2" r="1" fill={fields[i].locked ? 'var(--sm-panel-header)' : 'currentColor'} />
          </svg>
        </IconButton>
      {/if}
      <IconButton class="btn icon-btn field-icon-btn" title={t('tooltip.removeField')} on:click={() => remove(i)}>{t('text.removeGlyph')}</IconButton>
    </div>
    {#if errors[i]}
      <div class="field-error">{errors[i]}</div>
    {/if}
  {/each}
  <button class="btn" type="button" on:click={add}>{t('button.addField')}</button>
</div>

<style>
  .field-grid {
    display: flex;
    flex-direction: column;
    gap: 6px;
  }
  .field-row {
    display: flex;
    gap: 4px;
    align-items: flex-start;
  }
  .field-key {
    flex: 0 0 140px;
  }
  .field-kind {
    flex: 0 0 90px;
  }
  .field-value {
    flex: 1;
    min-width: 0;
  }
  .field-value-yaml {
    font-family: var(--sm-font-mono);
    font-size: var(--sm-type-sm);
  }

  .field-value-secret {
    -webkit-text-security: disc;
  }

  .field-error {
    color: var(--sm-bg-primary);
    font-size: var(--sm-type-sm);
    margin: -2px 0 4px 144px;
  }
  :global(.field-icon-btn) {
    display: flex;
    align-items: center;
    justify-content: center;
    box-sizing: border-box;
    width: 28px;
    height: 28px;
    padding: 0;
    flex: none;
  }
  :global(.field-icon-btn.active) {
    background: var(--sm-bg-primary);
    border-color: var(--sm-bg-primary);
    color: var(--sm-text-primary);
  }
  input,
  select,
  textarea {
    background: var(--sm-bg-deep);
    color: var(--sm-text);
    border: 1px solid var(--sm-border);
    border-radius: 4px;
    padding: 4px 6px;
    font-family: inherit;
    font-size: var(--sm-type-base);
  }

  .sm-select {
    background-position: right 8px center;
    padding-right: 24px;
  }
</style>
