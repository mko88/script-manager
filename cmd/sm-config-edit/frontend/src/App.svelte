<script lang="ts">
  import { onMount } from 'svelte'
  import Toast from '@shared/components/Toast.svelte'
  import { flash } from '@shared/toast'
  import { getTheme, getThemes, type Theme, type CustomPalette } from '@shared/theme'
  import StringListEditor from './components/StringListEditor.svelte'
  import FieldGrid from './components/FieldGrid.svelte'
  import ThemeEditor from './components/ThemeEditor.svelte'
  import MessagesEditor from './components/MessagesEditor.svelte'
  import DisplaysEditor from './components/DisplaysEditor.svelte'
  import ActionGroupsEditor from './components/ActionGroupsEditor.svelte'
  import ActionsEditor from './components/ActionsEditor.svelte'
  import ItemsEditor from './components/ItemsEditor.svelte'
  import ToolbarIcon from './components/ToolbarIcon.svelte'
  import RadioGroup from './components/RadioGroup.svelte'
  import Icon from '@shared/components/Icon.svelte'
  import IconButton from '@shared/components/IconButton.svelte'
  import { t } from './messages'
  import {
    InitialState,
    NewBlank,
    BrowseOpen,
    BrowseSaveAs,
    BrowseScriptFile,
    PreviewScriptFile,
    Save,
    OpenInEditor,
    OpenDataFolder,
    DataFolderPath,
    PreviewItem,
    PreviewAction,
    ValidateConfig,
    ValidateField,
    KnownTerminals,
    GetEditableMessages,
    GetDefaultMessages,
    SaveMessages,
    SetTheme,
    SaveTheme,
    DeleteTheme,
  } from '../wailsjs/go/configedit/App.js'
  import type { configedit } from '../wailsjs/go/models'

  function emptyConfig(): configedit.ConfigDTO {
    return {
      shell: [],
      display: [],
      terminal: { mode: 'auto', name: '', argv: [] },
      envFields: [],
      items: [],
      actionGroups: [],
      actions: [],
    } as unknown as configedit.ConfigDTO
  }

  let cfg: configedit.ConfigDTO = emptyConfig()
  let path = ''

  let theme: Theme = getTheme()
  let themes: Record<string, CustomPalette> | null = getThemes()
  let themeEditor: ThemeEditor | undefined
  let messagesEditor: MessagesEditor | undefined

  let knownTerminals: string[] = []
  let validation: configedit.ValidationIssueDTO[] = []
  let initialized = false

  type Section =
    | 'items'
    | 'actionGroups'
    | 'actions'
    | 'display'
    | 'env'
    | 'shell'
    | 'terminal'
    | 'theme'
    | 'messages'
  const sections: { key: Section; label: string }[] = [
    { key: 'items', label: t('nav.items') },
    { key: 'actionGroups', label: t('nav.actionGroups') },
    { key: 'actions', label: t('nav.actions') },
    { key: 'display', label: t('nav.displays') },
    { key: 'env', label: t('nav.environment') },
    { key: 'shell', label: t('nav.shell') },
    { key: 'terminal', label: t('nav.terminal') },
    { key: 'theme', label: t('nav.theme') },
    { key: 'messages', label: t('nav.messages') },
  ]
  let section: Section = 'items'
  $: sectionTitle = sections.find((s) => s.key === section)?.label ?? ''

  let selectedItem = -1
  let selectedActionGroup = -1
  let selectedAction = -1
  let selectedDisplay = -1
  let dataFolderPath = ''

  onMount(async () => {
    const state = await InitialState()
    applyState(state)
    knownTerminals = await KnownTerminals()
    dataFolderPath = await DataFolderPath()
    initialized = true
  })

  let cleanSnapshot = ''
  function markClean() {
    cleanSnapshot = JSON.stringify(cfg)
  }

  function applyState(state: configedit.StateDTO) {
    cfg = state.config
    path = state.path
    markClean()
    if (state.warning) flash(t('toast.configLoadWarning', { warning: state.warning }))
  }

  function resetSelection() {
    selectedItem = -1
    selectedActionGroup = -1
    selectedAction = -1
    selectedDisplay = -1
  }

  let validateTimer: ReturnType<typeof setTimeout>
  function scheduleValidate() {
    clearTimeout(validateTimer)
    validateTimer = setTimeout(async () => {
      validation = await ValidateConfig(cfg)
    }, 300)
  }

  $: dirty = initialized && JSON.stringify(cfg) !== cleanSnapshot

  $: if (initialized && cfg) scheduleValidate()

  $: hasBlockingError = validation.some((v) => v.severity === 'error')

  async function confirmDiscard(): Promise<boolean> {
    if (!dirty) return true
    return confirm(t('confirm.discardUnsaved'))
  }

  async function newConfig() {
    if (!(await confirmDiscard())) return
    applyState(await NewBlank())
    resetSelection()
  }

  async function openConfig() {
    if (!(await confirmDiscard())) return
    try {
      const state = await BrowseOpen()
      applyState(state)
      resetSelection()
    } catch (err) {
      flash(t('toast.openFailed', { error: String(err) }))
    }
  }

  async function doSave(target: string) {
    try {
      const result = await Save(cfg, target)
      path = result.path
      markClean()
      flash(t('toast.saved'))
    } catch (err) {
      flash(t('toast.saveFailed', { error: String(err) }))
    }
  }

  async function saveConfig() {
    if (hasBlockingError) {
      flash(t('toast.fixBlockingErrors'))
      return
    }
    if (path) {
      await doSave(path)
      return
    }
    const target = await BrowseSaveAs()
    if (target) await doSave(target)
  }

  async function saveAll() {
    await saveConfig()
    themeEditor?.save()
    messagesEditor?.save()
  }

  async function saveAsConfig() {
    if (hasBlockingError) {
      flash(t('toast.fixBlockingErrors'))
      return
    }
    const target = await BrowseSaveAs()
    if (target) await doSave(target)
  }

  async function openInEditor() {
    if (!path) return
    try {
      await OpenInEditor()
    } catch (err) {
      flash(t('toast.openInEditorFailed', { error: String(err) }))
    }
  }

  async function openDataFolder() {
    if (!dataFolderPath) return
    try {
      await OpenDataFolder()
    } catch (err) {
      flash(t('toast.openDataFolderFailed', { error: String(err) }))
    }
  }

  function handleGlobalKeydown(e: KeyboardEvent) {
    if (!(e.ctrlKey || e.metaKey)) return
    switch (e.key.toLowerCase()) {
      case 'n':
        e.preventDefault()
        newConfig()
        break
      case 'o':
        e.preventDefault()
        openConfig()
        break
      case 's':
        e.preventDefault()
        if (e.shiftKey) saveAsConfig()
        else saveAll()
        break
    }
  }

  $: allActionGroups = cfg.actionGroups.map((g) => g.id).filter((id) => id)

</script>

<svelte:window on:keydown={handleGlobalKeydown} />

<div class="app-root">
  <header class="toolbar">
    <IconButton title={t('tooltip.newTitle')} aria={t('tooltip.newAria')} on:click={newConfig}><ToolbarIcon mode="new" /></IconButton>
    <IconButton title={t('tooltip.openTitle')} aria={t('tooltip.openAria')} on:click={openConfig}><ToolbarIcon mode="open" /></IconButton>
    <IconButton
      title={t('tooltip.saveTitle')}
      aria={t('tooltip.saveAria')}
      disabled={hasBlockingError}
      on:click={saveAll}><ToolbarIcon mode="save" /></IconButton
    >
    <IconButton
      title={t('tooltip.saveAsTitle')}
      aria={t('tooltip.saveAsAria')}
      disabled={hasBlockingError}
      on:click={saveAsConfig}><ToolbarIcon mode="save-as" /></IconButton
    >
    <IconButton
      class="btn icon-btn open-data-folder-btn"
      disabled={!dataFolderPath}
      title={dataFolderPath ? t('tooltip.openDataFolderTitle', { path: dataFolderPath }) : ''}
      aria={t('tooltip.openDataFolderAria')}
      on:click={openDataFolder}><Icon name="open" /></IconButton
    >
    <IconButton
      class="btn icon-btn open-in-editor-btn"
      disabled={!path}
      title={path ? t('tooltip.openInEditorTitle', { path }) : ''}
      aria={t('tooltip.openInEditorAria')}
      on:click={openInEditor}><Icon name="edit" /></IconButton
    >
  </header>

  <main class="app-shell">
    {#if validation.length > 0}
      <div class="validation-banner">
        {#each validation as issue}
          <div class="validation-issue" class:validation-error={issue.severity === 'error'}>
            {issue.severity === 'error' ? t('text.errorIcon') : t('text.warningIcon')}
            {issue.message}
          </div>
        {/each}
      </div>
    {/if}

    <div class="body">
      <section class="panel section-nav">
        <header class="panel-title"><span>{t('panel.sections')}</span></header>
        <nav class="panel-body list">
          {#each sections as s (s.key)}
            <button class="row" class:selected={section === s.key} on:click={() => (section = s.key)}>{s.label}</button>
          {/each}
        </nav>
      </section>

      <section class="panel main-panel">
        <header class="panel-title"><span>{sectionTitle}</span></header>
        <div
          class="panel-body"
          class:list-body={section === 'items' || section === 'actionGroups' || section === 'actions' || section === 'theme' || section === 'messages'}
        >
        {#if section === 'shell'}
          <p class="hint">{t('hint.shellCommandPrefix')}<code>pwsh -NoLogo -Command</code>.</p>
          <StringListEditor
            bind:items={cfg.shell}
            placeholder={t('placeholder.shellCommand')}
            confirmRemoveMessage={(value) => t('confirm.removeShellEntry', { value: value || t('fallback.unnamed') })}
          />
        {:else if section === 'terminal'}
          <RadioGroup
            bind:value={cfg.terminal.mode}
            options={[
              { value: 'auto', label: t('radio.autoDetect') },
              { value: 'name', label: t('radio.named') },
              { value: 'argv', label: t('radio.customCommand') },
            ]}
          />
          {#if cfg.terminal.mode === 'name'}
            <label class="field">
              <span>{t('field.terminalName')}</span>
              <input type="text" list="known-terminals" bind:value={cfg.terminal.name} placeholder={t('placeholder.terminalName')} />
              <datalist id="known-terminals">
                {#each knownTerminals as name}<option value={name} />{/each}
              </datalist>
            </label>
          {:else if cfg.terminal.mode === 'argv'}
            <p class="hint">
              {t('hint.terminalArgvPrefix')}<code>{'{{title}}'}</code>/<code
                >{'{{dir}}'}</code
              >{t('hint.terminalArgvSuffix')}
            </p>
            <StringListEditor bind:items={cfg.terminal.argv} placeholder={t('placeholder.terminalArgv')} />
          {/if}
        {:else if section === 'env'}
          <p class="hint">{t('hint.envGlobal')}</p>
          <FieldGrid bind:fields={cfg.envFields} validateField={ValidateField} />
        {:else if section === 'display'}
          <DisplaysEditor
            bind:displays={cfg.display}
            bind:selectedDisplay
            items={cfg.items}
            envFields={cfg.envFields}
            previewItem={PreviewItem}
          />
        {:else if section === 'actionGroups'}
          <ActionGroupsEditor
            bind:actionGroups={cfg.actionGroups}
            bind:items={cfg.items}
            bind:actions={cfg.actions}
            bind:selectedActionGroup
          />
        {:else if section === 'actions'}
          <ActionsEditor
            bind:actions={cfg.actions}
            bind:selectedAction
            {allActionGroups}
            browseScriptFile={BrowseScriptFile}
            previewScriptFile={PreviewScriptFile}
          />
        {:else if section === 'items'}
          <ItemsEditor
            bind:items={cfg.items}
            bind:selectedItem
            actions={cfg.actions}
            {allActionGroups}
            displays={cfg.display}
            envFields={cfg.envFields}
            previewItem={PreviewItem}
            previewAction={PreviewAction}
            validateField={ValidateField}
            browseScriptFile={BrowseScriptFile}
            previewScriptFile={PreviewScriptFile}
          />
        {:else if section === 'theme'}
          <ThemeEditor
            bind:this={themeEditor}
            bind:theme
            bind:themes
            saveTheme={SaveTheme}
            deleteTheme={DeleteTheme}
            setActiveTheme={SetTheme}
            {flash}
          />
        {:else if section === 'messages'}
          <MessagesEditor
            bind:this={messagesEditor}
            getEditableMessages={GetEditableMessages}
            getDefaultMessages={GetDefaultMessages}
            saveMessages={SaveMessages}
          />
        {/if}
        </div>
      </section>
    </div>

    <Toast />
  </main>
</div>

<style>
  .app-root {
    display: flex;
    flex-direction: column;
    height: 100vh;
  }

  .toolbar {
    flex: none;
    display: flex;
    align-items: center;
    gap: 8px;
    padding: 6px 8px;
    background: var(--sm-panel-header);
    border-bottom: 1px solid var(--sm-border);
  }

  :global(.open-data-folder-btn) {
    margin-left: auto;
  }

  .app-shell {
    display: flex;
    flex-direction: column;
    flex: 1 1 auto;
    min-height: 0;
    box-sizing: border-box;
    padding: 8px;
    gap: 8px;
    text-align: left;
  }

  .validation-banner {
    flex: none;
    display: flex;
    flex-direction: column;
    gap: 2px;
    max-height: 120px;
    overflow-y: auto;
    background: rgba(232, 163, 61, 0.1);
    border: 1px solid var(--sm-border);
    border-radius: 6px;
    padding: 6px 10px;
  }

  .validation-issue {
    font-size: 0.8rem;
    color: var(--sm-text-muted);
  }

  .validation-issue.validation-error {
    color: var(--sm-bg-primary);
    font-weight: 700;
  }

  .body {
    flex: 1 1 auto;
    display: flex;
    gap: 8px;
    min-height: 0;
  }

  .section-nav {
    flex: 0 0 160px;
  }

  .main-panel {
    flex: 1 1 auto;
    min-width: 0;
  }

  .panel-body.list-body {
    display: flex;
    flex-direction: column;
    overflow-y: hidden;
  }

  .hint {
    color: var(--sm-text-muted);
    font-size: 0.8rem;
    margin: 0 0 8px;
  }

  .hint code {
    background: var(--sm-bg-deep);
    padding: 1px 4px;
    border-radius: 3px;
  }

</style>
