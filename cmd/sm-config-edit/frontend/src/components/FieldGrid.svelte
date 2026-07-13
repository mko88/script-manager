<script lang="ts">
  import { t } from '../messages'
  import { looksLikeSecretKey } from '../secretKey'
  import IconButton from '@shared/components/IconButton.svelte'

  // Drives a []FieldDTO (Environment, or an item's non-reserved "Additional
  // Fields"): a key, a kind selector, a kind-appropriate value widget, and a
  // lock toggle — the frontend half of internal/configedit's classify/decode
  // scheme, which avoids needing a widget per possible YAML shape. Secret is
  // independent of kind (unlike the old "password" kind it replaces), so a
  // multi-line value can be masked too, not just a plain string.
  export let fields: { key: string; kind: string; value: string; secret: boolean }[] = []
  export let validateField: (kind: string, value: string) => Promise<string> = async () => ''

  const kinds = ['string', 'multiline', 'number', 'bool', 'yaml'] as const

  // A fixed-length placeholder for an at-rest secret field — unlike
  // -webkit-text-security (which masks the real value character-for-character,
  // still revealing its length), this gives away nothing until the field is
  // focused, matching the masked-field convention seen in most other forms.
  const secretPlaceholder = t('text.secretMask')

  let errors: Record<number, string> = {}
  let focused: Record<number, boolean> = {}

  function onValueInput(i: number, e: Event) {
    fields[i].value = (e.currentTarget as HTMLInputElement | HTMLTextAreaElement).value
    check(i)
  }

  function add() {
    fields = [...fields, { key: '', kind: 'string', value: '', secret: false }]
  }
  function remove(i: number) {
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

  // Defaults a field to secret (masked) the moment its key looks like it
  // holds one, without waiting for a config reload to notice. Only while
  // secret is still at its "false" default: once the user has toggled the
  // lock themselves, further key edits shouldn't override that.
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
          value={fields[i].secret && !focused[i] ? secretPlaceholder : fields[i].value}
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
    font-family: "SF Mono", Consolas, monospace;
    font-size: 0.8rem;
  }

  /* Applied only while a secret field is focused — at rest it already shows
     the fixed secretPlaceholder text as-is, nothing further to hide. Masks
     the real value being typed/edited character-for-character; a
     non-standard but WebKit/Blink-supported property, which covers both of
     Wails' engines (WebKitGTK on Linux, Chromium-based WebView2 on Windows)
     — so a textarea (multiline/yaml) can be masked too, unlike a native
     type="password" input. */
  .field-value-secret {
    -webkit-text-security: disc;
  }

  .field-error {
    color: var(--sm-accent-warm);
    font-size: 0.75rem;
    margin: -2px 0 4px 144px;
  }
  /* A fixed box (not just matching padding) so the lock's SVG and the
     remove button's "✕" text glyph — different intrinsic content sizes —
     still render at identical dimensions. field-icon-btn (not the plain
     .icon-btn every toolbar button also carries), and :global since these
     buttons now render inside IconButton's own template — narrowly scoped
     so this fixed sizing doesn't leak to every icon button in the app. */
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
    background: var(--sm-accent-warm);
    border-color: var(--sm-accent-warm);
    color: var(--sm-accent-warm-text);
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
    font-size: 0.85rem;
  }

  /* appearance/arrow-image/light-theme-variant come from the shared
     .sm-select utility (@shared/theme.css) — only the tighter spacing
     FieldGrid's compact rows need is overridden locally here. */
  .sm-select {
    background-position: right 8px center;
    padding-right: 24px;
  }
</style>
