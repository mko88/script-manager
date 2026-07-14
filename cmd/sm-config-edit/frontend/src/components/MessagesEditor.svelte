<script lang="ts">
  import { flash } from '@shared/toast'
  import Icon from '@shared/components/Icon.svelte'
  import IconButton from '@shared/components/IconButton.svelte'
  import { t } from '../messages'

  // The Messages section: edits either app's runtime message-override file
  // (script-manager-gui.messages.json / sm-config-edit.messages.json),
  // flattened into dotted-key rows for a simple key+text-input editor — the
  // same dotted paths t() itself resolves. Independent of the config's own
  // dirty/save flow: a different file, a different Save action.

  // The actual Wails bindings, passed straight through like FieldGrid's
  // validateField prop — this component doesn't import bindings itself.
  export let getEditableMessages: (target: string) => Promise<unknown>
  export let getDefaultMessages: (target: string) => Promise<unknown>
  export let saveMessages: (target: string, messages: Record<string, unknown>) => Promise<void>

  type MessagesTarget = 'gui' | 'configedit'
  let messagesTarget: MessagesTarget = 'gui'
  let messagesRows: { key: string; value: string }[] = []
  let messagesError = ''
  let messagesSearch = ''
  // Which category groups are collapsed, by name — not reset on target
  // switch, so a layout you've arranged (e.g. collapsing categories you
  // don't care about) carries over between script-manager-gui/sm-config-edit.
  let collapsedMessageGroups = new Set<string>()

  function toggleMessageGroup(category: string) {
    const next = new Set(collapsedMessageGroups)
    if (next.has(category)) next.delete(category)
    else next.add(category)
    collapsedMessageGroups = next
  }

  $: allMessageGroupsCollapsed = messagesGroups.length > 0 && collapsedMessageGroups.size >= messagesGroups.length
  function toggleAllMessageGroups() {
    collapsedMessageGroups = allMessageGroupsCollapsed ? new Set() : new Set(messagesGroups.map((g) => g.category))
  }

  function flattenMessages(obj: unknown, prefix = ''): { key: string; value: string }[] {
    if (typeof obj !== 'object' || obj === null) return []
    const rows: { key: string; value: string }[] = []
    for (const [k, v] of Object.entries(obj as Record<string, unknown>)) {
      const key = prefix ? `${prefix}.${k}` : k
      if (typeof v === 'string') rows.push({ key, value: v })
      else rows.push(...flattenMessages(v, key))
    }
    return rows
  }

  function unflattenMessages(rows: { key: string; value: string }[]): Record<string, unknown> {
    const root: Record<string, unknown> = {}
    for (const { key, value } of rows) {
      const parts = key.split('.')
      let node = root
      for (let i = 0; i < parts.length - 1; i++) {
        const part = parts[i]
        if (typeof node[part] !== 'object' || node[part] === null) node[part] = {}
        node = node[part] as Record<string, unknown>
      }
      node[parts[parts.length - 1]] = value
    }
    return root
  }

  $: messagesGroups = (() => {
    const q = messagesSearch.trim().toLowerCase()
    const groups = new Map<string, { key: string; value: string }[]>()
    for (const row of messagesRows) {
      if (q && !row.key.toLowerCase().includes(q) && !row.value.toLowerCase().includes(q)) continue
      const category = row.key.split('.')[0]
      if (!groups.has(category)) groups.set(category, [])
      groups.get(category)!.push(row)
    }
    return Array.from(groups, ([category, rows]) => ({ category, rows }))
  })()

  // Runs once on mount (reactive statements fire initially) and again on
  // every tab switch.
  $: loadMessages(messagesTarget)

  async function loadMessages(target: MessagesTarget) {
    messagesError = ''
    try {
      messagesRows = flattenMessages(await getEditableMessages(target))
    } catch (err) {
      messagesRows = []
      messagesError = String(err)
    }
  }

  async function saveMessagesSection() {
    try {
      await saveMessages(messagesTarget, unflattenMessages(messagesRows))
      const app = messagesTarget === 'gui' ? t('messagesEditor.targetGui') : t('messagesEditor.targetConfigEdit')
      flash(t('messagesEditor.saved', { app }))
    } catch (err) {
      flash(t('messagesEditor.saveFailed', { error: String(err) }))
    }
  }

  // Resets the in-memory form to the target's compiled defaults — Save is
  // still required afterward to persist it, same as any other edit here.
  async function restoreDefaults() {
    if (!confirm(t('messagesEditor.confirmRestoreDefaults'))) return
    try {
      messagesRows = flattenMessages(await getDefaultMessages(messagesTarget))
    } catch (err) {
      flash(t('messagesEditor.restoreDefaultsFailed', { error: String(err) }))
    }
  }
</script>

<div class="messages-toolbar">
  <div class="messages-tabs">
    <button
      class="messages-tab"
      class:active={messagesTarget === 'gui'}
      type="button"
      on:click={() => (messagesTarget = 'gui')}>{t('messagesEditor.targetGui')}</button
    >
    <button
      class="messages-tab"
      class:active={messagesTarget === 'configedit'}
      type="button"
      on:click={() => (messagesTarget = 'configedit')}>{t('messagesEditor.targetConfigEdit')}</button
    >
  </div>
  <div class="messages-actions">
    <IconButton
      title={allMessageGroupsCollapsed ? t('messagesEditor.expandAll') : t('messagesEditor.collapseAll')}
      on:click={toggleAllMessageGroups}
      ><Icon name={allMessageGroupsCollapsed ? 'expand-all' : 'collapse-all'} /></IconButton
    >
    <IconButton title={t('messagesEditor.restoreDefaults')} on:click={restoreDefaults}><Icon name="restore" /></IconButton>
    <IconButton title={t('messagesEditor.saveButton')} on:click={saveMessagesSection}><Icon name="save" /></IconButton>
  </div>
</div>
<input
  type="text"
  class="messages-search"
  placeholder={t('messagesEditor.searchPlaceholder')}
  bind:value={messagesSearch}
/>
{#if messagesError}
  <div class="validation-issue validation-error">{messagesError}</div>
{:else}
  <div class="messages-rows">
    {#each messagesGroups as group (group.category)}
      <div class="messages-group">
        <button
          class="messages-group-header"
          type="button"
          on:click={() => toggleMessageGroup(group.category)}
        >
          <span class="messages-group-title">{group.category}</span>
          <span class="collapse-glyph">{collapsedMessageGroups.has(group.category) ? '▸' : '▾'}</span>
        </button>
        {#if messagesSearch.trim() || !collapsedMessageGroups.has(group.category)}
          {#each group.rows as row (row.key)}
            <label class="field messages-row">
              <span class="messages-row-key">{row.key}</span>
              <input type="text" bind:value={row.value} />
            </label>
          {/each}
        {/if}
      </div>
    {/each}
  </div>
{/if}

<style>
  .messages-toolbar {
    flex: none;
    display: flex;
    align-items: flex-end;
    justify-content: space-between;
    gap: 8px;
    margin-bottom: 10px;
    border-bottom: 1px solid var(--sm-border);
  }

  .messages-tabs {
    display: flex;
    gap: 4px;
  }

  .messages-tab {
    background: none;
    border: none;
    border-bottom: 2px solid transparent;
    padding: 6px 4px 8px;
    margin-bottom: -1px;
    color: var(--sm-text-muted);
    font-size: 0.85rem;
    font-family: inherit;
    cursor: pointer;
  }

  .messages-tab:hover {
    color: var(--sm-text);
  }

  .messages-tab.active {
    color: var(--sm-bg-primary);
    border-bottom-color: var(--sm-bg-primary);
    font-weight: 700;
  }

  .messages-actions {
    flex: none;
    display: flex;
    gap: 4px;
    margin-bottom: 6px;
  }

  .messages-search {
    flex: none;
    margin-bottom: 10px;
    background: var(--sm-bg-deep);
    color: var(--sm-text);
    border: 1px solid var(--sm-border);
    border-radius: 4px;
    padding: 5px 7px;
    font-family: inherit;
    font-size: 0.85rem;
  }

  .messages-rows {
    flex: 1 1 auto;
    overflow-y: auto;
  }

  .messages-row-key {
    font-family: "SF Mono", Consolas, monospace;
    font-size: 0.75rem;
  }
</style>
