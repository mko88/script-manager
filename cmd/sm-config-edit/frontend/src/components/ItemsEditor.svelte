<script lang="ts">
  import ActionForm from './ActionForm.svelte'
  import FieldGrid from './FieldGrid.svelte'
  import ListToolbar from './ListToolbar.svelte'
  import CheckboxChipList from './CheckboxChipList.svelte'
  import { t } from '../messages'
  import { wrap, sortableList, syncList, type DndEntry } from '../lib/sortable'
  import type { configedit } from '../../wailsjs/go/models'

  // The Items section: a reorderable master list, a detail form (reserved
  // keys get dedicated widgets, everything else is a FieldGrid), and a live
  // preview of the selected item against any display plus any action's
  // expanded command.

  // Two-way bound slices of the parent's cfg.
  export let items: configedit.ItemDTO[]
  export let selectedItem: number
  // Read-only context: the global actions/groups/displays/env the detail
  // form and previews reference.
  export let actions: configedit.ActionDTO[] = []
  export let allActionGroups: string[] = []
  export let displays: configedit.DisplayDTO[] = []
  export let envFields: configedit.FieldDTO[] = []
  // The actual Wails bindings, passed straight through like FieldGrid's
  // validateField prop — this component doesn't import bindings itself.
  export let previewItem: (
    item: configedit.ItemDTO,
    envFields: configedit.FieldDTO[],
    displays: configedit.DisplayDTO[],
    displayName: string,
  ) => Promise<configedit.PreviewDTO>
  export let previewAction: (
    item: configedit.ItemDTO,
    envFields: configedit.FieldDTO[],
    action: configedit.ActionDTO,
  ) => Promise<configedit.ActionPreviewDTO>
  export let validateField: (kind: string, value: string) => Promise<string>
  export let browseScriptFile: () => Promise<string>
  export let previewScriptFile: (path: string) => Promise<configedit.ScriptPreviewDTO>

  $: allActionIds = actions.map((a) => a.id).filter((id) => id)

  // The generated DTO classes for nested-object fields carry a
  // convertValues method, so a plain object literal isn't structurally
  // assignable — cast new entries the same way the initial state does.
  function newItem(): configedit.ItemDTO {
    return { name: '', display: '', actions: [], actionGroups: [], customActions: [], fields: [] } as unknown as configedit.ItemDTO
  }
  function newAction(): configedit.ActionDTO {
    return {
      id: '',
      title: '',
      description: '',
      cmd: '',
      script: '',
      groups: [],
      noWait: false,
      interactive: true,
    } as unknown as configedit.ActionDTO
  }

  function addItem() {
    items = [...items, newItem()]
    selectedItem = items.length - 1
    previewActionIdx = -1
  }
  function removeItem(i: number) {
    items = items.filter((_, idx) => idx !== i)
    if (selectedItem === i) selectedItem = -1
    else if (selectedItem > i) selectedItem -= 1
  }
  function confirmRemoveItem(i: number) {
    const name = items[i]?.name || t('fallback.unnamed')
    if (confirm(t('confirm.removeItem', { name }))) removeItem(i)
  }

  function addCustomAction(itemIdx: number) {
    items[itemIdx].customActions = [...items[itemIdx].customActions, newAction()]
  }
  function removeCustomAction(itemIdx: number, i: number) {
    items[itemIdx].customActions = items[itemIdx].customActions.filter((_, idx) => idx !== i)
  }
  function confirmRemoveCustomAction(itemIdx: number, i: number) {
    const action = items[itemIdx].customActions[i]
    const name = action?.title || action?.id || t('fallback.untitled')
    if (confirm(t('confirm.removeCustomAction', { name }))) removeCustomAction(itemIdx, i)
  }

  // Reordering is opt-in: off by default, toggled per-visit via the
  // reorder-mode button in the toolbar (not persisted — reopening the
  // section, or the app, starts back in "reordering off"). Without this
  // gate, a plain click-to-select on a row is only one accidental pixel of
  // movement away from silently reordering the list. Turning it on clears
  // the selection — keeping a selection alive through a reorder means
  // tracking its index through every live-shifting consider event, which
  // isn't worth the complication.
  let reorderMode = false
  function toggleReorderMode() {
    reorderMode = !reorderMode
    if (reorderMode) selectedItem = -1
  }

  // Re-derived from items on any change EXCEPT while a drag is active —
  // during the drag, dndzone owns itemEntries via consider (below), and
  // reactively overwriting it out from under it too (with freshly recreated
  // wrapper objects on every write) corrupted its internal drag tracking:
  // the dragged entry vanished entirely on drop instead of moving.
  let dragging = false
  let itemEntries: DndEntry<configedit.ItemDTO>[] = wrap(items)
  $: if (!dragging) itemEntries = wrap(items)

  // consider fires continuously during the drag (giving the live-shifting
  // preview via dndzone's own flip animation); finalize fires once,
  // settled, on drop or cancel. Only finalize commits to the real data.
  const syncItems = syncList<configedit.ItemDTO>({
    setEntries: (v) => (itemEntries = v),
    setDragging: (v) => (dragging = v),
    setList: (v) => (items = v),
  })

  let preview: configedit.PreviewDTO | null = null
  let previewDisplayName = ''
  let previewActionIdx = -1
  let actionPreview: configedit.ActionPreviewDTO | null = null

  let previewTimer: ReturnType<typeof setTimeout>
  $: if (selectedItem >= 0 && items[selectedItem]) schedulePreview()
  function schedulePreview() {
    clearTimeout(previewTimer)
    previewTimer = setTimeout(async () => {
      const item = items[selectedItem]
      if (!item) return
      if (!previewDisplayName && displays.length > 0) previewDisplayName = item.display || displays[0].name
      preview = await previewItem(item, envFields, displays, previewDisplayName || item.display)
    }, 250)
  }

  async function previewSelectedAction() {
    if (selectedItem < 0 || previewActionIdx < 0) {
      actionPreview = null
      return
    }
    const act = actions[previewActionIdx]
    if (!act) return
    actionPreview = await previewAction(items[selectedItem], envFields, act)
  }
</script>

<ListToolbar
  addLabel={t('tooltip.addItem')}
  removeLabel={t('tooltip.removeItem')}
  removeDisabled={selectedItem < 0}
  {reorderMode}
  reorderEnterLabel={t('tooltip.enterReorderMode')}
  reorderExitLabel={t('tooltip.exitReorderMode')}
  on:add={addItem}
  on:remove={() => confirmRemoveItem(selectedItem)}
  on:toggleReorder={toggleReorderMode}
/>
<div class="master-detail">
  <div
    class="master list"
    class:reorder-mode={reorderMode}
    use:sortableList={{ items: itemEntries, onSync: syncItems, dragDisabled: !reorderMode }}
  >
    {#each itemEntries as entry, i (entry.id)}
      <button
        class="row"
        class:selected={selectedItem === i}
        on:click={() => {
          if (reorderMode) return
          selectedItem = i
          previewActionIdx = -1
          actionPreview = null
        }}>{entry.ref.name || t('fallback.unnamed')}</button
      >
    {/each}
  </div>
  <div class="detail">
    {#if selectedItem >= 0 && items[selectedItem]}
      <label class="field">
        <span>{t('field.name')}</span>
        <input type="text" bind:value={items[selectedItem].name} />
      </label>
      <label class="field">
        <span>{t('field.display')}</span>
        <select bind:value={items[selectedItem].display}>
          <option value="">{t('option.defaultDisplay')}</option>
          {#each displays as d}<option value={d.name}>{d.name}</option>{/each}
        </select>
      </label>

      {#if allActionIds.length > 0}
        <div class="field">
          <span>{t('nav.actions')}</span>
          <CheckboxChipList options={allActionIds} bind:selected={items[selectedItem].actions} />
        </div>
      {/if}

      {#if allActionGroups.length > 0}
        <div class="field">
          <span>{t('field.itemActionGroupsList')}</span>
          <CheckboxChipList options={allActionGroups} bind:selected={items[selectedItem].actionGroups} />
        </div>
      {/if}

      <div class="field">
        <span>{t('field.customActions')}</span>
        {#each items[selectedItem].customActions as _, j (j)}
          <div class="nested-action">
            <ActionForm
              bind:action={items[selectedItem].customActions[j]}
              showId={false}
              {allActionGroups}
              {browseScriptFile}
              {previewScriptFile}
            />
            <button class="btn" type="button" on:click={() => confirmRemoveCustomAction(selectedItem, j)}
              >{t('button.removeCustomAction')}</button
            >
          </div>
        {/each}
        <button class="btn" type="button" on:click={() => addCustomAction(selectedItem)}
          >{t('button.addCustomAction')}</button
        >
      </div>

      <div class="field">
        <span>{t('nav.environment')}</span>
        <p class="hint">{t('hint.envItem')}</p>
        <FieldGrid bind:fields={items[selectedItem].fields} {validateField} />
      </div>

      <div class="preview-pane panel">
        <header class="panel-title"><span>{t('panel.preview')}</span></header>
        <div class="panel-body">
          {#if displays.length > 1}
            <label class="field">
              <span>{t('field.previewDisplay')}</span>
              <select bind:value={previewDisplayName} on:change={schedulePreview}>
                {#each displays as d}<option value={d.name}>{d.name}</option>{/each}
              </select>
            </label>
          {/if}
          {#if preview}
            {#if preview.error}
              <div class="validation-issue validation-error">{preview.error}</div>
            {/if}
            <p class="preview-label">{t('hint.listLabel')}<strong>{preview.listLabel}</strong></p>
            {#if preview.missingFields?.length}
              <p class="hint">{t('hint.missingFields', { fields: preview.missingFields.join(', ') })}</p>
            {/if}
            <div class="details-preview">{@html preview.detailsHtml}</div>
          {/if}

          {#if actions.length > 0}
            <label class="field">
              <span>{t('field.previewAction')}</span>
              <select bind:value={previewActionIdx} on:change={previewSelectedAction}>
                <option value={-1}>{t('option.none')}</option>
                {#each actions as a, i}<option value={i}>{a.title || a.id}</option>{/each}
              </select>
            </label>
            {#if actionPreview}
              {#if actionPreview.error}
                <div class="validation-issue validation-error">{actionPreview.error}</div>
              {/if}
              {#if actionPreview.script}
                <p class="preview-label">{t('hint.scriptLabel')}</p>
                <pre class="cmd-preview">{actionPreview.script}</pre>
              {:else}
                <p class="preview-label">{t('hint.commandLabel')}</p>
                <pre class="cmd-preview">{actionPreview.cmd}</pre>
              {/if}
              {#if actionPreview.description}<p class="hint action-desc-preview">{actionPreview.description}</p>{/if}
            {/if}
          {/if}
        </div>
      </div>
    {:else}
      <div class="empty">{t('empty.selectItemOrAdd')}</div>
    {/if}
  </div>
</div>

<style>
  .action-desc-preview {
    white-space: pre-wrap;
  }

  .nested-action {
    border: 1px solid var(--sm-border);
    border-radius: 6px;
    padding: 8px;
    margin-bottom: 8px;
    background: var(--sm-bg-deep);
  }

  .preview-pane {
    margin-top: 16px;
  }

  .cmd-preview {
    background: var(--sm-bg-deep);
    border-radius: 4px;
    padding: 8px;
    font-family: "SF Mono", Consolas, monospace;
    font-size: 0.8rem;
    white-space: pre-wrap;
    word-break: break-word;
  }
</style>
