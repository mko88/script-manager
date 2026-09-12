<script lang="ts">
  import { onMount, tick } from 'svelte'
  import { TOKEN_GROUPS, readPaletteFor, setTheme, type CustomPalette, type Theme } from '@shared/theme'
  import CollapseToggle from '@shared/components/CollapseToggle.svelte'
  import Icon from '@shared/components/Icon.svelte'
  import IconButton from '@shared/components/IconButton.svelte'
  import { t } from '../messages'
  import { copyLabel } from '../lib/duplicate'

  export let theme: Theme
  export let themes: Record<string, CustomPalette> | null
  export let saveTheme: (name: string, renamedFrom: string, palette: Record<string, string>) => Promise<void>
  export let deleteTheme: (name: string) => Promise<void>
  export let setActiveTheme: (active: string) => Promise<void>
  export let flash: (msg: string) => void
  export let fontUi = ''
  export let fontMono = ''
  export let uiScale = 100
  export let onSaveFonts: (fontUi: string, fontMono: string) => void

  const NEW_THEME_ENTRY = '__new-theme__'

  function isBuiltIn(name: string): name is 'dark' | 'light' {
    return name === 'dark' || name === 'light'
  }

  function loadSelection(name: string) {
    if (isBuiltIn(name)) {
      editedName = name === 'dark' ? t('theme.dark') : t('theme.light')
      palette = readPaletteFor(name)
      selectionAtLoad = ''
    } else {
      editedName = name
      palette = themes?.[name] ? { ...themes[name] } : readPaletteFor('dark')
      selectionAtLoad = name
    }
  }

  function nextDefaultName(): string {
    const base = t('themeEditor.newThemeDefaultName')
    const existing = new Set(Object.keys(themes ?? {}))
    if (!existing.has(base)) return base
    let n = 2
    while (existing.has(`${base} ${n}`)) n++
    return `${base} ${n}`
  }

  let selectedThemeName: string = theme
  let editedName = ''
  let selectionAtLoad = ''
  let palette: CustomPalette = {}
  let nameInputEl: HTMLInputElement | undefined
  loadSelection(selectedThemeName)

  $: isCustomSelected = selectedThemeName !== 'dark' && selectedThemeName !== 'light'
  $: isDraft = selectedThemeName === NEW_THEME_ENTRY
  $: canSave = isCustomSelected && editedName.trim().length > 0

  $: activeThemeLabel = theme === 'dark' ? t('theme.dark') : theme === 'light' ? t('theme.light') : theme

  function onThemeSelect() {
    if (selectedThemeName === NEW_THEME_ENTRY) {
      selectionAtLoad = ''
      editedName = nextDefaultName()
      palette = readPaletteFor('dark')
      tick().then(() => nameInputEl?.focus())
      return
    }
    loadSelection(selectedThemeName)
    theme = selectedThemeName
    setTheme(selectedThemeName, themes ?? undefined)
    setActiveTheme(selectedThemeName).catch(() => {})
  }

  function addTheme() {
    selectedThemeName = NEW_THEME_ENTRY
    onThemeSelect()
  }

  function copyTheme() {
    const baseName = editedName.trim() || activeThemeLabel
    const copiedPalette = { ...palette }
    selectionAtLoad = ''
    editedName = copyLabel(baseName, Object.keys(themes ?? {}))
    palette = copiedPalette
    selectedThemeName = NEW_THEME_ENTRY
  }

  function confirmResetFrom(base: 'dark' | 'light') {
    const target = base === 'dark' ? t('theme.dark') : t('theme.light')
    if (confirm(t('confirm.resetTheme', { target }))) resetFrom(base)
  }

  function resetToSaved() {
    if (!isCustomSelected || isDraft) return
    if (!confirm(t('confirm.resetTheme', { target: t('themeEditor.resetTargetSaved') }))) return
    loadSelection(selectionAtLoad)
  }

  const THEME_PANEL_KEY = 'sm-config-edit:themePanel'
  let themePanelCollapsed = false

  onMount(() => {
    try {
      const saved = JSON.parse(localStorage.getItem(THEME_PANEL_KEY) ?? '{}')
      themePanelCollapsed = !!saved.collapsed
    } catch {
    }
  })

  function persistThemePanelCollapsed() {
    localStorage.setItem(THEME_PANEL_KEY, JSON.stringify({ collapsed: themePanelCollapsed }))
  }

  const FONTS_GROUP_KEY = 'sm-config-edit:fontsGroup'
  let fontsCollapsed = true

  onMount(() => {
    try {
      const saved = JSON.parse(localStorage.getItem(FONTS_GROUP_KEY) ?? '{"collapsed":true}')
      fontsCollapsed = !!saved.collapsed
    } catch {
    }
  })

  function toggleFonts() {
    fontsCollapsed = !fontsCollapsed
    localStorage.setItem(FONTS_GROUP_KEY, JSON.stringify({ collapsed: fontsCollapsed }))
  }

  // Seeded from the saved prefs when they arrive, and again whenever they
  // change, without overwriting what's being typed in between.
  let fontUiDraft = ''
  let fontMonoDraft = ''
  let lastFontUi = ''
  let lastFontMono = ''
  $: if (fontUi !== lastFontUi) {
    lastFontUi = fontUi
    fontUiDraft = fontUi
  }
  $: if (fontMono !== lastFontMono) {
    lastFontMono = fontMono
    fontMonoDraft = fontMono
  }

  let collapsedGroups = new Set<string>()

  function toggleGroup(label: string) {
    const next = new Set(collapsedGroups)
    if (next.has(label)) next.delete(label)
    else next.add(label)
    collapsedGroups = next
  }

  function tokensForElement(el: Element): string[] {
    const found = new Set<string>()
    const tokenRe = /var\(\s*--sm-([\w-]+)/g
    function collect(style: CSSStyleDeclaration) {
      tokenRe.lastIndex = 0
      let m: RegExpExecArray | null
      while ((m = tokenRe.exec(style.cssText))) found.add(m[1])
    }
    function walk(rules: CSSRuleList) {
      for (const rule of Array.from(rules)) {
        if (rule instanceof CSSMediaRule || rule instanceof CSSSupportsRule) {
          walk(rule.cssRules)
          continue
        }
        if (!(rule instanceof CSSStyleRule)) continue
        try {
          if (el.matches(rule.selectorText)) collect(rule.style)
          const hoverless = rule.selectorText.includes(':hover') ? rule.selectorText.replace(/:hover/g, '') : ''
          if (hoverless && el.matches(hoverless)) collect(rule.style)
        } catch {
        }
      }
    }
    for (const sheet of Array.from(document.styleSheets)) {
      try {
        walk(sheet.cssRules)
      } catch {
      }
    }
    return Array.from(found)
  }

  let fieldFilter = ''
  function filterForElement(el: Element) {
    const tokens = tokensForElement(el)
    fieldFilter = (tokens.length > 0 ? tokens : previewPanelEl ? tokensForElement(previewPanelEl) : []).join(', ')
  }
  function onPreviewClick(e: MouseEvent) {
    filterForElement(e.currentTarget as Element)
  }
  function onNestedPreviewClick(e: MouseEvent) {
    e.stopPropagation()
    filterForElement(e.currentTarget as Element)
  }
  function onScrollbarClick() {
    fieldFilter = 'scrollbar'
  }

  function onBackgroundClick(e: MouseEvent) {
    if (e.target === e.currentTarget) filterForElement(e.currentTarget as Element)
  }

  let previewPanelEl: HTMLElement | undefined
  function onPreviewBodyClick(e: MouseEvent) {
    if (e.target === e.currentTarget && previewPanelEl) filterForElement(previewPanelEl)
  }

  $: filterTerms = fieldFilter
    .split(',')
    .map((s) => s.trim().toLowerCase())
    .filter(Boolean)
  $: visibleGroups = TOKEN_GROUPS.map((group) => ({
    label: group.label,
    tokens: filterTerms.length === 0 ? group.tokens : group.tokens.filter((name) => filterTerms.includes(name.toLowerCase())),
  })).filter((group) => group.tokens.length > 0)

  function resetFrom(base: 'dark' | 'light') {
    palette = readPaletteFor(base)
  }

  function isHexValue(v: string) {
    return /^#[0-9a-fA-F]{6}$/.test(v)
  }

  $: previewStyle = Object.entries(palette)
    .map(([name, value]) => `--sm-${name}: ${value}`)
    .join('; ')

  export async function save() {
    if (!canSave) return
    const name = editedName.trim()
    try {
      await saveTheme(name, selectionAtLoad, palette)
      const nextThemes: Record<string, CustomPalette> = { ...(themes ?? {}) }
      if (selectionAtLoad && selectionAtLoad !== name) delete nextThemes[selectionAtLoad]
      nextThemes[name] = { ...palette }
      themes = nextThemes
      theme = name
      selectedThemeName = name
      selectionAtLoad = name
      editedName = name
      setTheme(name, themes)
      flash(t('themeEditor.saved'))
    } catch (err) {
      flash(t('themeEditor.saveFailed', { error: String(err) }))
    }
  }

  async function remove() {
    if (!isCustomSelected || isDraft) return
    const name = selectionAtLoad
    if (!confirm(t('confirm.removeTheme', { name }))) return
    try {
      await deleteTheme(name)
      const nextThemes = { ...(themes ?? {}) }
      delete nextThemes[name]
      themes = nextThemes
      theme = 'dark'
      selectedThemeName = 'dark'
      setTheme('dark', themes)
      loadSelection('dark')
      flash(t('themeEditor.deleted'))
    } catch (err) {
      flash(t('themeEditor.deleteFailed', { error: String(err) }))
    }
  }
</script>

<div class="theme-editor-root">
  <div class="panel theme-editor-panel">
    <header class="panel-title">
      <span class="panel-title-text">{t('themeEditor.currentThemeLabel')}<strong>{activeThemeLabel}</strong></span>
      <CollapseToggle bind:collapsed={themePanelCollapsed} onToggle={persistThemePanelCollapsed} expandTitle="" collapseTitle="" />
    </header>
    {#if !themePanelCollapsed}
      <div class="panel-body theme-editor-panel-body">
        <div class="theme-editor-panel-actions">
          <div class="theme-editor-panel-actions-group">
            <IconButton title={t('themeEditor.addButton')} disabled={isDraft} on:click={addTheme}>
              <Icon name="add" />
            </IconButton>
            <IconButton title={t('themeEditor.copyButton')} on:click={copyTheme}>
              <Icon name="copy" />
            </IconButton>
            <IconButton
              title={t('themeEditor.deleteButton')}
              disabled={!isCustomSelected || isDraft}
              on:click={remove}
            >
              <Icon name="remove" />
            </IconButton>
          </div>
          <div class="theme-editor-panel-actions-group">
            <button class="btn" type="button" disabled={!isCustomSelected || isDraft} on:click={resetToSaved}>{t('themeEditor.resetButton')}</button>
            <button class="btn" type="button" disabled={!isCustomSelected} on:click={() => confirmResetFrom('dark')}>{t('themeEditor.resetToDark')}</button>
            <button class="btn" type="button" disabled={!isCustomSelected} on:click={() => confirmResetFrom('light')}>{t('themeEditor.resetToLight')}</button
            >
          </div>
        </div>
        <div class="theme-editor-panel-row">
          <label class="field theme-editor-panel-select">
            <span>{t('theme.selectTitle')}</span>
            <select bind:value={selectedThemeName} on:change={onThemeSelect}>
              <option value="dark">{t('theme.dark')}</option>
              <option value="light">{t('theme.light')}</option>
              {#each Object.keys(themes ?? {}) as name (name)}
                <option value={name}>{name}</option>
              {/each}
              {#if isDraft}
                <option value={NEW_THEME_ENTRY}>{editedName || t('themeEditor.newThemeDefaultName')}</option>
              {/if}
            </select>
          </label>
          <label class="field theme-editor-panel-name">
            <span>{t('field.name')}</span>
            <input
              type="text"
              bind:value={editedName}
              disabled={!isCustomSelected}
              placeholder={t('themeEditor.themeNamePlaceholder')}
              bind:this={nameInputEl}
            />
          </label>
        </div>

        <div class="messages-group theme-editor-fonts">
          <button class="messages-group-header" type="button" on:click={toggleFonts}>
            <span class="messages-group-title">{t('nav.fonts')}</span>
            <span class="collapse-glyph">{fontsCollapsed ? '▸' : '▾'}</span>
          </button>
          {#if !fontsCollapsed}
            <label class="field">
              <span>{t('field.fontUi')}</span>
              <input type="text" placeholder={t('placeholder.fontDefault')} bind:value={fontUiDraft} />
            </label>
            <label class="field">
              <span>{t('field.fontMono')}</span>
              <input type="text" placeholder={t('placeholder.fontDefaultMono')} bind:value={fontMonoDraft} />
            </label>
            <p class="hint">{t('hint.fonts')}</p>
            <p class="hint">{t('hint.uiScale', { percent: uiScale })}</p>
            <button class="btn" type="button" on:click={() => onSaveFonts(fontUiDraft, fontMonoDraft)}
              >{t('button.applyFonts')}</button
            >
          {/if}
        </div>
      </div>
    {/if}
  </div>

  <div class="theme-editor">
    <div class="theme-editor-fields">
      <input
        type="text"
        class="theme-editor-filter"
        placeholder={t('themeEditor.filterPlaceholder')}
        bind:value={fieldFilter}
      />
      {#each visibleGroups as group (group.label)}
        <div class="messages-group">
          <button class="messages-group-header" type="button" on:click={() => toggleGroup(group.label)}>
            <span class="messages-group-title">{group.label}</span>
            <span class="collapse-glyph">{collapsedGroups.has(group.label) ? '▸' : '▾'}</span>
          </button>
          {#if filterTerms.length > 0 || !collapsedGroups.has(group.label)}
            {#each group.tokens as name (name)}
              <label class="field">
                <span class="token-name">--sm-{name}</span>
                <div class="color-field">
                  <span class="color-swatch-wrap">
                    <span class="color-swatch" style="background: {palette[name]}"></span>
                    <input
                      type="color"
                      class="color-swatch-input"
                      value={isHexValue(palette[name]) ? palette[name] : '#7fd4ff'}
                      disabled={!isCustomSelected}
                      on:input={(e) => (palette[name] = e.currentTarget.value)}
                      title={t('themeEditor.pickColor')}
                    />
                  </span>
                  <input type="text" bind:value={palette[name]} disabled={!isCustomSelected} />
                </div>
              </label>
            {/each}
          {/if}
        </div>
      {/each}
    </div>

    <!-- svelte-ignore a11y-no-static-element-interactions -->
    <!-- svelte-ignore a11y-click-events-have-key-events -->
    <div class="theme-editor-preview-pane" style={previewStyle} on:click={onBackgroundClick}>
      <div class="theme-editor-preview" bind:this={previewPanelEl}>
        <button type="button" class="panel-title" on:click={onPreviewClick}>
          <span class="panel-title-text">{t('themeEditor.previewPanelTitle')}</span>
        </button>
        <!-- svelte-ignore a11y-no-static-element-interactions -->
        <!-- svelte-ignore a11y-click-events-have-key-events -->
        <div class="theme-editor-preview-body" on:click={onPreviewBodyClick}>
          <div class="list">
            <!-- svelte-ignore a11y-no-static-element-interactions -->
            <!-- svelte-ignore a11y-click-events-have-key-events -->
            <div class="row selected" on:click={onPreviewClick}>
              {t('themeEditor.previewSelectedRow')}
              <span class="theme-editor-preview-row-dots">
                <button type="button" class="status-running theme-editor-preview-dot" on:click={onNestedPreviewClick}>●</button>
                <button type="button" class="status-ok theme-editor-preview-dot" on:click={onNestedPreviewClick}>●</button>
                <button type="button" class="status-fail theme-editor-preview-dot" on:click={onNestedPreviewClick}>●</button>
              </span>
            </div>
            <!-- svelte-ignore a11y-no-static-element-interactions -->
            <!-- svelte-ignore a11y-click-events-have-key-events -->
            <div class="row" on:click={onPreviewClick}>
              {t('themeEditor.previewRow')}
              <span class="theme-editor-preview-row-dots">
                <button type="button" class="status-running theme-editor-preview-dot" on:click={onNestedPreviewClick}>●</button>
                <button type="button" class="status-ok theme-editor-preview-dot" on:click={onNestedPreviewClick}>●</button>
                <button type="button" class="status-fail theme-editor-preview-dot" on:click={onNestedPreviewClick}>●</button>
              </span>
            </div>
          </div>
          <div class="theme-editor-preview-chips">
            <button type="button" class="chip active" on:click={onPreviewClick}
              >{t('themeEditor.previewChipActive')}</button
            >
            <button type="button" class="chip" on:click={onPreviewClick}>{t('themeEditor.previewChip')}</button
            >
          </div>
          <div class="theme-editor-preview-buttons">
            <button type="button" class="btn" on:click={onPreviewClick}>{t('themeEditor.previewButton')}</button
            >
            <button type="button" class="btn btn-primary" on:click={onPreviewClick}
              >{t('themeEditor.previewButtonPrimary')}</button
            >
          </div>
          <div class="theme-editor-preview-tabs">
            <button type="button" class="theme-editor-preview-tab" on:click={onPreviewClick}
              >{t('themeEditor.previewTabTitle')}</button
            >
          </div>
          <div class="theme-editor-preview-text-examples">
            <div class="theme-editor-preview-text-row">
              <button type="button" class="theme-editor-preview-heading" on:click={onPreviewClick}
                >{t('themeEditor.previewHeadingText')}</button
              >
            </div>
            <div class="theme-editor-preview-text-row">
              <button type="button" class="theme-editor-preview-normal" on:click={onPreviewClick}
                >{@html t('themeEditor.previewNormalHtml')}</button
              >
            </div>
            <div class="theme-editor-preview-text-row">
              <button type="button" class="theme-editor-preview-highlighted" on:click={onPreviewClick}
                >{t('themeEditor.previewHighlightedText')}</button
              >
            </div>
            <div class="theme-editor-preview-text-row">
              <button
                type="button"
                class="theme-editor-preview-masked theme-editor-preview-hotspot"
                on:click={onPreviewClick}>{t('themeEditor.previewMasked')}</button
              >
            </div>
            <div class="theme-editor-preview-text-row">
              <button
                type="button"
                class="theme-editor-preview-warning theme-editor-preview-hotspot"
                on:click={onPreviewClick}>{t('themeEditor.previewWarning')}</button
              >
            </div>
            <div class="theme-editor-preview-text-row">
              <button
                type="button"
                class="theme-editor-preview-error theme-editor-preview-hotspot"
                on:click={onPreviewClick}>{t('themeEditor.previewError')}</button
              >
            </div>
          </div>
          <!-- svelte-ignore a11y-no-static-element-interactions -->
          <!-- svelte-ignore a11y-click-events-have-key-events -->
          <div class="messages-group-header theme-editor-preview-section" on:click={onPreviewClick}>
            <span class="messages-group-title">{t('themeEditor.previewSectionTitle')}</span>
            <!-- svelte-ignore a11y-no-static-element-interactions -->
            <!-- svelte-ignore a11y-click-events-have-key-events -->
            <span class="output-status" on:click={onNestedPreviewClick}>
              {t('themeEditor.previewStatusRunning')}<button
                type="button"
                class="status-dot status-running theme-editor-preview-dot"
                on:click={onNestedPreviewClick}>●</button
              >
              {t('themeEditor.previewStatusOk')}<button
                type="button"
                class="status-dot status-ok theme-editor-preview-dot"
                on:click={onNestedPreviewClick}>●</button
              >
              {t('themeEditor.previewStatusFail')}<button
                type="button"
                class="status-dot status-fail theme-editor-preview-dot"
                on:click={onNestedPreviewClick}>●</button
              >
            </span>
            <span class="collapse-glyph">▾</span>
          </div>
          <!-- svelte-ignore a11y-no-static-element-interactions -->
          <!-- svelte-ignore a11y-click-events-have-key-events -->
          <div class="theme-editor-preview-cmd theme-editor-preview-hotspot" on:click={onPreviewClick}>
            <button
              type="button"
              class="theme-editor-preview-copy-btn theme-editor-preview-copy-btn-corner"
              title={t('themeEditor.previewCopyButtonLabel')}
              aria-label={t('themeEditor.previewCopyButtonLabel')}
              on:click={onNestedPreviewClick}
            >
              <Icon name="copy" />
            </button>
            <div class="theme-editor-preview-cmd-line">
              <span class="theme-editor-preview-cmd-no">1</span>
              <span>{t('themeEditor.previewCommandLine1')}</span>
            </div>
            <div class="theme-editor-preview-cmd-line">
              <span class="theme-editor-preview-cmd-no">2</span>
              <span>{t('themeEditor.previewCommandLine2')}</span>
            </div>
          </div>
          <!-- svelte-ignore a11y-no-static-element-interactions -->
          <!-- svelte-ignore a11y-click-events-have-key-events -->
          <div class="theme-editor-preview-output-body theme-editor-preview-hotspot" on:click={onPreviewClick}>
            <button
              type="button"
              class="theme-editor-preview-copy-btn theme-editor-preview-copy-btn-corner"
              title={t('themeEditor.previewCopyButtonLabel')}
              aria-label={t('themeEditor.previewCopyButtonLabel')}
              on:click={onNestedPreviewClick}
            >
              <Icon name="copy" />
            </button>
            {t('themeEditor.previewOutputLine')}
          </div>
          <div class="theme-editor-preview-toast-row">
            <button
              type="button"
              class="theme-editor-preview-toast theme-editor-preview-hotspot"
              on:click={onPreviewClick}>{t('themeEditor.previewToast')}</button
            >
            <div class="theme-editor-preview-scrollbar-box">
              <button type="button" class="theme-editor-preview-scrollbar-text" on:click={onScrollbarClick}
                >{t('themeEditor.previewScrollbarLabel')}</button
              >
            </div>
          </div>
        </div>
      </div>
    </div>
  </div>
</div>

<style>
  .theme-editor-root {
    display: flex;
    flex-direction: column;
    gap: 12px;
    height: 100%;
    min-height: 0;
  }

  .theme-editor-panel {
    flex: none;
  }

  .theme-editor-fonts {
    margin-top: 4px;
  }

  .theme-editor-fonts .btn {
    align-self: flex-start;
  }

  .theme-editor-panel-body {
    display: flex;
    flex-direction: column;
    gap: 10px;
  }

  .theme-editor-panel-row {
    display: flex;
    gap: 12px;
    flex-wrap: wrap;
  }

  .theme-editor-panel-select,
  .theme-editor-panel-name {
    flex: 1 1 200px;
    min-width: 0;
    margin-bottom: 0;
  }

  .theme-editor-panel-actions {
    display: flex;
    justify-content: space-between;
    gap: 6px;
    flex-wrap: wrap;
  }

  .theme-editor-panel-actions-group {
    display: flex;
    gap: 6px;
    flex-wrap: wrap;
  }

  .theme-editor-panel-actions .btn:disabled {
    opacity: 0.4;
    cursor: default;
    pointer-events: none;
  }

  .theme-editor {
    display: flex;
    gap: 16px;
    flex: 1 1 auto;
    min-height: 0;
  }

  .theme-editor-fields {
    flex: 1 1 50%;
    min-width: 0;
    overflow-y: auto;
    padding-right: 10px;
  }

  .theme-editor-filter {
    box-sizing: border-box;
    width: 100%;
    margin-bottom: 10px;
    background: var(--sm-bg-deep);
    color: var(--sm-text);
    border: 1px solid var(--sm-border);
    border-radius: 4px;
    padding: 5px 7px;
    font-family: inherit;
    font-size: var(--sm-type-base);
  }

  .token-name {
    font-family: var(--sm-font-mono);
    font-size: var(--sm-type-sm);
  }

  .theme-editor-preview-pane {
    flex: 1 1 50%;
    min-width: 0;
    overflow-y: auto;
    background: var(--sm-bg);
    border-radius: 6px;
    padding: 14px;
    cursor: pointer;
  }

  .theme-editor-preview-pane:hover {
    outline: 1px dashed var(--sm-border);
    outline-offset: -2px;
  }

  .theme-editor-preview {
    border: 1px solid var(--sm-border);
    border-radius: 6px;
    background: var(--sm-bg-alt);
    overflow: hidden;
  }

  .theme-editor-preview-body {
    padding: 12px;
    display: flex;
    flex-direction: column;
    gap: 10px;
  }

  .theme-editor-preview-hotspot {
    display: block;
    width: 100%;
    box-sizing: border-box;
    text-align: left;
    font-family: inherit;
    cursor: pointer;
    background: transparent;
    border: none;
    padding: 0;
    margin: 0;
  }

  .theme-editor-preview-hotspot:hover {
    outline: 1px dashed var(--sm-border);
    outline-offset: 2px;
  }

  .theme-editor-preview-section {
    cursor: pointer;
  }

  .theme-editor-preview-section:hover {
    outline: 1px dashed var(--sm-border);
    outline-offset: 2px;
  }

  .theme-editor-preview-dot {
    background: transparent;
    border: none;
    padding: 0;
    margin: 0;
    cursor: pointer;
    font: inherit;
  }

  .theme-editor-preview-tab {
    background: none;
    border: none;
    border-bottom: 2px solid var(--sm-text-tab);
    padding: 6px 4px 8px;
    color: var(--sm-text-tab);
    font-size: var(--sm-type-base);
    font-weight: 700;
    font-family: inherit;
    cursor: pointer;
  }

  .theme-editor-preview .row {
    display: flex;
    align-items: center;
  }
  .theme-editor-preview-row-dots {
    display: flex;
    gap: 6px;
    margin-left: auto;
    padding-left: 8px;
    font-size: var(--sm-type-xl);
    line-height: 1;
  }

  button.panel-title {
    width: 100%;
    text-align: left;
    font-family: inherit;
    cursor: pointer;
    border-top: none;
    border-left: none;
    border-right: none;
  }

  .list {
    display: flex;
    flex-direction: column;
    gap: 2px;
  }

  .theme-editor-preview-chips,
  .theme-editor-preview-buttons {
    display: flex;
    gap: 6px;
    align-self: flex-start;
  }

  .theme-editor-preview-text-examples {
    display: flex;
    flex-direction: column;
    gap: 6px;
    align-items: flex-start;
    align-self: flex-start;
  }

  .theme-editor-preview-text-row {
    display: flex;
    align-items: baseline;
    gap: 8px;
  }

  .theme-editor-preview-heading,
  .theme-editor-preview-normal,
  .theme-editor-preview-highlighted {
    background: transparent;
    border: none;
    padding: 0;
    font-family: inherit;
    cursor: pointer;
  }

  .theme-editor-preview-heading {
    color: var(--sm-text-heading);
    font-size: var(--sm-type-lg);
    font-weight: 700;
  }

  .theme-editor-preview-normal {
    font-size: var(--sm-type-base);
    color: var(--sm-text);
  }

  .theme-editor-preview-highlighted {
    background: var(--sm-bg-deep);
    color: var(--sm-text-highlight);
    padding: 1px 5px;
    border-radius: 3px;
    font-family: var(--sm-font-mono);
    font-size: var(--sm-type-sm);
  }

  .theme-editor-preview-highlighted:hover {
    background: var(--sm-tint-hover);
    outline: 1px solid var(--sm-text-highlight);
  }

  .theme-editor-preview-cmd {
    position: relative;
    background: var(--sm-bg-deep);
    border-radius: 4px;
    padding: 8px 0;
    font-family: var(--sm-font-mono);
    font-size: var(--sm-type-sm);
    color: var(--sm-text);
  }

  .theme-editor-preview-cmd-line {
    display: flex;
    gap: 10px;
    padding: 0 8px;
  }

  .theme-editor-preview-cmd-no {
    flex: none;
    width: 1.4em;
    text-align: right;
    color: var(--sm-line-number);
    user-select: none;
  }

  .theme-editor-preview-output-body {
    position: relative;
    background: var(--sm-bg-deep);
    border-radius: 4px;
    margin: 0;
    padding: 8px 10px;
    font-family: var(--sm-font-mono);
    font-size: var(--sm-type-sm);
    color: var(--sm-text-muted);
    white-space: pre-wrap;
  }

  .theme-editor-preview-error {
    align-self: flex-start;
    width: auto;
    margin: 0;
    font-size: var(--sm-type-sm);
    color: var(--sm-error);
  }

  .theme-editor-preview-warning {
    align-self: flex-start;
    width: auto;
    margin: 0;
    font-size: var(--sm-type-sm);
    color: var(--sm-warning);
  }

  .theme-editor-preview-masked {
    align-self: flex-start;
    width: auto;
    margin: 0;
    font-size: var(--sm-type-sm);
    font-family: var(--sm-font-mono);
    color: var(--sm-masked);
  }

  .theme-editor-preview-toast {
    align-self: flex-start;
    width: auto;
    background: var(--sm-panel-header);
    color: var(--sm-text);
    border: 1px solid var(--sm-border);
    border-radius: 6px;
    padding: 8px 16px;
    font-size: var(--sm-type-base);
    box-shadow: 0 4px 12px var(--sm-shadow);
  }

  .theme-editor-preview-copy-btn {
    display: flex;
    align-items: center;
    justify-content: center;
    background: transparent;
    border: none;
    padding: 3px 5px;
    border-radius: 4px;
    color: var(--sm-text-muted);
    cursor: pointer;
  }

  .theme-editor-preview-copy-btn:hover {
    background: var(--sm-overlay-soft);
    color: var(--sm-text);
  }

  .theme-editor-preview-copy-btn-corner {
    position: absolute;
    top: 4px;
    right: 4px;
  }

  .theme-editor-preview-toast-row {
    display: flex;
    align-items: flex-start;
    gap: 12px;
    align-self: flex-start;
  }

  .theme-editor-preview-scrollbar-box {
    flex: none;
    width: 180px;
    overflow-x: scroll;
    overflow-y: hidden;
    background: var(--sm-bg-deep);
    border-radius: 4px;
    padding: 10px;
    box-sizing: border-box;
    scrollbar-color: var(--sm-scrollbar) transparent;
  }

  .theme-editor-preview-scrollbar-box::-webkit-scrollbar {
    height: 10px;
  }

  .theme-editor-preview-scrollbar-box::-webkit-scrollbar-track {
    background: transparent;
  }

  .theme-editor-preview-scrollbar-box::-webkit-scrollbar-thumb {
    background-color: var(--sm-scrollbar);
    border-radius: 5px;
  }

  .theme-editor-preview-scrollbar-text {
    display: inline-block;
    min-width: 480px;
    text-align: left;
    background: transparent;
    border: none;
    padding: 0;
    font-family: inherit;
    font-size: var(--sm-type-base);
    color: var(--sm-text-muted);
    cursor: pointer;
    white-space: nowrap;
  }
</style>
