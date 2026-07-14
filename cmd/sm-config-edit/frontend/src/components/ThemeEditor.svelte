<script lang="ts">
  import { onMount, tick } from 'svelte'
  import { TOKEN_GROUPS, readPaletteFor, setTheme, type CustomPalette, type Theme } from '@shared/theme'
  import CollapseToggle from '@shared/components/CollapseToggle.svelte'
  import Icon from '@shared/components/Icon.svelte'
  import IconButton from '@shared/components/IconButton.svelte'
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
  // Shows a transient toast — the parent's own flash(), passed through so
  // save/delete feedback here looks the same as every other action in the
  // app instead of a locally-styled inline message.
  export let flash: (msg: string) => void

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

  // The toolbar's "+" button — same draft-staging path as picking the old
  // "+ New theme" dropdown entry, just triggered directly now that entry
  // is gone (it only reappears, dynamically, once a draft is in progress;
  // see the template). Disabled while already drafting, since calling this
  // again would silently reset the in-progress draft's name/palette.
  function addTheme() {
    selectedThemeName = NEW_THEME_ENTRY
    onThemeSelect()
  }

  // Duplicates whatever's currently loaded — built-in, saved custom, or an
  // in-progress draft — into a new unsaved draft, "<name> - copy", the same
  // one-step-duplicate shape Displays' Copy display button already uses.
  // Unlike addTheme, this doesn't go through onThemeSelect: that would
  // reset editedName/palette to fresh draft defaults instead of preserving
  // what's being copied.
  function copyTheme() {
    const baseName = editedName.trim() || activeThemeLabel
    const copiedPalette = { ...palette }
    selectionAtLoad = ''
    editedName = `${baseName} - copy`
    palette = copiedPalette
    selectedThemeName = NEW_THEME_ENTRY
  }

  // Applies before resetFrom/resetToSaved actually replace the working
  // palette — all three silently discard whatever's currently unsaved, so
  // they share one confirm message parameterized by what they're resetting
  // to.
  function confirmResetFrom(base: 'dark' | 'light') {
    const target = base === 'dark' ? t('theme.dark') : t('theme.light')
    if (confirm(t('confirm.resetTheme', { target }))) resetFrom(base)
  }

  // Reverts the working palette back to this theme's last-saved state,
  // discarding any in-progress edits — only meaningful for an existing
  // saved custom theme (same disabled condition as Delete: a draft has no
  // saved state yet, and Dark/Light's fields aren't editable to begin
  // with).
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
      // ignore corrupt/missing layout, default already set
    }
  })

  function persistThemePanelCollapsed() {
    localStorage.setItem(THEME_PANEL_KEY, JSON.stringify({ collapsed: themePanelCollapsed }))
  }

  let collapsedGroups = new Set<string>()

  function toggleGroup(label: string) {
    const next = new Set(collapsedGroups)
    if (next.has(label)) next.delete(label)
    else next.add(label)
    collapsedGroups = next
  }

  // Every --sm-* token referenced by a CSS rule that directly matches el —
  // in its resting state, or in its :hover state (tested by stripping
  // :hover from the selector and re-matching) — not one it merely
  // inherits from an ancestor's own rule. Walks the live stylesheets
  // instead of a hand-maintained per-element map, so a new preview
  // element's own CSS automatically shows up here without this needing an
  // update. Can't fully resolve cascade/specificity (e.g. a selected row's
  // own :hover rule never actually applies, since .row.selected comes
  // after .row:hover in theme.css at equal specificity) — this unions
  // every matching rule's tokens regardless, so the result is occasionally
  // over-inclusive. Acceptable for narrowing the filter box below; not a
  // precision requirement.
  function tokensForElement(el: Element): string[] {
    const found = new Set<string>()
    const tokenRe = /var\(\s*--sm-([\w-]+)/g
    // Scans the whole declaration block's text rather than enumerating
    // style.item(i)/getPropertyValue() one property at a time — WebKitGTK
    // (this app's Linux target) doesn't reliably expose a shorthand like
    // "background: var(--sm-bg-primary)" through per-property lookups
    // the same way Chromium/WebView2 does; cssText sidesteps that engine
    // difference entirely.
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
          // selector unsupported by this browser — skip
        }
      }
    }
    for (const sheet of Array.from(document.styleSheets)) {
      try {
        walk(sheet.cssRules)
      } catch {
        // inaccessible (cross-origin) stylesheet — skip
      }
    }
    return Array.from(found)
  }

  // Falls back to the panel's own tokens (--sm-bg-alt/--sm-border) whenever
  // the clicked element has none of its own — e.g. the markdown preview's
  // plain body text, which only inherits color from its wrapping button
  // rather than referencing a token directly itself. Without this, such a
  // click would blank the filter box instead of showing something useful.
  let fieldFilter = ''
  function filterForElement(el: Element) {
    const tokens = tokensForElement(el)
    fieldFilter = (tokens.length > 0 ? tokens : previewPanelEl ? tokensForElement(previewPanelEl) : []).join(', ')
  }
  function onPreviewClick(e: MouseEvent) {
    filterForElement(e.currentTarget as Element)
  }
  // The copy buttons now nest inside the cmd/output blocks (matching the
  // real UI, where a copy button floats in a code block's corner) — a
  // <button> can't nest inside another <button>, so the cmd/output blocks
  // switched from <button> to a plain clickable <div> (same pattern as
  // onPreviewBodyClick/onBackgroundClick above) for exactly this reason.
  // Without stopping propagation here, a click on the copy button would
  // also bubble to the wrapping block's own onPreviewClick, which runs
  // after and would overwrite the copy button's own tokens.
  function onCopyBtnClick(e: MouseEvent) {
    e.stopPropagation()
    filterForElement(e.currentTarget as Element)
  }
  // --sm-scrollbar styles a ::-webkit-scrollbar-thumb pseudo-element — not
  // a real DOM node, so tokensForElement (which works by testing
  // el.matches(rule.selectorText)) can never discover it no matter what's
  // clicked. Set the filter directly instead. (There's no separate hover
  // token: ::-webkit-scrollbar-thumb:hover's background-color didn't
  // actually take effect in testing, so the thumb's hover state is left to
  // the browser/OS default instead of a custom one that doesn't apply.)
  function onScrollbarClick() {
    fieldFilter = 'scrollbar'
  }

  // The pane itself carries --sm-bg (the page-level background the panel
  // sits on, kept visible as a frame around it — see
  // .theme-editor-preview-pane) — clicking that frame, rather than
  // anything inside it, filters for it. Every click from an inner element
  // bubbles up here too, so this only acts when the pane itself — not a
  // descendant — was the actual click target.
  function onBackgroundClick(e: MouseEvent) {
    if (e.target === e.currentTarget) filterForElement(e.currentTarget as Element)
  }

  // .theme-editor-preview (the panel surface itself, --sm-bg-alt) has no
  // blank space of its own to click — its only direct child besides the
  // title button is .theme-editor-preview-body, which fills it edge to
  // edge via padding. So blank-space clicks land on the body wrapper, not
  // the panel — this mirrors onBackgroundClick's target-must-be-currentTarget
  // guard (only fires for the body's own blank padding/gaps, not a bubbled
  // click from a row/chip/button inside it) but resolves the token filter
  // against the panel element, since that's the one whose CSS rule actually
  // references --sm-bg-alt.
  let previewPanelEl: HTMLElement | undefined
  function onPreviewBodyClick(e: MouseEvent) {
    if (e.target === e.currentTarget && previewPanelEl) filterForElement(previewPanelEl)
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

  // Exported so App.svelte's global Ctrl+S handler can reach it via
  // bind:this — the panel's own Save button is gone now that Ctrl+S covers
  // it, but the underlying action still needs a callable entry point.
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
            <button type="button" class="row selected" on:click={onPreviewClick}
              >{t('themeEditor.previewSelectedRow')}</button
            >
            <button type="button" class="row" on:click={onPreviewClick}>{t('themeEditor.previewRow')}</button>
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
          <div class="theme-editor-preview-cmd theme-editor-preview-hotspot" on:click={onPreviewClick}>
            <button
              type="button"
              class="theme-editor-preview-copy-btn theme-editor-preview-copy-btn-corner"
              title={t('themeEditor.previewCopyButtonLabel')}
              aria-label={t('themeEditor.previewCopyButtonLabel')}
              on:click={onCopyBtnClick}
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
              on:click={onCopyBtnClick}
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
    font-size: 0.85rem;
  }

  /* .field/.field input/.field select (including the select-arrow
     override and its light-theme variant) come from the shared design
     system (@shared/theme.css) — not redefined here. */

  .token-name {
    font-family: "SF Mono", Consolas, monospace;
    font-size: 0.75rem;
  }

  .color-field {
    display: flex;
    align-items: center;
    gap: 8px;
  }

  /* input[type="color"] only accepts a 6-digit hex — it silently ignores
     anything else (rgba(), named colors, …), so its own native swatch
     would always show the same fallback color for every non-hex token
     (--sm-scrollbar, --sm-overlay-soft, --sm-shadow, …) rather than the
     value actually in effect. .color-swatch is a plain div instead, which
     CSS's background property renders correctly for any valid color
     including alpha — laid under the real <input>, which is kept but made
     fully transparent (not display:none — it must stay a real, clickable
     element for the native picker to still open on click) so picking still
     works for hex-editable tokens without misrepresenting the rest. */
  .color-swatch-wrap {
    position: relative;
    flex: none;
    width: 40px;
    height: 30px;
  }

  .color-swatch {
    position: absolute;
    inset: 0;
    border-radius: 3px;
    border: 1px solid var(--sm-border);
    pointer-events: none;
  }

  .color-swatch-input {
    position: absolute;
    inset: 0;
    box-sizing: border-box;
    width: 100%;
    height: 100%;
    padding: 0;
    border: none;
    opacity: 0;
    cursor: pointer;
  }

  .color-swatch-input:disabled {
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
    border-bottom: 1px solid color-mix(in srgb, var(--sm-text-highlight) 35%, var(--sm-border));
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
    color: var(--sm-text-highlight);
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

  /* Every preview element is a real <button> (or, for .theme-editor-preview-cmd/
     -output-body, a plain clickable <div> — see onCopyBtnClick's comment for
     why) — not just for the ones that already look like one (.row, .chip,
     .btn, all originally designed as buttons, so they need no reset at all)
     but also plain text/block ones (command, output, error, masked, toast)
     that need their native button chrome neutralized. Only the display-
     affecting/font/alignment properties are reset here — each element's own
     class still owns its padding/margin/color/background, so this can't
     clobber them regardless of stylesheet order. box-sizing: border-box
     keeps width: 100% meaning "100% of the parent" even for the elements
     among these (like .theme-editor-preview-output-body) that also carry
     their own horizontal padding — without it, that padding would add to
     the 100% instead of eating into it, making just that one element wider
     than its siblings. */
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

  /* align-self: flex-start shrink-wraps these to their content instead of
     stretching full width (the column flex container's default) — the
     leftover space to their right is then the body's own box, not theirs,
     so it falls through to onPreviewBodyClick's blank-space handler
     (--sm-bg-alt) instead of being wrongly attributed to the chip/button. */
  .theme-editor-preview-chips,
  .theme-editor-preview-buttons {
    display: flex;
    gap: 6px;
    align-self: flex-start;
  }

  /* One row per text style, in a fixed order (Heading, Normal, Highlighted,
     Masked, Warning, Error) so every text-related token has its own
     unambiguous, individually clickable example instead of the old single
     paragraph that mixed several of them together. align-items: flex-start
     (rather than the column default of stretch) keeps each row shrink-wrapped
     to its own content — see the .theme-editor-preview-chips/-buttons comment
     above for why that matters for blank-space clicks. */
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
    font-size: 0.95rem;
    font-weight: 700;
  }

  .theme-editor-preview-normal {
    font-size: 0.85rem;
    color: var(--sm-text);
  }

  /* {@html}-inserted content isn't scoped by Svelte, so styling the
     <strong>/<em> tags inside .theme-editor-preview-normal's message would
     need :global() — not needed here since neither gets its own color
     override, they just inherit the button's own var(--sm-text) above. */

  .theme-editor-preview-highlighted {
    background: var(--sm-bg-deep);
    color: var(--sm-text-highlight);
    padding: 1px 5px;
    border-radius: 3px;
    font-family: "SF Mono", Consolas, monospace;
    font-size: 0.78rem;
  }

  /* Same hover treatment as the real .details-content code.copy-value:hover
     it mirrors — the only difference is this preview element isn't
     copyable itself, it just demonstrates the token pair. */
  .theme-editor-preview-highlighted:hover {
    background: var(--sm-tint-hover);
    outline: 1px solid var(--sm-text-highlight);
  }

  .theme-editor-preview-cmd {
    position: relative;
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

  .theme-editor-preview-output-body {
    position: relative;
    background: var(--sm-bg-deep);
    border-radius: 4px;
    margin: 0;
    padding: 8px 10px;
    font-family: "SF Mono", Consolas, monospace;
    font-size: 0.78rem;
    color: var(--sm-text-muted);
    white-space: pre-wrap;
  }

  .theme-editor-preview-error {
    align-self: flex-start;
    width: auto;
    margin: 0;
    font-size: 0.8rem;
    color: var(--sm-error);
  }

  .theme-editor-preview-warning {
    align-self: flex-start;
    width: auto;
    margin: 0;
    font-size: 0.8rem;
    color: var(--sm-warning);
  }

  .theme-editor-preview-masked {
    align-self: flex-start;
    width: auto;
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

  /* Mirrors script-manager-gui's .cmd-copy-btn(:hover) — the only real
     consumer of --sm-overlay-soft — floated into the corner of the cmd/
     output blocks below, same as .cmd-line-copy-btn/.cmd-output-copy-btn
     do in the real Command pane. */
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

  /* --sm-scrollbar styles a real ::-webkit-scrollbar-thumb here, not a
     lookalike — the inner text is forced wider than the box via min-width,
     so a horizontal scrollbar is always present regardless of the box's own
     width (overflow-y stays hidden since only the horizontal bar is being
     demonstrated, and a horizontal one reads more clearly here than the
     app's usual thin vertical ones). The scrollbar itself still can't be
     clicked (see onScrollbarClick above), so the click target is the label
     text sitting inside the scrolling area. A fixed width (rather than
     shrink-wrapped or stretched like its neighbors) keeps the box's own
     size — and so the visible length of track behind the thumb — stable
     regardless of how wide the toast next to it happens to be. No :hover
     rule on the thumb — ::-webkit-scrollbar-thumb:hover's background-color
     doesn't actually take effect, so the thumb's hover is left to the
     browser/OS default rather than a custom color that wouldn't show. */
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
    font-size: 0.85rem;
    color: var(--sm-text-muted);
    cursor: pointer;
    white-space: nowrap;
  }
</style>
