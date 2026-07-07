<script lang="ts">
  // Drives a []FieldDTO (Environment, or an item's non-reserved "Additional
  // Fields"): a key, a kind selector, and a kind-appropriate value widget —
  // the frontend half of internal/configedit's classify/decode scheme, which
  // avoids needing a widget per possible YAML shape.
  export let fields: { key: string; kind: string; value: string }[] = []
  export let validateField: (kind: string, value: string) => Promise<string> = async () => ''

  const kinds = ['string', 'number', 'bool', 'yaml'] as const

  let errors: Record<number, string> = {}

  function add() {
    fields = [...fields, { key: '', kind: 'string', value: '' }]
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
</script>

<div class="field-grid">
  {#each fields as _, i (i)}
    <div class="field-row">
      <input class="field-key" type="text" placeholder="key" bind:value={fields[i].key} />
      <select class="field-kind" bind:value={fields[i].kind} on:change={() => onKindChange(i)}>
        {#each kinds as k}<option value={k}>{k}</option>{/each}
      </select>
      {#if fields[i].kind === 'bool'}
        <select class="field-value" bind:value={fields[i].value} on:change={() => check(i)}>
          <option value="true">true</option>
          <option value="false">false</option>
        </select>
      {:else if fields[i].kind === 'yaml'}
        <textarea
          class="field-value field-value-yaml"
          rows="2"
          bind:value={fields[i].value}
          on:input={() => check(i)}
        ></textarea>
      {:else}
        <input class="field-value" type="text" bind:value={fields[i].value} on:input={() => check(i)} />
      {/if}
      <button class="btn" type="button" title="Remove field" on:click={() => remove(i)}>✕</button>
    </div>
    {#if errors[i]}
      <div class="field-error">{errors[i]}</div>
    {/if}
  {/each}
  <button class="btn" type="button" on:click={add}>+ Add field</button>
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
  .field-error {
    color: var(--sm-accent-warm);
    font-size: 0.75rem;
    margin: -2px 0 4px 144px;
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

  /* See App.svelte's .field select for why appearance: none is needed —
     without it, Chromium/WebView2 keeps a native, light dropdown-arrow
     box regardless of the background/color set above. */
  select {
    appearance: none;
    -webkit-appearance: none;
    -moz-appearance: none;
    background-image: url("data:image/svg+xml,%3Csvg xmlns='http://www.w3.org/2000/svg' width='10' height='6' viewBox='0 0 10 6'%3E%3Cpath d='M1 1l4 4 4-4' fill='none' stroke='%23a9b6c8' stroke-width='1.4' stroke-linecap='round' stroke-linejoin='round'/%3E%3C/svg%3E");
    background-repeat: no-repeat;
    background-position: right 8px center;
    padding-right: 24px;
  }
</style>
