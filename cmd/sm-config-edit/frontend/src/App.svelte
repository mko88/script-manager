<script lang="ts">
  import { onMount } from 'svelte'
  import Toast from '@shared/components/Toast.svelte'
  import { flash } from '@shared/toast'
  import { loadPersisted, savePersisted } from '@shared/persist'
  import { applyUIPrefs, clampScale, scaleFromKeydown, UI_SCALE } from '@shared/uiprefs'
  import { getTheme, getThemes, type Theme, type CustomPalette } from '@shared/theme'
  import StringListEditor from './components/StringListEditor.svelte'
  import FieldGrid from './components/FieldGrid.svelte'
  import ThemeEditor from './components/ThemeEditor.svelte'
  import MessagesEditor from './components/MessagesEditor.svelte'
  import SecretsEditor from './components/SecretsEditor.svelte'
  import DisplaysEditor from './components/DisplaysEditor.svelte'
  import ActionGroupsEditor from './components/ActionGroupsEditor.svelte'
  import ActionsEditor from './components/ActionsEditor.svelte'
  import ItemsEditor from './components/ItemsEditor.svelte'
  import ToolbarIcon from './components/ToolbarIcon.svelte'
  import RadioGroup from './components/RadioGroup.svelte'
  import Icon from '@shared/components/Icon.svelte'
  import IconButton from '@shared/components/IconButton.svelte'
  import RecentMenu from '@shared/components/RecentMenu.svelte'
  import PinDialog from '@shared/components/PinDialog.svelte'
  import { EventsOn, WindowMinimise, WindowToggleMaximise, Quit } from '../wailsjs/runtime'
  import { t } from './messages'
  import { shellLanguage } from './lib/shellLanguage'
  import {
    InitialState,
    NewBlank,
    BrowseOpen,
    RecentConfigs,
    GetUIPrefs,
    SetUIScale,
    SetUIFonts,
    WatchScript,
    OpenScriptInEditor,
    OpenRecent,
    ClearRecentConfigs,
    CreateSecretsPIN,
    ChangeSecretsPIN,
    RemoveSecretsPIN,
    UnlockSecrets,
    LockValue,
    RevealValue,
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
    | 'secrets'
  const sections: { key: Section; label: string }[] = [
    { key: 'items', label: t('nav.items') },
    { key: 'actionGroups', label: t('nav.actionGroups') },
    { key: 'actions', label: t('nav.actions') },
    { key: 'display', label: t('nav.displays') },
    { key: 'env', label: t('nav.environment') },
    { key: 'shell', label: t('nav.shell') },
    { key: 'terminal', label: t('nav.terminal') },
    { key: 'secrets', label: t('nav.pin') },
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
    await refreshRecents()
    initialized = true
  })

  let cleanSnapshot = ''
  function markClean() {
    cleanSnapshot = JSON.stringify(cfg)
  }

  function applyState(state: configedit.StateDTO) {
    cfg = state.config
    path = state.path
    secretsUnlocked = false
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

  $: cmdLanguage = shellLanguage(cfg.shell?.[0])

  // Bumped when a watched script is written, which re-runs the preview.
  let scriptReloadToken = 0
  onMount(() => EventsOn('script:changed', () => (scriptReloadToken += 1)))

  async function openScriptInEditor(path: string) {
    try {
      await OpenScriptInEditor(path)
    } catch (err) {
      flash(t('toast.openScriptFailed', { error: String(err) }))
    }
  }

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
      await refreshRecents()
    } catch (err) {
      flash(t('toast.openFailed', { error: String(err) }))
    }
  }

  let uiScale: number = UI_SCALE.default
  let uiPrefs: { fontUi?: string; fontMono?: string; scalePercent?: number } = {}

  onMount(async () => {
    uiPrefs = await GetUIPrefs()
    uiScale = clampScale(uiPrefs.scalePercent)
    applyUIPrefs(uiPrefs)
  })

  async function onScaleKeydown(e: KeyboardEvent) {
    const next = scaleFromKeydown(e, uiScale)
    if (next === null) return
    e.preventDefault()
    if (next === uiScale) return
    uiScale = next
    uiPrefs = await SetUIScale(next)
    applyUIPrefs(uiPrefs)
  }

  async function saveFonts(fontUi: string, fontMono: string) {
    uiPrefs = await SetUIFonts(fontUi, fontMono)
    uiScale = clampScale(uiPrefs.scalePercent)
    applyUIPrefs(uiPrefs)
    flash(t('toast.fontsSaved'))
  }

  let recents: string[] = []

  async function refreshRecents() {
    recents = await RecentConfigs()
  }

  let pinDialogOpen = false
  let pinError = ''
  let secretsUnlocked = false
  let pinResolve: ((ok: boolean) => void) | null = null

  function askPin(): Promise<boolean> {
    pinError = ''
    pinDialogOpen = true
    return new Promise((resolve) => (pinResolve = resolve))
  }

  async function submitPin(pin: string) {
    if (!cfg.secrets) return
    try {
      await UnlockSecrets(pin, cfg.secrets)
    } catch (err) {
      pinError = String(err)
      return
    }
    secretsUnlocked = true
    pinDialogOpen = false
    pinResolve?.(true)
    pinResolve = null
  }

  function cancelPin() {
    pinDialogOpen = false
    pinError = ''
    pinResolve?.(false)
    pinResolve = null
  }

  async function ensureUnlocked(): Promise<boolean> {
    if (secretsUnlocked) return true
    if (!cfg.secrets) {
      flash(t('toast.pinNotSet'))
      section = 'secrets'
      return false
    }
    return askPin()
  }

  async function toggleFieldLock(field: { key: string; value: string; locked?: boolean }): Promise<string | null> {
    if (!(await ensureUnlocked())) return null
    try {
      return field.locked ? await RevealValue(field.value) : await LockValue(field.value)
    } catch (err) {
      flash(t('toast.lockFailed', { error: String(err) }))
      return null
    }
  }

  $: lockedValueCount =
    (cfg?.envFields ?? []).filter((f) => f.locked).length +
    (cfg?.items ?? []).reduce((n, it) => n + (it.fields ?? []).filter((f) => f.locked).length, 0)

  async function setPin(newPin: string): Promise<string> {
    try {
      cfg.secrets = await CreateSecretsPIN(newPin)
      cfg = cfg
      secretsUnlocked = true
      flash(t('toast.pinSet'))
      return ''
    } catch (err) {
      return String(err)
    }
  }

  async function changePin(oldPin: string, newPin: string): Promise<string> {
    try {
      cfg = await ChangeSecretsPIN(oldPin, newPin, cfg)
      secretsUnlocked = true
      flash(t('toast.pinChanged'))
      return ''
    } catch (err) {
      return String(err)
    }
  }

  async function removePin(pin: string): Promise<string> {
    try {
      cfg = await RemoveSecretsPIN(pin, cfg)
      secretsUnlocked = false
      flash(t('toast.pinRemoved'))
      return ''
    } catch (err) {
      return String(err)
    }
  }

  async function openRecent(recentPath: string) {
    if (!(await confirmDiscard())) return
    try {
      applyState(await OpenRecent(recentPath))
      resetSelection()
    } catch (err) {
      flash(t('toast.openFailed', { error: String(err) }))
    }
    await refreshRecents()
  }

  async function clearRecents() {
    recents = await ClearRecentConfigs()
  }

  async function doSave(target: string, silent = false) {
    try {
      const result = await Save(cfg, target)
      path = result.path
      markClean()
      await refreshRecents()
      if (!silent) flash(t('toast.saved'))
    } catch (err) {
      flash(t('toast.saveFailed', { error: String(err) }))
    }
  }

  const AUTOSAVE_DELAY_MS = 800
  let autosave = loadPersisted('sm-config-edit.autosave', { enabled: true }).enabled
  let autosaveTimer: ReturnType<typeof setTimeout>
  let autosaving = false

  function toggleAutosave() {
    autosave = !autosave
    savePersisted('sm-config-edit.autosave', { enabled: autosave })
    if (!autosave) clearTimeout(autosaveTimer)
  }

  $: if (autosave && initialized && dirty && path && !hasBlockingError) scheduleAutosave()

  function scheduleAutosave() {
    clearTimeout(autosaveTimer)
    autosaveTimer = setTimeout(async () => {
      if (!autosave || !path || hasBlockingError || !dirty || autosaving) return
      autosaving = true
      await doSave(path, true)
      autosaving = false
    }, AUTOSAVE_DELAY_MS)
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

<svelte:window on:keydown|capture={onScaleKeydown} on:keydown={handleGlobalKeydown} />

<div class="app-root">
  <header class="toolbar">
    <IconButton title={t('tooltip.newTitle')} aria={t('tooltip.newAria')} on:click={newConfig}><ToolbarIcon mode="new" /></IconButton>
    <RecentMenu
      title={t('tooltip.openTitle')}
      aria={t('tooltip.openAria')}
      {recents}
      currentPath={path}
      recentsHeading={t('tooltip.recentHeading')}
      emptyLabel={t('tooltip.recentEmpty')}
      clearLabel={t('tooltip.recentClear')}
      browseLabel={t('tooltip.recentBrowse')}
      onOpen={openRecent}
      onBrowse={openConfig}
      onClear={clearRecents}><ToolbarIcon mode="open" /></RecentMenu
    >
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
    <button
      class="save-state"
      class:save-state-on={autosave}
      type="button"
      aria-pressed={autosave}
      title={autosave ? t('tooltip.autosaveOn') : t('tooltip.autosaveOff')}
      on:click={toggleAutosave}
    >
      <span class="save-dot" class:save-dot-unsaved={dirty}></span>
      {autosave ? t('text.autosaveOn') : t('text.autosaveOff')}
      {#if dirty}<span class="save-state-detail">{t('text.unsaved')}</span>{/if}
    </button>
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
    <div class="window-controls">
      <IconButton title={t('tooltip.minimizeWindow')} on:click={() => WindowMinimise()}><Icon name="minimize" /></IconButton>
      <IconButton title={t('tooltip.maximizeWindow')} on:click={() => WindowToggleMaximise()}><Icon name="maximize" /></IconButton>
      <IconButton class="btn icon-btn window-close-btn" title={t('tooltip.closeWindow')} on:click={() => Quit()}><Icon name="cancel" /></IconButton>
    </div>
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
          <FieldGrid bind:fields={cfg.envFields} validateField={ValidateField} onToggleLock={toggleFieldLock} />
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
            {cmdLanguage}
            watchScript={WatchScript}
            {openScriptInEditor}
            {scriptReloadToken}
            browseScriptFile={BrowseScriptFile}
            previewScriptFile={PreviewScriptFile}
          />
        {:else if section === 'items'}
          <ItemsEditor
            bind:items={cfg.items}
            bind:selectedItem
            actions={cfg.actions}
            {allActionGroups}
            {cmdLanguage}
            watchScript={WatchScript}
            {openScriptInEditor}
            {scriptReloadToken}
            displays={cfg.display}
            envFields={cfg.envFields}
            previewItem={PreviewItem}
            previewAction={PreviewAction}
            validateField={ValidateField}
            browseScriptFile={BrowseScriptFile}
            previewScriptFile={PreviewScriptFile}
            onToggleLock={toggleFieldLock}
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
            fontUi={uiPrefs.fontUi ?? ''}
            fontMono={uiPrefs.fontMono ?? ''}
            {uiScale}
            onSaveFonts={saveFonts}
          />
        {:else if section === 'secrets'}
          <SecretsEditor
            configured={!!cfg.secrets}
            unlocked={secretsUnlocked}
            lockedCount={lockedValueCount}
            onSetPin={setPin}
            onChangePin={changePin}
            onRemovePin={removePin}
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

    <PinDialog
    open={pinDialogOpen}
    title={t('tooltip.pinEnterTitle')}
    message={t('tooltip.pinEnterMessage')}
    pinLabel={t('tooltip.pinLabel')}
    confirmLabel={t('tooltip.pinConfirmButton')}
    cancelLabel={t('tooltip.pinCancelButton')}
    error={pinError}
    onSubmit={submitPin}
    onCancel={cancelPin}
  />
</main>
</div>


<style>
  .app-root {
    display: flex;
    flex-direction: column;
    position: relative;
    /* zoom multiplies every length, and 100vh resolves against the unzoomed
       viewport — so the scaled root would be taller than the window (scroll
       bars) or shorter (dead space). Dividing first cancels the zoom out. */
    height: calc(100vh / var(--sm-ui-scale, 1));
  }

  .toolbar {
    --wails-draggable: drag;
    flex: none;
    display: flex;
    align-items: center;
    gap: 8px;
    padding: 6px 8px;
    background: var(--sm-panel-header);
    border-bottom: 1px solid var(--sm-border);
  }

  .toolbar :global(.btn),
  .toolbar :global(.recent-menu),
  .toolbar .save-state {
    --wails-draggable: no-drag;
  }

  .window-controls {
    display: flex;
    gap: 2px;
    margin-left: 8px;
  }

  :global(.window-close-btn:hover) {
    background: var(--sm-error);
    border-color: var(--sm-error);
    color: var(--sm-bg-alt);
  }

  .save-state {
    margin-left: auto;
    display: flex;
    align-items: center;
    gap: 6px;
    padding: 4px 8px;
    border: 1px solid transparent;
    border-radius: 4px;
    background: none;
    color: var(--sm-text-muted);
    font-family: inherit;
    font-size: var(--sm-type-sm);
    cursor: pointer;
  }

  .save-state:hover {
    border-color: var(--sm-border);
  }

  .save-state-on {
    color: var(--sm-text);
  }

  .save-dot {
    width: 7px;
    height: 7px;
    border-radius: 50%;
    border: 1px solid var(--sm-text-muted);
  }

  .save-dot-unsaved {
    background: var(--sm-warning);
    border-color: var(--sm-warning);
  }

  .save-state-detail {
    color: var(--sm-warning);
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
    background: var(--sm-warning-tint);
    border: 1px solid var(--sm-border);
    border-radius: 6px;
    padding: 6px 10px;
  }

  .validation-issue {
    font-size: var(--sm-type-sm);
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
    font-size: var(--sm-type-sm);
    margin: 0 0 8px;
  }

  .hint code {
    background: var(--sm-bg-deep);
    padding: 1px 4px;
    border-radius: 3px;
  }

</style>
