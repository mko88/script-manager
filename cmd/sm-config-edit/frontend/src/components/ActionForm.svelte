<script lang="ts">
  import { t } from '../messages'
  import RadioGroup from './RadioGroup.svelte'
  import CheckboxChipList from './CheckboxChipList.svelte'
  import ScriptSource from '@shared/components/ScriptSource.svelte'
  import type { configedit } from '../../wailsjs/go/models'

  export let action: {
    id: string
    title: string
    description: string
    cmd: string
    script: string
    groups: string[]
    noWait: boolean
    interactive: boolean
    requiresPin: boolean
  }
  export let showId = true
  export let allActionGroups: string[] = []
  export let browseScriptFile: () => Promise<string>
  export let previewScriptFile: (path: string) => Promise<configedit.ScriptPreviewDTO>

  let mode: 'cmd' | 'script' = action.script ? 'script' : 'cmd'
  let modeFor = action
  $: if (action !== modeFor) {
    modeFor = action
    mode = action.script ? 'script' : 'cmd'
    scriptPreview = null
  }
  function setMode(next: 'cmd' | 'script') {
    if (next === mode) return
    mode = next
    if (next === 'cmd') action.script = ''
    else action.cmd = ''
  }
  async function browseScript() {
    const path = await browseScriptFile()
    if (path) action.script = path
  }

  let scriptPreview: configedit.ScriptPreviewDTO | null = null
  let scriptPreviewTimer: ReturnType<typeof setTimeout>
  $: if (mode === 'script') scheduleScriptPreview(action.script)
  function scheduleScriptPreview(path: string) {
    clearTimeout(scriptPreviewTimer)
    if (!path) {
      scriptPreview = null
      return
    }
    scriptPreviewTimer = setTimeout(async () => {
      scriptPreview = await previewScriptFile(path)
    }, 250)
  }
</script>

<div class="action-form">
  {#if showId}
    <label class="field">
      <span>{t('field.id')}</span>
      <input type="text" bind:value={action.id} placeholder={t('placeholder.actionId')} />
    </label>
  {/if}
  <label class="field">
    <span>{t('field.title')}</span>
    <input type="text" bind:value={action.title} />
  </label>
  <label class="field">
    <span>{t('field.description')}</span>
    <textarea rows="3" bind:value={action.description}></textarea>
  </label>
  <RadioGroup
    value={mode}
    options={[
      { value: 'cmd', label: t('radio.cmd') },
      { value: 'script', label: t('radio.script') },
    ]}
    on:change={(e) => setMode(e.detail === 'script' ? 'script' : 'cmd')}
  />
  {#if mode === 'cmd'}
    <label class="field cmd-field">
      <textarea rows="3" bind:value={action.cmd}></textarea>
    </label>
  {:else}
    <label class="field cmd-field">
      <div class="script-path-row">
        <input type="text" bind:value={action.script} placeholder={t('placeholder.scriptPath')} />
        <button class="btn" type="button" on:click={browseScript}>{t('button.browse')}</button>
      </div>
    </label>
    {#if scriptPreview?.error}
      <div class="validation-issue validation-error">{scriptPreview.error}</div>
    {:else if scriptPreview?.content}
      <ScriptSource content={scriptPreview.content} />
    {/if}
  {/if}
  {#if allActionGroups.length > 0}
    <div class="field">
      <span>{t('field.groups')}</span>
      <CheckboxChipList options={allActionGroups} bind:selected={action.groups} />
    </div>
  {/if}
  <label class="field-checkbox">
    <input type="checkbox" bind:checked={action.noWait} />
    <span>{t('hint.noWaitCheckbox')}</span>
  </label>
  <label class="field-checkbox">
    <input type="checkbox" bind:checked={action.interactive} />
    <span>{t('hint.interactiveCheckbox')}</span>
  </label>
  <label class="field-checkbox">
    <input type="checkbox" bind:checked={action.requiresPin} />
    <span>{t('hint.requiresPinCheckbox')}</span>
  </label>
</div>

<style>
  .action-form {
    display: flex;
    flex-direction: column;
  }
  .field {
    display: flex;
    flex-direction: column;
    gap: 2px;
    font-size: var(--sm-type-sm);
    color: var(--sm-text-muted);
  }
  .field input,
  .field textarea {
    background: var(--sm-bg-deep);
    color: var(--sm-text);
    border: 1px solid var(--sm-border);
    border-radius: 4px;
    padding: 5px 7px;
    font-family: inherit;
    font-size: var(--sm-type-base);
  }
  .cmd-field textarea {
    font-family: var(--sm-font-mono);
  }
  .script-path-row {
    display: flex;
    gap: 6px;
  }
  .script-path-row input {
    flex: 1 1 auto;
    min-width: 0;
    font-family: var(--sm-font-mono);
  }
  .field-checkbox {
    display: flex;
    align-items: center;
    gap: 6px;
    font-size: var(--sm-type-sm);
    color: var(--sm-text-muted);
    margin-bottom: 10px;
  }
</style>
