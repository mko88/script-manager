<script lang="ts">
  import { TOKEN_GROUPS, readPaletteFor, setTheme, type CustomPalette, type Theme } from '@shared/theme'
  import { t } from '../messages'

  // Seeded from the shared custom palette if one's already been saved
  // (see App.svelte's syncTheme call), otherwise from Dark's own static
  // values — either way this is a working copy; nothing here is applied
  // app-wide until Save.
  export let initialPalette: CustomPalette | null = null
  // The actual Wails binding, passed straight through like FieldGrid's
  // validateField prop — this component doesn't import bindings itself.
  export let saveCustomTheme: (palette: Record<string, string>) => Promise<void>
  // Two-way bound so saving updates the toolbar dropdown in the parent.
  export let theme: Theme
  export let hasCustomTheme: boolean

  let palette: CustomPalette = initialPalette ? { ...initialPalette } : readPaletteFor('dark')
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
    saving = true
    saveError = ''
    try {
      await saveCustomTheme(palette)
      setTheme('custom', palette)
      theme = 'custom'
      hasCustomTheme = true
      // Syncs back through the bind: so a later re-mount of this
      // component (e.g. navigating away and back to this section) seeds
      // from what was actually saved, not a stale pre-save copy.
      initialPalette = { ...palette }
      savedFlash = true
      clearTimeout(savedFlashTimer)
      savedFlashTimer = setTimeout(() => (savedFlash = false), 2000)
    } catch (err) {
      saveError = String(err)
    } finally {
      saving = false
    }
  }
</script>

<div class="theme-editor">
  <div class="theme-editor-fields">
    <div class="theme-editor-toolbar">
      <div class="theme-editor-reset">
        <button class="btn" type="button" on:click={() => resetFrom('dark')}>{t('themeEditor.resetToDark')}</button>
        <button class="btn" type="button" on:click={() => resetFrom('light')}>{t('themeEditor.resetToLight')}</button
        >
      </div>
      <button class="btn btn-primary" type="button" disabled={saving} on:click={save}>
        {saving ? t('themeEditor.saving') : t('themeEditor.saveButton')}
      </button>
    </div>
    <input
      type="text"
      class="theme-editor-filter"
      placeholder={t('themeEditor.filterPlaceholder')}
      bind:value={fieldFilter}
    />
    {#if saveError}
      <div class="theme-editor-error">{saveError}</div>
    {/if}
    {#if savedFlash}
      <div class="theme-editor-saved">{t('themeEditor.saved')}</div>
    {/if}
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
                  on:input={(e) => (palette[name] = e.currentTarget.value)}
                  title={t('themeEditor.pickColor')}
                />
                <input type="text" bind:value={palette[name]} />
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

<style>
  .theme-editor {
    display: flex;
    gap: 16px;
    height: 100%;
    min-height: 0;
  }

  .theme-editor-fields {
    flex: 1 1 50%;
    min-width: 0;
    overflow-y: auto;
  }

  .theme-editor-toolbar {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 8px;
    margin-bottom: 10px;
  }

  .theme-editor-reset {
    display: flex;
    gap: 6px;
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
    margin-bottom: 10px;
  }

  .theme-editor-error {
    color: var(--sm-error);
    font-size: 0.8rem;
    font-weight: 700;
    margin-bottom: 10px;
  }

  .field {
    display: flex;
    flex-direction: column;
    gap: 2px;
    font-size: 0.8rem;
    color: var(--sm-text-muted);
    margin-bottom: 10px;
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

  .color-field input[type="text"] {
    flex: 1 1 auto;
    min-width: 0;
    background: var(--sm-bg-deep);
    color: var(--sm-text);
    border: 1px solid var(--sm-border);
    border-radius: 4px;
    padding: 5px 7px;
    font-family: inherit;
    font-size: 0.85rem;
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
