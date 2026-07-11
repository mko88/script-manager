<script lang="ts">
  import { onMount, tick } from 'svelte'
  import { TOKEN_GROUPS, readPaletteFor, setTheme, type CustomPalette, type Theme } from '@shared/theme'
  import { t } from '../messages'

  // Two-way bound: this component both seeds from and writes back to the
  // parent's theme/themes state, so switching/saving/deleting here is
  // immediately reflected in whatever else reads them (e.g. a future
  // toolbar indicator).
  export let theme: Theme
  export let themes: Record<string, CustomPalette> | null
  // The actual Wails bindings, passed straight through like FieldGrid's
  // validateField prop — this component doesn't import bindings itself.
  export let saveTheme: (name: string, renamedFrom: string, palette: Record<string, string>) => Promise<void>
  export let deleteTheme: (name: string) => Promise<void>
  export let setActiveTheme: (active: string) => Promise<void>

  // Sentinel dropdown value for "+ New theme" — picked to never collide
  // with a real (user-chosen) theme name.
  const NEW_THEME_ENTRY = '__new-theme__'

  function isBuiltIn(name: string): name is 'dark' | 'light' {
    return name === 'dark' || name === 'light'
  }

  // Seeds the working copy (name + palette) for a given selection — the
  // dropdown's current value for an existing theme (built-in or custom),
  // never called for the "+ New theme" sentinel (that path stages its own
  // draft instead; see onThemeSelect).
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

  // "Custom", "Custom 2", ... — skips any name already taken so a second,
  // third, etc. "+ New theme" draft never collides with an existing save.
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

  // Anything other than the two built-ins is a named custom theme — either
  // one already saved, or the as-yet-unsaved "+ New theme" draft.
  $: isCustomSelected = selectedThemeName !== 'dark' && selectedThemeName !== 'light'
  $: isDraft = selectedThemeName === NEW_THEME_ENTRY
  $: canSave = isCustomSelected && editedName.trim().length > 0

  $: activeThemeLabel = theme === 'dark' ? t('theme.dark') : theme === 'light' ? t('theme.light') : theme

  function onThemeSelect() {
    saveError = ''
    if (selectedThemeName === NEW_THEME_ENTRY) {
      // Stages a local draft only — nothing is activated until Save, per
      // the confirmed design (unlike picking an existing theme below).
      selectionAtLoad = ''
      editedName = nextDefaultName()
      palette = readPaletteFor('dark')
      tick().then(() => nameInputEl?.focus())
      return
    }
    loadSelection(selectedThemeName)
    // Picking an existing theme (built-in or custom) applies it
    // immediately, everywhere — matches the toolbar dropdown's old
    // behavior, just relocated here.
    theme = selectedThemeName
    setTheme(selectedThemeName, themes ?? undefined)
    setActiveTheme(selectedThemeName).catch(() => {})
  }

  const THEME_PANEL_KEY = 'sm-config-edit:themePanel'
  let themePanelCollapsed = false

  onMount(() => {
    try {
      const saved = JSON.parse(localStorage.getItem(THEME_PANEL_KEY) ?? '{}')
      themePanelCollapsed = !!saved.collapsed
    } catch {
      // ignore corrupt/missing layout, default already set
    }
  })

  function toggleThemePanel() {
    themePanelCollapsed = !themePanelCollapsed
    localStorage.setItem(THEME_PANEL_KEY, JSON.stringify({ collapsed: themePanelCollapsed }))
  }

  let collapsedGroups = new Set<string>()
  let saving = false
  let saveError = ''
  let savedFlash = false
  let savedFlashTimer: ReturnType<typeof setTimeout>

  function toggleGroup(label: string) {
    const next = new Set(collapsedGroups)
    if (next.has(label)) next.delete(label)
    else next.add(label)
    collapsedGroups = next
  }

  // Which tokens visibly distinguish each preview element from a plain,
  // unstyled one — clicking that element fills the field-filter box below
  // with exactly these, so "what do I edit to change this?" is a click
  // away instead of a hunt through 22 fields. Only the token(s) actually
  // set by that element's own CSS rule (resting *and* :hover — a plain row/
  // chip/button's hover background is a real, independently editable
  // token, not a lighter/darker computed shade of its resting color), not
  // ones it merely inherits from an ancestor. selectedRow/chipActive don't
  // list a hover token: .row.selected/.chip.active both come after their
  // own :hover rule in theme.css at equal specificity, so hovering a
  // selected row or active chip keeps its active styling — the plain
  // .row:hover/.chip:hover rule never actually applies to them.
  const PREVIEW_TOKEN_MAP: Record<string, string[]> = {
    background: ['bg'],
    panelTitle: ['accent', 'panel-header', 'border'],
    selectedRow: ['accent-warm', 'bg'],
    row: ['text', 'hover'],
    chipActive: ['accent-warm', 'bg'],
    chip: ['bg-deep', 'text-muted', 'border', 'tint-hover', 'text'],
    button: ['border', 'hover', 'hover-strong', 'text'],
    buttonPrimary: ['accent-warm', 'bg', 'btn-primary-hover'],
    heading: ['accent'],
    bodyText: ['text'],
    highlight: ['bg-deep', 'code'],
    command: ['bg-deep', 'text', 'line-number'],
    outputStatus: ['accent'],
    outputBody: ['bg-deep', 'text-muted'],
    errorText: ['error'],
    maskedText: ['masked'],
    toast: ['panel-header', 'text', 'border', 'shadow'],
  }

  let fieldFilter = ''
  function filterFor(key: keyof typeof PREVIEW_TOKEN_MAP) {
    fieldFilter = PREVIEW_TOKEN_MAP[key].join(', ')
  }
  // The rendered example markdown comes from one {@html} block, so its
  // three clickable pieces (heading/body text/highlight) are told apart by
  // delegating from a single listener rather than per-element on:click —
  // Svelte can't attach handlers inside raw HTML it didn't render itself.
  function onMarkdownClick(e: MouseEvent) {
    const tag = (e.target as HTMLElement).tagName
    if (tag === 'CODE') filterFor('highlight')
    else if (tag === 'H1' || tag === 'H2' || tag === 'H3') filterFor('heading')
    else filterFor('bodyText')
  }

  // The pane itself carries --sm-bg (the page-level background the panel
  // sits on, kept visible as a frame around it — see
  // .theme-editor-preview-pane) — clicking that frame, rather than
  // anything inside it, filters for it. Every click from an inner element
  // bubbles up here too, so this only acts when the pane itself — not a
  // descendant — was the actual click target.
  function onBackgroundClick(e: MouseEvent) {
    if (e.target === e.currentTarget) filterFor('background')
  }

  $: filterTerms = fieldFilter
    .split(',')
    .map((s) => s.trim().toLowerCase())
    .filter(Boolean)
  // Exact match, not substring — "bg" from a clicked preview element should
  // show only --sm-bg, not also sweep in --sm-bg-alt/--sm-bg-deep just
  // because they share that prefix.
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

  async function save() {
    if (!canSave) return
    const name = editedName.trim()
    saving = true
    saveError = ''
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
      savedFlash = true
      clearTimeout(savedFlashTimer)
      savedFlashTimer = setTimeout(() => (savedFlash = false), 2000)
    } catch (err) {
      saveError = String(err)
    } finally {
      saving = false
    }
  }

  async function remove() {
    if (!isCustomSelected || isDraft) return
    const name = selectionAtLoad
    try {
      await deleteTheme(name)
      const nextThemes = { ...(themes ?? {}) }
      delete nextThemes[name]
      themes = nextThemes
      theme = 'dark'
      selectedThemeName = 'dark'
      setTheme('dark', themes)
      loadSelection('dark')
    } catch (err) {
      saveError = String(err)
    }
  }
</script>

<div class="theme-editor-root">
  <div class="panel theme-editor-panel">
    <header class="panel-title">
      <span class="panel-title-text">{t('themeEditor.currentThemeLabel')}<strong>{activeThemeLabel}</strong></span>
      <button class="collapse-btn" type="button" on:click={toggleThemePanel}>
        {themePanelCollapsed ? '▸' : '▾'}
      </button>
    </header>
    {#if !themePanelCollapsed}
      <div class="panel-body theme-editor-panel-body">
        <div class="theme-editor-panel-row">
          <label class="field theme-editor-panel-select">
            <span>{t('theme.selectTitle')}</span>
            <select bind:value={selectedThemeName} on:change={onThemeSelect}>
              <option value="dark">{t('theme.dark')}</option>
              <option value="light">{t('theme.light')}</option>
              {#each Object.keys(themes ?? {}) as name (name)}
                <option value={name}>{name}</option>
              {/each}
              <option value={NEW_THEME_ENTRY}>{t('themeEditor.newThemeOption')}</option>
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
        <div class="theme-editor-panel-actions">
          <button class="btn" type="button" disabled={!isCustomSelected} on:click={() => resetFrom('dark')}>{t('themeEditor.resetToDark')}</button>
          <button class="btn" type="button" disabled={!isCustomSelected} on:click={() => resetFrom('light')}>{t('themeEditor.resetToLight')}</button
          >
          <button class="btn" type="button" disabled={!isCustomSelected || isDraft} on:click={remove}>{t('themeEditor.deleteButton')}</button>
          <button class="btn btn-primary" type="button" disabled={!canSave || saving} on:click={save}>
            {saving ? t('themeEditor.saving') : t('themeEditor.saveButton')}
          </button>
        </div>
        {#if saveError}
          <div class="theme-editor-error">{saveError}</div>
        {/if}
        {#if savedFlash}
          <div class="theme-editor-saved">{t('themeEditor.saved')}</div>
        {/if}
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
                  <input
                    type="color"
                    value={isHexValue(palette[name]) ? palette[name] : '#7fd4ff'}
                    disabled={!isCustomSelected}
                    on:input={(e) => (palette[name] = e.currentTarget.value)}
                    title={t('themeEditor.pickColor')}
                  />
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
      <div class="theme-editor-preview">
        <button type="button" class="panel-title" on:click={() => filterFor('panelTitle')}>
          <span class="panel-title-text">{t('themeEditor.previewPanelTitle')}</span>
        </button>
        <div class="theme-editor-preview-body">
          <div class="list">
            <button type="button" class="row selected" on:click={() => filterFor('selectedRow')}
              >{t('themeEditor.previewSelectedRow')}</button
            >
            <button type="button" class="row" on:click={() => filterFor('row')}>{t('themeEditor.previewRow')}</button>
          </div>
          <div class="theme-editor-preview-chips">
            <button type="button" class="chip active" on:click={() => filterFor('chipActive')}
              >{t('themeEditor.previewChipActive')}</button
            >
            <button type="button" class="chip" on:click={() => filterFor('chip')}>{t('themeEditor.previewChip')}</button
            >
          </div>
          <div class="theme-editor-preview-buttons">
            <button type="button" class="btn" on:click={() => filterFor('button')}>{t('themeEditor.previewButton')}</button
            >
            <button type="button" class="btn btn-primary" on:click={() => filterFor('buttonPrimary')}
              >{t('themeEditor.previewButtonPrimary')}</button
            >
          </div>
          <button
            type="button"
            class="theme-editor-preview-markdown theme-editor-preview-hotspot"
            on:click={onMarkdownClick}>{@html t('themeEditor.previewMarkdownHtml')}</button
          >
          <button
            type="button"
            class="theme-editor-preview-cmd theme-editor-preview-hotspot"
            on:click={() => filterFor('command')}
          >
            <div class="theme-editor-preview-cmd-line">
              <span class="theme-editor-preview-cmd-no">1</span>
              <span>{t('themeEditor.previewCommandLine1')}</span>
            </div>
            <div class="theme-editor-preview-cmd-line">
              <span class="theme-editor-preview-cmd-no">2</span>
              <span>{t('themeEditor.previewCommandLine2')}</span>
            </div>
          </button>
          <div class="theme-editor-preview-output">
            <button
              type="button"
              class="theme-editor-preview-output-status theme-editor-preview-hotspot"
              on:click={() => filterFor('outputStatus')}>{t('themeEditor.previewOutputStatus')}</button
            >
            <button
              type="button"
              class="theme-editor-preview-output-body theme-editor-preview-hotspot"
              on:click={() => filterFor('outputBody')}>{t('themeEditor.previewOutputLine')}</button
            >
          </div>
          <button
            type="button"
            class="theme-editor-preview-error theme-editor-preview-hotspot"
            on:click={() => filterFor('errorText')}>{t('themeEditor.previewError')}</button
          >
          <button
            type="button"
            class="theme-editor-preview-masked theme-editor-preview-hotspot"
            on:click={() => filterFor('maskedText')}>{t('themeEditor.previewMasked')}</button
          >
          <button
            type="button"
            class="theme-editor-preview-toast theme-editor-preview-hotspot"
            on:click={() => filterFor('toast')}>{t('themeEditor.previewToast')}</button
          >
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
    font-size: 0.85rem;
  }

  .theme-editor-saved {
    color: var(--sm-accent);
    font-size: 0.8rem;
  }

  .theme-editor-error {
    color: var(--sm-error);
    font-size: 0.8rem;
    font-weight: 700;
  }

  .field {
    display: flex;
    flex-direction: column;
    gap: 2px;
    font-size: 0.8rem;
    color: var(--sm-text-muted);
    margin-bottom: 10px;
  }

  .field input,
  .field select {
    background: var(--sm-bg-deep);
    color: var(--sm-text);
    border: 1px solid var(--sm-border);
    border-radius: 4px;
    padding: 5px 7px;
    font-family: inherit;
    font-size: 0.85rem;
  }

  /* Without appearance: none, <select> keeps its native dropdown-arrow
     chrome in Chromium/WebView2 regardless of the background/color set
     above, showing as a jarring light box behind the arrow against this
     dark theme — same fix as App.svelte's own .field select, duplicated
     here since Svelte's per-component scoping means that rule doesn't
     reach this file's markup. */
  .field select {
    appearance: none;
    -webkit-appearance: none;
    -moz-appearance: none;
    background-image: url("data:image/svg+xml,%3Csvg xmlns='http://www.w3.org/2000/svg' width='10' height='6' viewBox='0 0 10 6'%3E%3Cpath d='M1 1l4 4 4-4' fill='none' stroke='%23a9b6c8' stroke-width='1.4' stroke-linecap='round' stroke-linejoin='round'/%3E%3C/svg%3E");
    background-repeat: no-repeat;
    background-position: right 10px center;
    padding-right: 28px;
  }

  :global([data-theme="light"]) .field select {
    background-image: url("data:image/svg+xml,%3Csvg xmlns='http://www.w3.org/2000/svg' width='10' height='6' viewBox='0 0 10 6'%3E%3Cpath d='M1 1l4 4 4-4' fill='none' stroke='%2355647a' stroke-width='1.4' stroke-linecap='round' stroke-linejoin='round'/%3E%3C/svg%3E");
  }

  .field input:disabled,
  .field select:disabled {
    opacity: 0.6;
    cursor: default;
  }

  .token-name {
    font-family: "SF Mono", Consolas, monospace;
    font-size: 0.75rem;
  }

  .color-field {
    display: flex;
    align-items: center;
    gap: 8px;
  }

  .color-field input[type="color"] {
    flex: none;
    width: 40px;
    height: 30px;
    padding: 2px;
    cursor: pointer;
  }

  .color-field input[type="color"]:disabled {
    cursor: default;
  }

  .color-field input[type="text"] {
    flex: 1 1 auto;
    min-width: 0;
  }

  .messages-group {
    margin-bottom: 14px;
  }

  .messages-group-header {
    display: flex;
    align-items: center;
    justify-content: space-between;
    width: 100%;
    background: none;
    border: none;
    border-bottom: 1px solid color-mix(in srgb, var(--sm-code) 35%, var(--sm-border));
    padding: 0 0 6px;
    margin: 0 0 6px;
    cursor: pointer;
    font-family: inherit;
  }

  .messages-group-title {
    font-size: 0.75rem;
    font-weight: 700;
    text-transform: uppercase;
    letter-spacing: 0.04em;
    color: var(--sm-code);
  }

  .collapse-glyph {
    color: var(--sm-text-muted);
  }

  /* The preview pane is a self-contained little "app" — every --sm-* token
     is overridden inline (see previewStyle) from the working palette, not
     yet saved, so it reflects edits live without touching the real theme
     anywhere else in this window. Reuses the app's actual shared classes
     (.panel-title, .row, .chip, .btn) so it can't visually drift from the
     real components; only the toast and code styling are re-declared here
     since .toast's position:fixed and code's App.svelte-local scoping
     don't make sense reused inside a preview box. */
  /* --sm-bg is the page-level background the whole app sits on — keeping
     this frame around .theme-editor-preview (--sm-bg-alt, the panel
     surface) is the only place in the preview that's visible, exactly like
     the gap around real panels in either app's own window. */
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

  /* Every preview element is a real <button> — not just for the ones that
     already look like one (.row, .chip, .btn, all originally designed as
     buttons, so they need no reset at all) but also plain text/block ones
     (markdown, command, output, error, masked, toast) that need their
     native button chrome neutralized. Only the display-affecting/font/
     alignment properties are reset here — each element's own class still
     owns its padding/margin/color/background, so this can't clobber them
     regardless of stylesheet order. */
  .theme-editor-preview-hotspot {
    display: block;
    width: 100%;
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

  /* .panel-title is already display:flex (global, theme.css) and already
     fills its flex parent's width — only the native <button> chrome needs
     resetting here, and specifically not display, or it'd break the
     panel-title-text/right-side layout .panel-title relies on. */
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
  }

  /* {@html}-inserted content isn't scoped by Svelte, so styling anything
     inside .theme-editor-preview-markdown needs :global() — same reasoning
     as .details-preview's own goldmark-rendered output in App.svelte. */
  .theme-editor-preview-markdown {
    font-size: 0.85rem;
    color: var(--sm-text);
  }

  .theme-editor-preview-markdown :global(h1),
  .theme-editor-preview-markdown :global(h2),
  .theme-editor-preview-markdown :global(h3) {
    color: var(--sm-accent);
    margin: 0 0 0.3em;
  }

  .theme-editor-preview-markdown :global(p) {
    margin: 0;
  }

  .theme-editor-preview-markdown :global(code) {
    background: var(--sm-bg-deep);
    color: var(--sm-code);
    padding: 1px 5px;
    border-radius: 3px;
    font-family: "SF Mono", Consolas, monospace;
  }

  .theme-editor-preview-cmd {
    background: var(--sm-bg-deep);
    border-radius: 4px;
    padding: 8px 0;
    font-family: "SF Mono", Consolas, monospace;
    font-size: 0.8rem;
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

  .theme-editor-preview-output {
    background: var(--sm-bg-deep);
    border-radius: 4px;
    overflow: hidden;
  }

  .theme-editor-preview-output-status {
    padding: 6px 10px;
    font-size: 0.8rem;
    color: var(--sm-accent);
  }

  .theme-editor-preview-output-body {
    margin: 0;
    padding: 0 10px 8px;
    font-family: "SF Mono", Consolas, monospace;
    font-size: 0.78rem;
    color: var(--sm-text-muted);
    white-space: pre-wrap;
  }

  .theme-editor-preview-error {
    margin: 0;
    font-size: 0.8rem;
    color: var(--sm-error);
  }

  .theme-editor-preview-masked {
    margin: 0;
    font-size: 0.8rem;
    font-family: "SF Mono", Consolas, monospace;
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
    font-size: 0.85rem;
    box-shadow: 0 4px 12px var(--sm-shadow);
  }
</style>
