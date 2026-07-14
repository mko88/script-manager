<script lang="ts">
  import { t } from '../messages'
  import RadioGroup from './RadioGroup.svelte'
  import CheckboxChipList from './CheckboxChipList.svelte'
  import ScriptSource from '@shared/components/ScriptSource.svelte'
  import type { configedit } from '../../wailsjs/go/models'

  // Shared by the global Actions section and an item's Custom Actions list —
  // both edit the same []Action-shaped data (internal/configedit.ActionDTO).
  export let action: {
    id: string
    title: string
    description: string
    cmd: string
    script: string
    groups: string[]
    noWait: boolean
    interactive: boolean
  }
  export let showId = true
  // The Action Groups catalog's IDs — Groups is a picker against this list
  // (like Items' Actions/Action groups pickers), not free text, so an
  // action can only belong to a group that's actually been catalogued.
  export let allActionGroups: string[] = []
  // Opens a native file picker and returns the chosen path ("" if cancelled).
  export let browseScriptFile: () => Promise<string>
  // Reads path's content and returns it as syntax-highlighted HTML (or an
  // error, e.g. "no such file") for the script-file source preview below.
  export let previewScriptFile: (path: string) => Promise<configedit.ScriptPreviewDTO>

  // cmd and script are mutually exclusive. mode starts derived from which
  // one is currently populated, but from then on is tracked as its own
  // piece of state — deriving it from action.script's truthiness on every
  // keystroke would snap back to "cmd" the instant a user picks "Script
  // file" and the field is still empty (nothing typed yet). It's re-derived
  // only when a genuinely different action is bound (switching the selected
  // action/custom action), not on every edit to the same one.
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

  // Debounced the same way ItemsEditor's live preview is (250ms) — this
  // fires on every keystroke in the path field, not just on Browse.
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
</div>

<style>
  /* No gap here — .field already carries its own margin-bottom (shared
     theme.css), the same single spacing mechanism every other section's
     detail pane relies on. A flex gap on top of that was double-spacing
     every field (8px gap + 10px margin-bottom = 18px instead of 10px). */
  .action-form {
    display: flex;
    flex-direction: column;
  }
  .field {
    display: flex;
    flex-direction: column;
    gap: 2px;
    font-size: 0.8rem;
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
    font-size: 0.85rem;
  }
  .cmd-field textarea {
    font-family: "SF Mono", Consolas, monospace;
  }
  .script-path-row {
    display: flex;
    gap: 6px;
  }
  .script-path-row input {
    flex: 1 1 auto;
    min-width: 0;
    font-family: "SF Mono", Consolas, monospace;
  }
  /* The script-source preview itself comes from the shared
     @shared/components/ScriptSource.svelte, used identically by
     script-manager-gui's Command pane — no local styling needed here. */
  .field-checkbox {
    display: flex;
    align-items: center;
    gap: 6px;
    font-size: 0.8rem;
    color: var(--sm-text-muted);
    margin-bottom: 10px;
  }
  /* .checkbox-list/.checkbox-chip come from the shared design system
     (@shared/theme.css) — not redefined here. */
</style>
