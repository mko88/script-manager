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
    {#if saveError}
      <div class="theme-editor-error">{saveError}</div>
    {/if}
    {#if savedFlash}
      <div class="theme-editor-saved">{t('themeEditor.saved')}</div>
    {/if}
    {#each TOKEN_GROUPS as group (group.label)}
      <div class="messages-group">
        <button class="messages-group-header" type="button" on:click={() => toggleGroup(group.label)}>
          <span class="messages-group-title">{group.label}</span>
          <span class="collapse-glyph">{collapsedGroups.has(group.label) ? '▸' : '▾'}</span>
        </button>
        {#if !collapsedGroups.has(group.label)}
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

  <div class="theme-editor-preview-pane">
    <div class="theme-editor-preview" style={previewStyle}>
      <header class="panel-title">
        <span class="panel-title-text">{t('themeEditor.previewPanelTitle')}</span>
      </header>
      <div class="theme-editor-preview-body">
        <div class="list">
          <div class="row selected">{t('themeEditor.previewSelectedRow')}</div>
          <div class="row">{t('themeEditor.previewRow')}</div>
        </div>
        <div class="theme-editor-preview-chips">
          <span class="chip active">{t('themeEditor.previewChipActive')}</span>
          <span class="chip">{t('themeEditor.previewChip')}</span>
        </div>
        <div class="theme-editor-preview-buttons">
          <span class="btn">{t('themeEditor.previewButton')}</span>
          <span class="btn btn-primary">{t('themeEditor.previewButtonPrimary')}</span>
        </div>
        <p class="theme-editor-preview-text">
          {t('themeEditor.previewBodyText')} <code>{t('themeEditor.previewCode')}</code>
        </p>
        <p class="theme-editor-preview-error">{t('themeEditor.previewError')}</p>
        <p class="theme-editor-preview-masked">{t('themeEditor.previewMasked')}</p>
        <div class="theme-editor-preview-toast">{t('themeEditor.previewToast')}</div>
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
  .theme-editor-preview-pane {
    flex: 1 1 50%;
    min-width: 0;
    overflow-y: auto;
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

  .theme-editor-preview-text {
    margin: 0;
    font-size: 0.85rem;
    color: var(--sm-text);
  }

  .theme-editor-preview-text code {
    background: var(--sm-bg-deep);
    color: var(--sm-code);
    padding: 1px 5px;
    border-radius: 3px;
    font-family: "SF Mono", Consolas, monospace;
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
    background: var(--sm-panel-header);
    color: var(--sm-text);
    border: 1px solid var(--sm-border);
    border-radius: 6px;
    padding: 8px 16px;
    font-size: 0.85rem;
    box-shadow: 0 4px 12px var(--sm-shadow);
  }
</style>
