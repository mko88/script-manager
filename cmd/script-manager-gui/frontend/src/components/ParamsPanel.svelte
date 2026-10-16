<script lang="ts">
  import { onMount } from 'svelte'
  import { flash } from '@shared/toast'
  import { askConfirm } from '@shared/confirm'
  import { t } from '../messages'
  import { GetPresets, SavePreset, DeletePreset } from '../../wailsjs/go/gui/App.js'
  import { presets, scriptparams } from '../../wailsjs/go/models'

  export let itemIndex: number
  export let actionIndex: number
  export let spec: scriptparams.Spec | null
  export let onChange: (values: Record<string, any>, args: string) => void
  export let askName: (initial: string, existing: string[]) => Promise<string | null>

  let values: Record<string, any> = {}
  let args = ''
  let presetList: presets.Preset[] = []
  let selectedPreset = ''

  $: params = spec?.documented ? (spec.params ?? []) : []
  $: documented = params.length > 0
  $: onChange(values, args)

  onMount(async () => {
    const dto = await GetPresets(itemIndex, actionIndex)
    presetList = dto.presets ?? []
    if (dto.warning) flash(t('toast.presetsWarning', { warning: dto.warning }))
  })

  function fieldId(name: string): string {
    return `param-${itemIndex}-${actionIndex}-${name}`
  }

  function isMulti(p: scriptparams.Param): boolean {
    return p.kind === 'multiChoice' || p.kind === 'multiValue'
  }

  function list(v: unknown): string[] {
    if (Array.isArray(v)) return v.map(String)
    if (typeof v === 'string' && v !== '') return [v]
    return []
  }

  // Matches the backend: a flag is on for true or "true", in any case.
  function isOn(v: unknown): boolean {
    return v === true || (typeof v === 'string' && v.toLowerCase() === 'true')
  }

  function applyPreset(name: string) {
    selectedPreset = name
    const p = presetList.find((x) => x.name === name)
    if (!p) {
      values = {}
      args = ''
      return
    }
    const next: Record<string, any> = { ...(p.values ?? {}) }
    for (const param of params) {
      if (!(param.name in next)) continue
      if (isMulti(param)) next[param.name] = list(next[param.name])
      else if (param.kind === 'flag') next[param.name] = isOn(next[param.name])
    }
    values = next
    args = p.args ?? ''
  }

  function setMultiChoice(name: string, e: Event) {
    const select = e.currentTarget as HTMLSelectElement
    values[name] = Array.from(select.selectedOptions).map((o) => o.value)
  }

  function setMultiValue(name: string, e: Event) {
    values[name] = (e.currentTarget as HTMLTextAreaElement).value.split('\n')
  }

  // Matches the backend's UnknownNames: a value counts when it would pass
  // something.
  function carries(v: unknown): boolean {
    if (v === true || typeof v === 'number') return true
    if (Array.isArray(v)) return v.some((s) => s !== null && s !== undefined && String(s) !== '')
    return typeof v === 'string' && v !== '' && v.toLowerCase() !== 'false' && v !== '0'
  }

  $: paramNames = new Set(params.map((p) => p.name))
  $: staleNames = Object.keys(values)
    .filter((k) => !paramNames.has(k) && carries(values[k]))
    .sort()

  // Leaves out what would pass nothing, and anything the script doesn't
  // document, so Update drops values a preset no longer applies to.
  function savedValues(): Record<string, any> {
    const out: Record<string, any> = {}
    for (const [k, v] of Object.entries(values)) {
      if (!paramNames.has(k)) continue
      if (v === true) out[k] = true
      else if (Array.isArray(v)) {
        const l = v.map((s) => String(s).trim()).filter((s) => s !== '')
        if (l.length > 0) out[k] = l
      } else if (typeof v === 'string' && v !== '') out[k] = v
    }
    return out
  }

  async function save(name: string) {
    try {
      const dto = await SavePreset(itemIndex, actionIndex, presets.Preset.createFrom({ name, values: savedValues(), args: args.trim() }))
      presetList = dto.presets ?? []
      selectedPreset = name
      flash(t('toast.presetSaved', { name }))
    } catch (err) {
      flash(t('toast.presetFailed', { error: String(err) }))
    }
  }

  async function saveAs() {
    const name = await askName(selectedPreset, presetList.map((p) => p.name))
    if (name) await save(name)
  }

  async function update() {
    if (selectedPreset) await save(selectedPreset)
  }

  async function remove() {
    const name = selectedPreset
    if (!name) return
    const ok = await askConfirm({
      title: t('params.deleteTitle'),
      message: t('params.deleteMessage', { name }),
      confirmLabel: t('params.deleteConfirm'),
      cancelLabel: t('params.deleteCancel'),
    })
    if (!ok) return
    try {
      const dto = await DeletePreset(itemIndex, actionIndex, name)
      presetList = dto.presets ?? []
      selectedPreset = ''
      flash(t('toast.presetDeleted', { name }))
    } catch (err) {
      flash(t('toast.presetDeleteFailed', { error: String(err) }))
    }
  }
</script>

<div class="params">
  <div class="preset-row">
    <label class="field preset-field">
      <span>{t('params.preset')}</span>
      <select value={selectedPreset} on:change={(e) => applyPreset(e.currentTarget.value)}>
        <option value="">{t('params.noPreset')}</option>
        {#each presetList as p (p.name)}
          <option value={p.name}>{p.name}</option>
        {/each}
      </select>
    </label>
    <button class="btn" type="button" on:click={saveAs}>{t('params.savePreset')}</button>
    <button class="btn" type="button" title={t('params.updatePresetTitle')} disabled={!selectedPreset} on:click={update}
      >{t('params.updatePreset')}</button
    >
    <button class="btn" type="button" disabled={!selectedPreset} on:click={remove}>{t('params.deletePreset')}</button>
  </div>

  {#if staleNames.length > 0}
    <p class="stale-note">
      {documented
        ? t('params.staleValues', { names: staleNames.join(', ') })
        : t('params.staleValuesUndocumented', { names: staleNames.join(', ') })}
    </p>
  {/if}

  {#if documented}
    <div class="param-list">
      {#each params as p (p.name)}
        <div class="param">
          {#if p.kind === 'flag'}
            <label class="param-flag">
              <input type="checkbox" bind:checked={values[p.name]} />
              <span class="param-name">{p.name}</span>
              <code class="param-spelling">{p.flag}</code>
            </label>
          {:else}
            <label class="param-head" for={fieldId(p.name)}>
              <span class="param-name">{p.name}</span>
              {#if p.required}<span class="param-required" title={t('params.requiredTitle')}>{t('params.requiredMarker')}</span>{/if}
              {#if p.flag}<code class="param-spelling">{p.flag}</code>{/if}
            </label>
            <div class="field param-control">
              {#if p.kind === 'choice'}
                <select id={fieldId(p.name)} bind:value={values[p.name]}>
                  <option value="">{p.default ? t('params.defaultOption', { value: p.default }) : t('params.unsetOption')}</option>
                  {#each p.choices ?? [] as c (c)}
                    <option value={c}>{c}</option>
                  {/each}
                </select>
              {:else if p.kind === 'multiChoice'}
                <select
                  id={fieldId(p.name)}
                  class="multi"
                  multiple
                  size={Math.min((p.choices ?? []).length, 5)}
                  on:change={(e) => setMultiChoice(p.name, e)}
                >
                  {#each p.choices ?? [] as c (c)}
                    <option value={c} selected={list(values[p.name]).includes(c)}>{c}</option>
                  {/each}
                </select>
              {:else if p.kind === 'multiValue'}
                <textarea
                  id={fieldId(p.name)}
                  rows="2"
                  placeholder={p.default ? t('params.defaultOption', { value: p.default }) : t('params.multiValueHint')}
                  value={list(values[p.name]).join('\n')}
                  on:input={(e) => setMultiValue(p.name, e)}
                ></textarea>
              {:else}
                <input
                  id={fieldId(p.name)}
                  type="text"
                  placeholder={p.default ? t('params.defaultOption', { value: p.default }) : ''}
                  bind:value={values[p.name]}
                />
              {/if}
            </div>
          {/if}
          {#if p.description}<div class="param-desc">{p.description}</div>{/if}
        </div>
      {/each}
    </div>
  {/if}

  <label class="field args-field">
    <span>{documented ? t('params.extraArguments') : t('params.arguments')}</span>
    <input type="text" bind:value={args} placeholder={t('params.argumentsPlaceholder')} spellcheck="false" />
  </label>
</div>

<style>
  .params {
    display: flex;
    flex-direction: column;
    gap: 8px;
    padding-top: 6px;
  }

  .preset-row {
    display: flex;
    align-items: flex-end;
    flex-wrap: wrap;
    gap: 6px;
  }

  .preset-row .btn:disabled {
    opacity: 0.5;
    cursor: default;
  }

  .preset-field {
    flex: 1 1 160px;
    min-width: 0;
    margin: 0;
  }

  .stale-note {
    margin: 0;
    font-size: var(--sm-type-sm);
    color: var(--sm-warning);
  }

  .param-list {
    display: flex;
    flex-direction: column;
    gap: 8px;
  }

  .param-head,
  .param-flag {
    display: flex;
    align-items: baseline;
    gap: 6px;
    font-size: var(--sm-type-sm);
    color: var(--sm-text);
  }

  .param-flag {
    align-items: center;
  }

  .param-name {
    font-weight: 700;
  }

  .param-required {
    color: var(--sm-error);
    font-weight: 700;
  }

  .param-spelling {
    font-family: var(--sm-font-mono);
    color: var(--sm-text-muted);
  }

  .param-control {
    margin: 2px 0 0;
  }

  .param-control select.multi {
    background-image: none;
    padding-right: 7px;
  }

  .param-control textarea {
    resize: vertical;
    font-family: var(--sm-font-mono);
  }

  .param-desc {
    margin-top: 2px;
    font-size: var(--sm-type-sm);
    color: var(--sm-text-muted);
    white-space: pre-wrap;
  }

  .args-field {
    margin: 0;
  }

  .args-field input {
    font-family: var(--sm-font-mono);
  }
</style>
