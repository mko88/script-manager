<script lang="ts">
  import { onMount } from 'svelte'
  import { dndzone } from 'svelte-dnd-action'
  import type { DndEvent } from 'svelte-dnd-action'
  import Toast from '@shared/components/Toast.svelte'
  import StringListEditor from './components/StringListEditor.svelte'
  import FieldGrid from './components/FieldGrid.svelte'
  import ActionForm from './components/ActionForm.svelte'
  import ViewModeIcon from './components/ViewModeIcon.svelte'
  import ListActionIcon from './components/ListActionIcon.svelte'
  import ToolbarIcon from './components/ToolbarIcon.svelte'
  import { t } from './messages'
  import { looksLikeSecretKey } from './secretKey'
  import {
    InitialState,
    NewBlank,
    BrowseOpen,
    BrowseSaveAs,
    Save,
    PreviewItem,
    PreviewAction,
    ValidateConfig,
    ValidateField,
    KnownTerminals,
    GetEditableMessages,
    SaveMessages,
  } from '../wailsjs/go/configedit/App.js'
  import type { configedit } from '../wailsjs/go/models'

  function emptyConfig(): configedit.ConfigDTO {
    return {
      shell: [],
      display: [],
      titles: { items: '', actions: '', details: '', command: '' },
      terminal: { mode: 'auto', name: '', argv: [] },
      envFields: [],
      items: [],
      actionGroups: [],
      actions: [],
    } as unknown as configedit.ConfigDTO
  }

  let cfg: configedit.ConfigDTO = emptyConfig()
  let path = ''
  let toast = ''
  let toastTimer: ReturnType<typeof setTimeout>
  let knownTerminals: string[] = []
  let validation: configedit.ValidationIssueDTO[] = []
  let initialized = false

  type Section = 'items' | 'actionGroups' | 'actions' | 'display' | 'env' | 'shell' | 'titles' | 'terminal' | 'messages'
  const sections: { key: Section; label: string }[] = [
    { key: 'items', label: t('nav.items') },
    { key: 'actionGroups', label: t('nav.actionGroups') },
    { key: 'actions', label: t('nav.actions') },
    { key: 'display', label: t('nav.displays') },
    { key: 'env', label: t('nav.environment') },
    { key: 'shell', label: t('nav.shell') },
    { key: 'titles', label: t('nav.titles') },
    { key: 'terminal', label: t('nav.terminal') },
    { key: 'messages', label: t('nav.messages') },
  ]
  let section: Section = 'items'
  $: sectionTitle = sections.find((s) => s.key === section)?.label ?? ''

  let selectedItem = -1
  let selectedActionGroup = -1
  let selectedAction = -1
  let selectedDisplay = -1

  let preview: configedit.PreviewDTO | null = null
  let previewDisplayName = ''
  let previewActionIdx = -1
  let actionPreview: configedit.ActionPreviewDTO | null = null

  // Displays section: preview an arbitrary item against the display being
  // edited (a display isn't tied to one item the way an item's own preview
  // is), with a layout toggle for how much space editing vs. previewing
  // gets. Kept across display switches (not reset in resetSelection) so you
  // can flip through displays while comparing the same item.
  type DisplayViewMode = 'edit' | 'preview' | 'split-v' | 'split-h'
  let previewItemForDisplay = -1
  let displayViewMode: DisplayViewMode = 'split-v'
  let displayPreview: configedit.PreviewDTO | null = null

  // Split-v sizes the edit pane by width, split-h by height; the other pane
  // always gets flex:1 to soak up whatever's left, so together they fill
  // the full available width (split-v) or height (split-h) — same
  // fixed-primary-pane-plus-flex:1-remainder pattern script-manager-gui's
  // own resizer uses (leftWidth/itemsHeight there). Minimums match gui's
  // MIN_COL/MIN_PANEL exactly — same reasoning: a width split needs more
  // headroom for usable content than a height split does.
  const DISPLAY_MIN_WIDTH = 180
  const DISPLAY_MIN_HEIGHT = 60
  const DISPLAY_RESIZER = 6
  let displayEditWidth = 480
  let displayEditHeight = 260
  let displaySplitEl: HTMLElement

  // The Details template's helper toolbar (Insert env / formatting buttons)
  // needs a real element handle for cursor/selection-based edits —
  // setRangeText operates on the DOM element directly, not on Svelte's
  // bound value.
  let detailsTextareaEl: HTMLTextAreaElement | undefined

  // Env var names available to insert into the Details template: global
  // Environment fields plus the currently-selected preview item's own
  // fields (if any), deduped — both already loaded client-side for the
  // preview feature, so no new backend call is needed just to list them.
  $: availableEnvKeys = Array.from(
    new Set([
      ...cfg.envFields.map((f) => f.key),
      ...(previewItemForDisplay >= 0 ? (cfg.items[previewItemForDisplay]?.fields ?? []).map((f) => f.key) : []),
    ]),
  ).filter((k) => k)

  function insertEnvVar(key: string) {
    // A key that looks like it holds a secret (same heuristic FieldGrid uses
    // to auto-lock a new field) is inserted already masked, not as a plain
    // reference someone would otherwise have to remember to wrap themselves.
    if (looksLikeSecretKey(key)) insertAtCursor('`{{mask .' + key + '}}`')
    else insertAtCursor(`{{.${key}}}`)
  }

  function onEnvSelectChange(e: Event) {
    const select = e.currentTarget as HTMLSelectElement
    const key = select.value
    if (key) insertEnvVar(key)
    select.selectedIndex = 0
  }

  function insertAtCursor(text: string) {
    const el = detailsTextareaEl
    if (!el) return
    const start = el.selectionStart ?? el.value.length
    const end = el.selectionEnd ?? el.value.length
    // 'select' (rather than 'end') leaves the just-inserted text highlighted,
    // so it's clear what was inserted and easy to overtype/replace.
    el.setRangeText(text, start, end, 'select')
    // setRangeText mutates the element directly and doesn't fire an input
    // event Svelte's bind:value would otherwise pick up, so the bound state
    // has to be synced back explicitly.
    cfg.display[selectedDisplay].details = el.value
    el.focus()
  }

  function wrapSelection(before: string, after: string = before) {
    const el = detailsTextareaEl
    if (!el) return
    const start = el.selectionStart ?? 0
    const end = el.selectionEnd ?? 0
    const selected = el.value.slice(start, end)
    el.setRangeText(before + selected + after, start, end, 'end')
    cfg.display[selectedDisplay].details = el.value
    el.focus()
    // Re-select just the original text (not the before/after markers), so
    // it stays visibly highlighted and a second formatting button or typed
    // replacement acts on the content, not the markup around it.
    el.setSelectionRange(start + before.length, start + before.length + selected.length)
  }

  // Matches a bare `{{.field}}`/`{{.field.nested}}` reference — the only
  // shape `mask` can meaningfully wrap; wrapping arbitrary literal text in
  // `{{mask ...}}` would just produce an invalid template.
  const FIELD_REF_RE = /^\{\{\s*(\.[\w.]+)\s*\}\}$/

  // Turns a selected `{{.field}}` reference into a masked `` `{{mask .field}}` ``
  // one — the same transform insertEnvVar applies automatically for a
  // secret-looking key, but usable on a variable already in the template
  // (e.g. one written by hand, or inserted before its key was recognized).
  // No-ops with a flash if the selection isn't a bare field reference.
  function maskSelection() {
    const el = detailsTextareaEl
    if (!el) return
    const start = el.selectionStart ?? 0
    const end = el.selectionEnd ?? 0
    const match = el.value.slice(start, end).match(FIELD_REF_RE)
    if (!match) {
      flash(t('toast.maskNeedsVariable'))
      return
    }
    const replacement = '`{{mask ' + match[1] + '}}`'
    el.setRangeText(replacement, start, end, 'select')
    cfg.display[selectedDisplay].details = el.value
    el.focus()
  }

  const DISPLAY_LAYOUT_KEY = 'sm-config-edit:displayLayout'

  onMount(async () => {
    const state = await InitialState()
    applyState(state)
    knownTerminals = await KnownTerminals()
    initialized = true
  })

  onMount(() => {
    try {
      const saved = JSON.parse(localStorage.getItem(DISPLAY_LAYOUT_KEY) ?? '{}')
      if (saved.viewMode) displayViewMode = saved.viewMode
      if (typeof saved.editWidth === 'number') displayEditWidth = saved.editWidth
      if (typeof saved.editHeight === 'number') displayEditHeight = saved.editHeight
    } catch {
      // ignore corrupt/missing layout, defaults already set
    }
  })

  function saveDisplayLayout() {
    localStorage.setItem(
      DISPLAY_LAYOUT_KEY,
      JSON.stringify({
        viewMode: displayViewMode,
        editWidth: displayEditWidth,
        editHeight: displayEditHeight,
      }),
    )
  }
  function setDisplayViewMode(mode: DisplayViewMode) {
    displayViewMode = mode
    saveDisplayLayout()
  }

  function dragDisplaySplit(e: MouseEvent) {
    e.preventDefault()
    const horizontal = displayViewMode === 'split-h'
    const min = horizontal ? DISPLAY_MIN_HEIGHT : DISPLAY_MIN_WIDTH
    const startPos = horizontal ? e.clientY : e.clientX
    const startSize = horizontal ? displayEditHeight : displayEditWidth
    function onMove(ev: MouseEvent) {
      const rect = displaySplitEl.getBoundingClientRect()
      const total = horizontal ? rect.height : rect.width
      const max = total - min - DISPLAY_RESIZER
      const pos = horizontal ? ev.clientY : ev.clientX
      const next = Math.min(max, Math.max(min, startSize + (pos - startPos)))
      if (horizontal) displayEditHeight = next
      else displayEditWidth = next
    }
    function onUp() {
      window.removeEventListener('mousemove', onMove)
      window.removeEventListener('mouseup', onUp)
      saveDisplayLayout()
    }
    window.addEventListener('mousemove', onMove)
    window.addEventListener('mouseup', onUp)
  }

  // The "clean" snapshot dirty is compared against, taken every time cfg is
  // set programmatically (load/save) rather than by the user editing a
  // field. A boolean toggled by "cfg was reassigned" doesn't work here: cfg
  // is reassigned by applyState too (loading is itself a reassignment), and
  // initialized flips true in a *later* tick than applyState's — so an
  // edge-triggered flag fires once, spuriously, right after every load. A
  // value comparison instead of an edge trigger sidesteps that entirely.
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
    previewActionIdx = -1
    preview = null
    actionPreview = null
    // A brand new/loaded config invalidates item indices entirely, unlike
    // switching between displays within the same config (where keeping the
    // same preview item is the point).
    previewItemForDisplay = -1
    displayPreview = null
  }

  function flash(msg: string) {
    toast = msg
    clearTimeout(toastTimer)
    toastTimer = setTimeout(() => (toast = ''), 3000)
  }

  let validateTimer: ReturnType<typeof setTimeout>
  function scheduleValidate() {
    clearTimeout(validateTimer)
    validateTimer = setTimeout(async () => {
      validation = await ValidateConfig(cfg)
    }, 300)
  }

  // dirty is a pure derived comparison against the last clean snapshot, not
  // an edge-triggered flag — see markClean's comment for why.
  $: dirty = initialized && JSON.stringify(cfg) !== cleanSnapshot

  // Validation re-runs on every nested edit too: Svelte's bind: chains
  // (StringListEditor, FieldGrid, ActionForm) all invalidate cfg up to this
  // root, which this statement is watching. Re-validating once extra right
  // after a load (before cleanSnapshot is compared) is harmless.
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

  async function saveAsConfig() {
    if (hasBlockingError) {
      flash(t('toast.fixBlockingErrors'))
      return
    }
    const target = await BrowseSaveAs()
    if (target) await doSave(target)
  }

  // Standard New/Open/Save/Save As shortcuts — Ctrl on Windows/Linux, Cmd on
  // Mac. These are modifier combos, not text a focused input would ever
  // insert, so it's safe to handle them regardless of what's focused.
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
        else saveConfig()
        break
    }
  }

  // The generated DTO classes for nested-object fields (ItemDTO, ConfigDTO)
  // carry a convertValues method, so a plain object literal isn't
  // structurally assignable — cast new entries the same way the rest of this
  // file's initial state does.
  function newItem(): configedit.ItemDTO {
    return { name: '', display: '', actions: [], actionGroups: [], customActions: [], fields: [] } as unknown as configedit.ItemDTO
  }
  function newAction(): configedit.ActionDTO {
    return {
      id: '',
      title: '',
      description: '',
      cmd: '',
      groups: [],
      noWait: false,
      interactive: true,
    } as unknown as configedit.ActionDTO
  }
  function newActionGroup(): configedit.ActionGroupDTO {
    return { id: '', title: '', color: '' } as unknown as configedit.ActionGroupDTO
  }
  function newDisplay(): configedit.DisplayDTO {
    return { name: '', list: '{{.name}}', details: '' } as unknown as configedit.DisplayDTO
  }

  function addItem() {
    cfg.items = [...cfg.items, newItem()]
    selectedItem = cfg.items.length - 1
    previewActionIdx = -1
  }
  function removeItem(i: number) {
    cfg.items = cfg.items.filter((_, idx) => idx !== i)
    if (selectedItem === i) selectedItem = -1
    else if (selectedItem > i) selectedItem -= 1
  }
  function confirmRemoveItem(i: number) {
    const name = cfg.items[i]?.name || t('fallback.unnamed')
    if (confirm(t('confirm.removeItem', { name }))) removeItem(i)
  }

  function addAction() {
    cfg.actions = [...cfg.actions, newAction()]
    selectedAction = cfg.actions.length - 1
  }
  function removeAction(i: number) {
    cfg.actions = cfg.actions.filter((_, idx) => idx !== i)
    if (selectedAction === i) selectedAction = -1
    else if (selectedAction > i) selectedAction -= 1
  }
  function confirmRemoveAction(i: number) {
    const name = cfg.actions[i]?.title || cfg.actions[i]?.id || t('fallback.untitled')
    if (confirm(t('confirm.removeAction', { name }))) removeAction(i)
  }

  function addActionGroup() {
    cfg.actionGroups = [...cfg.actionGroups, newActionGroup()]
    selectedActionGroup = cfg.actionGroups.length - 1
  }
  // How many actions/items/custom-actions currently reference a group id —
  // used both to warn before deleting and to actually scrub the reference so
  // deleting a group doesn't leave dangling entries in groups/actionGroups
  // lists (the picker UI already hides them, but the underlying data would
  // otherwise silently keep the stale id forever).
  function actionGroupRefCount(id: string): number {
    let count = 0
    for (const a of cfg.actions) if (a.groups.includes(id)) count++
    for (const it of cfg.items) {
      if (it.actionGroups.includes(id)) count++
      for (const ca of it.customActions) if (ca.groups.includes(id)) count++
    }
    return count
  }
  function removeActionGroupReferences(id: string) {
    cfg.actions = cfg.actions.map((a) => ({ ...a, groups: a.groups.filter((g) => g !== id) })) as unknown as configedit.ActionDTO[]
    cfg.items = cfg.items.map((it) => ({
      ...it,
      actionGroups: it.actionGroups.filter((g) => g !== id),
      customActions: it.customActions.map((ca) => ({ ...ca, groups: ca.groups.filter((g) => g !== id) })),
    })) as unknown as configedit.ItemDTO[]
  }
  function removeActionGroup(i: number) {
    const id = cfg.actionGroups[i]?.id
    cfg.actionGroups = cfg.actionGroups.filter((_, idx) => idx !== i)
    if (id) removeActionGroupReferences(id)
    if (selectedActionGroup === i) selectedActionGroup = -1
    else if (selectedActionGroup > i) selectedActionGroup -= 1
  }
  function confirmRemoveActionGroup(i: number) {
    const g = cfg.actionGroups[i]
    const name = g?.title || g?.id || t('fallback.unnamed')
    const refCount = g?.id ? actionGroupRefCount(g.id) : 0
    const refSuffix = refCount > 0 ? t('confirm.removeActionGroupRefSuffix', { count: refCount, plural: refCount > 1 ? 's' : '' }) : ''
    if (confirm(t('confirm.removeActionGroup', { name, refSuffix }))) removeActionGroup(i)
  }

  function addDisplay() {
    cfg.display = [...cfg.display, newDisplay()]
    selectedDisplay = cfg.display.length - 1
  }
  function copyDisplay() {
    const src = cfg.display[selectedDisplay]
    if (!src) return
    cfg.display = [...cfg.display, { ...src, name: `${src.name} - copy` }]
    selectedDisplay = cfg.display.length - 1
  }
  function removeDisplay(i: number) {
    cfg.display = cfg.display.filter((_, idx) => idx !== i)
    if (selectedDisplay === i) selectedDisplay = -1
    else if (selectedDisplay > i) selectedDisplay -= 1
  }
  function confirmRemoveDisplay(i: number) {
    const name = cfg.display[i]?.name || t('fallback.unnamed')
    if (confirm(t('confirm.removeDisplay', { name }))) removeDisplay(i)
  }

  function addCustomAction(itemIdx: number) {
    cfg.items[itemIdx].customActions = [...cfg.items[itemIdx].customActions, newAction()]
  }
  function removeCustomAction(itemIdx: number, i: number) {
    cfg.items[itemIdx].customActions = cfg.items[itemIdx].customActions.filter((_, idx) => idx !== i)
  }

  function toggleInList(list: string[], value: string): string[] {
    return list.includes(value) ? list.filter((v) => v !== value) : [...list, value]
  }

  // Drag-and-drop reordering for the Items/Action Groups/Actions master
  // lists, via svelte-dnd-action rather than native HTML5 drag-and-drop.
  // Native dnd's cursor is browser-controlled, and it disagreed with the
  // live reorder + animation (dragover hit-tests against the real,
  // already-reordered layout, while the FLIP transform visually lags
  // behind it) — that mismatch is what read as the cursor flickering
  // between "grab" and a no-drop icon. svelte-dnd-action drives
  // everything from pointer events instead, so there's no browser drag
  // cursor involved at all, and it handles the live-reorder animation and
  // cancelled-drag revert internally.
  //
  // dndzone needs every item to carry an "id" it can track across
  // reorders. None of items/actions/action groups reliably have one — an
  // item has no id field at all, and a brand-new action/action group
  // defaults its own id to "" (several can exist before being named) — so
  // each list is wrapped in a {id, ref} pair with a synthetic,
  // session-only id (a WeakMap keyed by object identity; never touches
  // the saved data) instead of reusing the domain id field.
  let dndSeq = 0
  const dndIds = new WeakMap<object, string>()
  function dndId(ref: object): string {
    let id = dndIds.get(ref)
    if (id === undefined) {
      id = `d${dndSeq++}`
      dndIds.set(ref, id)
    }
    return id
  }

  type DndEntry<T> = { id: string; ref: T }
  function wrap<T extends object>(list: T[]): DndEntry<T>[] {
    return list.map((ref) => ({ id: dndId(ref), ref }))
  }

  // svelte-dnd-action's consider/finalize are custom events the dndzone
  // action adds to its node, not real attributes of a plain <div> — this
  // project's Svelte/svelte-check versions don't have a working ambient
  // typing hook for that, so on:consider/on:finalize on the element
  // itself won't type-check. Attaching them here via plain
  // addEventListener sidesteps Svelte's (mistaken) typed-attribute check
  // entirely; nothing about actual behavior changes.
  type SyncFn<T> = (e: CustomEvent<DndEvent<DndEntry<T>>>, final: boolean) => void
  type SortableParams<T> = { items: DndEntry<T>[]; onSync: SyncFn<T>; dragDisabled: boolean }
  function sortableList<T extends object>(node: HTMLElement, params: SortableParams<T>) {
    const zone = dndzone(node, { items: params.items, flipDurationMs: 200, dragDisabled: params.dragDisabled })
    const considerHandler = (e: Event) => params.onSync(e as CustomEvent<DndEvent<DndEntry<T>>>, false)
    const finalizeHandler = (e: Event) => params.onSync(e as CustomEvent<DndEvent<DndEntry<T>>>, true)
    node.addEventListener('consider', considerHandler)
    node.addEventListener('finalize', finalizeHandler)
    return {
      update(newParams: SortableParams<T>) {
        zone.update?.({ items: newParams.items, flipDurationMs: 200, dragDisabled: newParams.dragDisabled })
      },
      destroy() {
        node.removeEventListener('consider', considerHandler)
        node.removeEventListener('finalize', finalizeHandler)
        zone.destroy?.()
      },
    }
  }

  // Reordering is opt-in: off by default, toggled per-visit via the
  // reorder-mode button in each list's toolbar (not persisted — reopening
  // a section, or the app, starts back in "reordering off"). Without this
  // gate, a plain click-to-select on a row is only one accidental pixel
  // of movement away from silently reordering the list.
  let reorderMode = false
  // Turning reorder mode on clears whatever's selected — keeping a
  // selection alive through a reorder means tracking its index through
  // every live-shifting consider event, which isn't worth the
  // complication. Turning it back off deliberately does *not* try to
  // restore the old selection; it's simply gone, same as if you'd
  // clicked away.
  function toggleReorderMode() {
    reorderMode = !reorderMode
    if (reorderMode) {
      selectedItem = -1
      selectedActionGroup = -1
      selectedAction = -1
    }
  }

  // Re-derived from cfg.* on any change EXCEPT while a drag is active —
  // during the drag, dndzone owns itemEntries via consider (below), and
  // reactively overwriting it out from under it too (with freshly
  // recreated wrapper objects on every one of our own cfg.items writes)
  // corrupted its internal drag tracking: the dragged entry vanished
  // entirely on drop instead of moving. Outside a drag this still stays
  // correct for free with no manual sync needed at every add/remove/load
  // call site.
  let dragging = false
  let itemEntries: DndEntry<configedit.ItemDTO>[] = wrap(cfg.items)
  let actionGroupEntries: DndEntry<configedit.ActionGroupDTO>[] = wrap(cfg.actionGroups)
  let actionEntries: DndEntry<configedit.ActionDTO>[] = wrap(cfg.actions)
  $: if (!dragging) itemEntries = wrap(cfg.items)
  $: if (!dragging) actionGroupEntries = wrap(cfg.actionGroups)
  $: if (!dragging) actionEntries = wrap(cfg.actions)

  // consider fires continuously during the drag (giving the live-shifting
  // preview via dndzone's own flip animation); finalize fires once,
  // settled, on drop or cancel. Only finalize commits to the real cfg.*
  // data — consider only updates what's rendered, exactly the pattern
  // svelte-dnd-action's own examples use. No selection to track through
  // the reorder here: entering reorder mode clears it and blocks
  // reselecting until reorder mode is off again (see toggleReorderMode),
  // so selectedX is always -1 for the whole reorder, nothing to keep in
  // sync with a live-changing index.
  function syncItems(e: CustomEvent<DndEvent<DndEntry<configedit.ItemDTO>>>, final: boolean) {
    itemEntries = e.detail.items
    dragging = !final
    if (final) cfg.items = itemEntries.filter((w) => w.ref).map((w) => w.ref)
  }
  function syncActionGroups(e: CustomEvent<DndEvent<DndEntry<configedit.ActionGroupDTO>>>, final: boolean) {
    actionGroupEntries = e.detail.items
    dragging = !final
    if (final) cfg.actionGroups = actionGroupEntries.filter((w) => w.ref).map((w) => w.ref)
  }
  function syncActions(e: CustomEvent<DndEvent<DndEntry<configedit.ActionDTO>>>, final: boolean) {
    actionEntries = e.detail.items
    dragging = !final
    if (final) cfg.actions = actionEntries.filter((w) => w.ref).map((w) => w.ref)
  }

  $: allActionIds = cfg.actions.map((a) => a.id).filter((id) => id)
  $: allActionGroups = cfg.actionGroups.map((g) => g.id).filter((id) => id)

  let previewTimer: ReturnType<typeof setTimeout>
  $: if (initialized && section === 'items' && selectedItem >= 0 && cfg.items[selectedItem]) schedulePreview()
  function schedulePreview() {
    clearTimeout(previewTimer)
    previewTimer = setTimeout(async () => {
      const item = cfg.items[selectedItem]
      if (!item) return
      if (!previewDisplayName && cfg.display.length > 0) previewDisplayName = item.display || cfg.display[0].name
      preview = await PreviewItem(item, cfg.envFields, cfg.display, previewDisplayName || item.display)
    }, 250)
  }

  async function previewSelectedAction() {
    if (selectedItem < 0 || previewActionIdx < 0) {
      actionPreview = null
      return
    }
    const act = cfg.actions[previewActionIdx]
    if (!act) return
    actionPreview = await PreviewAction(cfg.items[selectedItem], cfg.envFields, act)
  }

  let displayPreviewTimer: ReturnType<typeof setTimeout>
  // previewItemForDisplay >= -1 is always true — it's just there so Svelte
  // tracks it as a dependency of this statement (picking a different
  // preview item must re-trigger this the same way editing the template
  // does), not a real condition.
  $: if (
    initialized &&
    section === 'display' &&
    selectedDisplay >= 0 &&
    cfg.display[selectedDisplay] &&
    previewItemForDisplay >= -1
  )
    scheduleDisplayPreview()
  function scheduleDisplayPreview() {
    clearTimeout(displayPreviewTimer)
    displayPreviewTimer = setTimeout(async () => {
      const d = cfg.display[selectedDisplay]
      const item = cfg.items[previewItemForDisplay]
      if (!d || !item) {
        displayPreview = null
        return
      }
      displayPreview = await PreviewItem(item, cfg.envFields, cfg.display, d.name)
    }, 250)
  }

  // Messages section: edits either app's runtime message-override file
  // (script-manager-gui.messages.json / sm-config-edit.messages.json),
  // flattened into dotted-key rows for a simple key+text-input editor —
  // the same dotted paths t() itself resolves. Independent of cfg's own
  // dirty/save flow: a different file, a different Save action.
  type MessagesTarget = 'gui' | 'configedit'
  let messagesTarget: MessagesTarget = 'gui'
  let messagesRows: { key: string; value: string }[] = []
  let messagesError = ''

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
    const groups = new Map<string, { key: string; value: string }[]>()
    for (const row of messagesRows) {
      const category = row.key.split('.')[0]
      if (!groups.has(category)) groups.set(category, [])
      groups.get(category)!.push(row)
    }
    return Array.from(groups, ([category, rows]) => ({ category, rows }))
  })()

  $: if (initialized && section === 'messages') loadMessages(messagesTarget)

  async function loadMessages(target: MessagesTarget) {
    messagesError = ''
    try {
      messagesRows = flattenMessages(await GetEditableMessages(target))
    } catch (err) {
      messagesRows = []
      messagesError = String(err)
    }
  }

  async function saveMessagesSection() {
    try {
      await SaveMessages(messagesTarget, unflattenMessages(messagesRows))
      flash(t('messagesEditor.saved'))
    } catch (err) {
      flash(t('messagesEditor.saveFailed', { error: String(err) }))
    }
  }
</script>

<svelte:window on:keydown={handleGlobalKeydown} />

<main class="app-shell">
  <header class="toolbar">
    <button class="btn icon-btn" type="button" title={t('tooltip.newTitle')} aria-label={t('tooltip.newAria')} on:click={newConfig}
      ><ToolbarIcon mode="new" /></button
    >
    <button class="btn icon-btn" type="button" title={t('tooltip.openTitle')} aria-label={t('tooltip.openAria')} on:click={openConfig}
      ><ToolbarIcon mode="open" /></button
    >
    <button
      class="btn btn-primary icon-btn"
      type="button"
      title={t('tooltip.saveTitle')}
      aria-label={t('tooltip.saveAria')}
      disabled={hasBlockingError}
      on:click={saveConfig}><ToolbarIcon mode="save" /></button
    >
    <button
      class="btn icon-btn"
      type="button"
      title={t('tooltip.saveAsTitle')}
      aria-label={t('tooltip.saveAsAria')}
      disabled={hasBlockingError}
      on:click={saveAsConfig}><ToolbarIcon mode="save-as" /></button
    >
    <span class="toolbar-path">{path || t('text.unsaved')}{dirty ? t('text.dirtyMarker') : ''}</span>
  </header>

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
    <nav class="section-nav list">
      {#each sections as s (s.key)}
        <button class="row" class:selected={section === s.key} on:click={() => (section = s.key)}>{s.label}</button>
      {/each}
    </nav>

    <section class="panel main-panel">
      <header class="panel-title"><span>{sectionTitle}</span></header>
      <div
        class="panel-body"
        class:list-body={section === 'items' || section === 'actionGroups' || section === 'actions' || section === 'messages'}
      >
        {#if section === 'shell'}
          <p class="hint">{t('hint.shellCommandPrefix')}<code>pwsh -NoLogo -Command</code>.</p>
          <StringListEditor bind:items={cfg.shell} placeholder={t('placeholder.shellCommand')} />
        {:else if section === 'titles'}
          <label class="field">
            <span>{t('field.itemsPaneTitle')}</span>
            <input type="text" bind:value={cfg.titles.items} placeholder={t('nav.items')} />
          </label>
          <label class="field">
            <span>{t('field.actionsPaneTitle')}</span>
            <input type="text" bind:value={cfg.titles.actions} placeholder={t('nav.actions')} />
          </label>
          <label class="field">
            <span>{t('field.detailsPaneTitle')}</span>
            <input type="text" bind:value={cfg.titles.details} placeholder={t('placeholder.detailsPaneTitle')} />
          </label>
          <label class="field">
            <span>{t('field.commandPaneTitle')}</span>
            <input type="text" bind:value={cfg.titles.command} placeholder={t('field.command')} />
          </label>
        {:else if section === 'terminal'}
          <div class="radio-group">
            <label><input type="radio" bind:group={cfg.terminal.mode} value="auto" /> {t('radio.autoDetect')}</label>
            <label><input type="radio" bind:group={cfg.terminal.mode} value="name" /> {t('radio.named')}</label>
            <label><input type="radio" bind:group={cfg.terminal.mode} value="argv" /> {t('radio.customCommand')}</label>
          </div>
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
          <div class="display-section">
            <div class="display-select-row">
              <label class="field display-select-field">
                <span>{t('field.display')}</span>
                <select bind:value={selectedDisplay}>
                  <option value={-1}>{t('option.selectDisplay')}</option>
                  {#each cfg.display as d, i (i)}<option value={i}>{d.name || t('option.unnamedDisplay', { n: i + 1 })}</option
                    >{/each}
                </select>
              </label>
              <button class="btn icon-btn" type="button" title={t('tooltip.addDisplay')} aria-label={t('tooltip.addDisplay')} on:click={addDisplay}
                ><ListActionIcon mode="add" /></button
              >
              <button
                class="btn icon-btn"
                type="button"
                title={t('tooltip.copyDisplay')}
                aria-label={t('tooltip.copyDisplay')}
                disabled={selectedDisplay < 0}
                on:click={copyDisplay}><ListActionIcon mode="copy" /></button
              >
              <button
                class="btn icon-btn"
                type="button"
                title={t('tooltip.removeDisplay')}
                aria-label={t('tooltip.removeDisplay')}
                disabled={selectedDisplay < 0}
                on:click={() => confirmRemoveDisplay(selectedDisplay)}><ListActionIcon mode="remove" /></button
              >
            </div>

            {#if selectedDisplay >= 0 && cfg.display[selectedDisplay]}
              <label class="field">
                <span>{t('field.name')}</span>
                <input type="text" bind:value={cfg.display[selectedDisplay].name} />
              </label>

              <div class="display-toolbar">
                <div class="view-mode-group">
                  <button
                    class="btn icon-btn"
                    class:active={displayViewMode === 'edit'}
                    type="button"
                    title={t('tooltip.editOnly')}
                    aria-label={t('tooltip.editOnly')}
                    on:click={() => setDisplayViewMode('edit')}><ViewModeIcon mode="edit" /></button
                  >
                  <button
                    class="btn icon-btn"
                    class:active={displayViewMode === 'preview'}
                    type="button"
                    title={t('tooltip.previewOnly')}
                    aria-label={t('tooltip.previewOnly')}
                    on:click={() => setDisplayViewMode('preview')}><ViewModeIcon mode="preview" /></button
                  >
                  <button
                    class="btn icon-btn"
                    class:active={displayViewMode === 'split-v'}
                    type="button"
                    title={t('tooltip.sideBySide')}
                    aria-label={t('tooltip.sideBySide')}
                    on:click={() => setDisplayViewMode('split-v')}><ViewModeIcon mode="split-v" /></button
                  >
                  <button
                    class="btn icon-btn"
                    class:active={displayViewMode === 'split-h'}
                    type="button"
                    title={t('tooltip.stacked')}
                    aria-label={t('tooltip.stacked')}
                    on:click={() => setDisplayViewMode('split-h')}><ViewModeIcon mode="split-h" /></button
                  >
                </div>
                <label class="field preview-item-picker">
                  <span>{t('field.previewItem')}</span>
                  <select bind:value={previewItemForDisplay} on:change={scheduleDisplayPreview}>
                    <option value={-1}>{t('option.none')}</option>
                    {#each cfg.items as it, i}<option value={i}>{it.name || t('option.unnamedItem', { n: i + 1 })}</option
                      >{/each}
                  </select>
                </label>
              </div>

              <div
                class="display-edit-preview"
                class:split-v={displayViewMode === 'split-v'}
                class:split-h={displayViewMode === 'split-h'}
                bind:this={displaySplitEl}
              >
                {#if displayViewMode !== 'preview'}
                  <div
                    class="edit-pane panel"
                    style={displayViewMode === 'split-v'
                      ? `flex: 0 1 ${displayEditWidth}px`
                      : displayViewMode === 'split-h'
                        ? `flex: 0 1 ${displayEditHeight}px`
                        : ''}
                  >
                    <header class="panel-title"><span>{t('panel.edit')}</span></header>
                    <div class="panel-body edit-pane-body">
                      <label class="field list-template-field">
                        <span>{t('field.listTemplate')}</span>
                        <input type="text" bind:value={cfg.display[selectedDisplay].list} />
                      </label>
                      <label class="field details-template-field">
                        <span>{t('field.detailsTemplate')}</span>
                        <div class="details-helper-toolbar">
                          <select class="env-insert-select" title={t('tooltip.insertEnvVar')} on:change={onEnvSelectChange}>
                            <option value="">{t('option.insertEnv')}</option>
                            {#each availableEnvKeys as key (key)}<option value={key}>{key}</option>{/each}
                          </select>
                          <button class="btn icon-btn" type="button" title={t('tooltip.bold')} on:click={() => wrapSelection('**')}
                            ><strong>B</strong></button
                          >
                          <button class="btn icon-btn" type="button" title={t('tooltip.italic')} on:click={() => wrapSelection('_')}
                            ><em>I</em></button
                          >
                          <button
                            class="btn icon-btn"
                            type="button"
                            title={t('tooltip.highlight')}
                            on:click={() => wrapSelection('`')}><code>`</code></button
                          >
                          <button class="btn icon-btn" type="button" title={t('tooltip.mask')} on:click={maskSelection}
                            ><svg viewBox="0 0 16 16" width="13" height="13" aria-hidden="true">
                              <rect x="3.5" y="7" width="9" height="6.5" rx="1.2" fill="none" stroke="currentColor" stroke-width="1.3" />
                              <path d="M5 7V5a3 3 0 0 1 6 0v2" fill="none" stroke="currentColor" stroke-width="1.3" />
                            </svg></button
                          >
                        </div>
                        <textarea bind:value={cfg.display[selectedDisplay].details} bind:this={detailsTextareaEl}
                        ></textarea>
                      </label>
                    </div>
                  </div>
                {/if}
                {#if displayViewMode === 'split-v' || displayViewMode === 'split-h'}
                  <!-- svelte-ignore a11y-no-static-element-interactions -->
                  <div
                    class="resizer {displayViewMode === 'split-h' ? 'horizontal' : 'vertical'}"
                    on:mousedown={dragDisplaySplit}
                  ></div>
                {/if}
                {#if displayViewMode !== 'edit'}
                  <div class="preview-pane-inline panel">
                    <header class="panel-title"><span>{t('panel.preview')}</span></header>
                    <div class="panel-body">
                      {#if previewItemForDisplay < 0}
                        <div class="empty">{t('empty.pickItemToPreview')}</div>
                      {:else if displayPreview}
                        {#if displayPreview.error}
                          <div class="validation-issue validation-error">{displayPreview.error}</div>
                        {/if}
                        <p class="preview-label">{t('hint.listLabel')}<strong>{displayPreview.listLabel}</strong></p>
                        {#if displayPreview.missingFields?.length}
                          <p class="hint">{t('hint.missingFields', { fields: displayPreview.missingFields.join(', ') })}</p>
                        {/if}
                        <div class="details-preview">{@html displayPreview.detailsHtml}</div>
                      {/if}
                    </div>
                  </div>
                {/if}
              </div>
            {:else}
              <div class="empty">{t('empty.selectDisplayOrAdd')}</div>
            {/if}
          </div>
        {:else if section === 'actionGroups'}
          <div class="list-toolbar">
            <button
              class="btn icon-btn"
              type="button"
              title={t('tooltip.addActionGroup')}
              aria-label={t('tooltip.addActionGroup')}
              on:click={addActionGroup}><ListActionIcon mode="add" /></button
            >
            <button
              class="btn icon-btn"
              type="button"
              title={t('tooltip.removeActionGroup')}
              aria-label={t('tooltip.removeActionGroup')}
              disabled={selectedActionGroup < 0}
              on:click={() => confirmRemoveActionGroup(selectedActionGroup)}><ListActionIcon mode="remove" /></button
            >
            <button
              class="btn icon-btn"
              class:active={reorderMode}
              type="button"
              title={reorderMode ? t('tooltip.exitReorderMode') : t('tooltip.enterReorderMode')}
              aria-label={reorderMode ? t('tooltip.exitReorderMode') : t('tooltip.enterReorderMode')}
              on:click={toggleReorderMode}><ListActionIcon mode="reorder" /></button
            >
          </div>
          <div class="master-detail">
            <div
              class="master list"
              class:reorder-mode={reorderMode}
              use:sortableList={{ items: actionGroupEntries, onSync: syncActionGroups, dragDisabled: !reorderMode }}
            >
              {#each actionGroupEntries as entry, i (entry.id)}
                <button
                  class="row"
                  class:selected={selectedActionGroup === i}
                  on:click={() => {
                    if (!reorderMode) selectedActionGroup = i
                  }}
                >
                  <span class="group-swatch" style="background: {entry.ref.color || 'var(--sm-border)'}"></span>
                  {entry.ref.title || entry.ref.id || t('fallback.unnamed')}
                </button>
              {/each}
            </div>
            <div class="detail">
              {#if selectedActionGroup >= 0 && cfg.actionGroups[selectedActionGroup]}
                <label class="field">
                  <span>{t('field.id')}</span>
                  <input
                    type="text"
                    bind:value={cfg.actionGroups[selectedActionGroup].id}
                    placeholder={t('placeholder.actionGroupId')}
                  />
                </label>
                <label class="field">
                  <span>{t('field.title')}</span>
                  <input
                    type="text"
                    bind:value={cfg.actionGroups[selectedActionGroup].title}
                    placeholder={t('placeholder.actionGroupTitle')}
                  />
                </label>
                <div class="field">
                  <span>{t('field.color')}</span>
                  <div class="color-field">
                    <input
                      type="color"
                      value={/^#[0-9a-fA-F]{6}$/.test(cfg.actionGroups[selectedActionGroup].color)
                        ? cfg.actionGroups[selectedActionGroup].color
                        : '#7fd4ff'}
                      on:input={(e) => (cfg.actionGroups[selectedActionGroup].color = e.currentTarget.value)}
                      title={t('tooltip.pickColor')}
                    />
                    <input
                      type="text"
                      bind:value={cfg.actionGroups[selectedActionGroup].color}
                      placeholder={t('placeholder.actionGroupColor')}
                    />
                  </div>
                </div>
              {:else}
                <div class="empty">{t('empty.selectActionGroupOrAdd')}</div>
              {/if}
            </div>
          </div>
        {:else if section === 'actions'}
          <div class="list-toolbar">
            <button class="btn icon-btn" type="button" title={t('tooltip.addAction')} aria-label={t('tooltip.addAction')} on:click={addAction}
              ><ListActionIcon mode="add" /></button
            >
            <button
              class="btn icon-btn"
              type="button"
              title={t('tooltip.removeAction')}
              aria-label={t('tooltip.removeAction')}
              disabled={selectedAction < 0}
              on:click={() => confirmRemoveAction(selectedAction)}><ListActionIcon mode="remove" /></button
            >
            <button
              class="btn icon-btn"
              class:active={reorderMode}
              type="button"
              title={reorderMode ? t('tooltip.exitReorderMode') : t('tooltip.enterReorderMode')}
              aria-label={reorderMode ? t('tooltip.exitReorderMode') : t('tooltip.enterReorderMode')}
              on:click={toggleReorderMode}><ListActionIcon mode="reorder" /></button
            >
          </div>
          <div class="master-detail">
            <div
              class="master list"
              class:reorder-mode={reorderMode}
              use:sortableList={{ items: actionEntries, onSync: syncActions, dragDisabled: !reorderMode }}
            >
              {#each actionEntries as entry, i (entry.id)}
                <button
                  class="row"
                  class:selected={selectedAction === i}
                  on:click={() => {
                    if (!reorderMode) selectedAction = i
                  }}>{entry.ref.title || entry.ref.id || t('fallback.untitled')}</button
                >
              {/each}
            </div>
            <div class="detail">
              {#if selectedAction >= 0 && cfg.actions[selectedAction]}
                <ActionForm bind:action={cfg.actions[selectedAction]} {allActionGroups} />
              {:else}
                <div class="empty">{t('empty.selectActionOrAdd')}</div>
              {/if}
            </div>
          </div>
        {:else if section === 'items'}
          <div class="list-toolbar">
            <button class="btn icon-btn" type="button" title={t('tooltip.addItem')} aria-label={t('tooltip.addItem')} on:click={addItem}
              ><ListActionIcon mode="add" /></button
            >
            <button
              class="btn icon-btn"
              type="button"
              title={t('tooltip.removeItem')}
              aria-label={t('tooltip.removeItem')}
              disabled={selectedItem < 0}
              on:click={() => confirmRemoveItem(selectedItem)}><ListActionIcon mode="remove" /></button
            >
            <button
              class="btn icon-btn"
              class:active={reorderMode}
              type="button"
              title={reorderMode ? t('tooltip.exitReorderMode') : t('tooltip.enterReorderMode')}
              aria-label={reorderMode ? t('tooltip.exitReorderMode') : t('tooltip.enterReorderMode')}
              on:click={toggleReorderMode}><ListActionIcon mode="reorder" /></button
            >
          </div>
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
              {#if selectedItem >= 0 && cfg.items[selectedItem]}
                <label class="field">
                  <span>{t('field.name')}</span>
                  <input type="text" bind:value={cfg.items[selectedItem].name} />
                </label>
                <label class="field">
                  <span>{t('field.display')}</span>
                  <select bind:value={cfg.items[selectedItem].display}>
                    <option value="">{t('option.defaultDisplay')}</option>
                    {#each cfg.display as d}<option value={d.name}>{d.name}</option>{/each}
                  </select>
                </label>

                {#if allActionIds.length > 0}
                  <div class="field">
                    <span>{t('nav.actions')}</span>
                    <div class="checkbox-list">
                      {#each allActionIds as id}
                        <label class="checkbox-chip">
                          <input
                            type="checkbox"
                            checked={cfg.items[selectedItem].actions.includes(id)}
                            on:change={() =>
                              (cfg.items[selectedItem].actions = toggleInList(cfg.items[selectedItem].actions, id))}
                          />
                          {id}
                        </label>
                      {/each}
                    </div>
                  </div>
                {/if}

                {#if allActionGroups.length > 0}
                  <div class="field">
                    <span>{t('field.itemActionGroupsList')}</span>
                    <div class="checkbox-list">
                      {#each allActionGroups as g}
                        <label class="checkbox-chip">
                          <input
                            type="checkbox"
                            checked={cfg.items[selectedItem].actionGroups.includes(g)}
                            on:change={() =>
                              (cfg.items[selectedItem].actionGroups = toggleInList(
                                cfg.items[selectedItem].actionGroups,
                                g,
                              ))}
                          />
                          {g}
                        </label>
                      {/each}
                    </div>
                  </div>
                {/if}

                <div class="field">
                  <span>{t('field.customActions')}</span>
                  {#each cfg.items[selectedItem].customActions as _, j (j)}
                    <div class="nested-action">
                      <ActionForm bind:action={cfg.items[selectedItem].customActions[j]} showId={false} {allActionGroups} />
                      <button class="btn" type="button" on:click={() => removeCustomAction(selectedItem, j)}
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
                  <FieldGrid bind:fields={cfg.items[selectedItem].fields} validateField={ValidateField} />
                </div>

                <div class="preview-pane panel">
                  <header class="panel-title"><span>{t('panel.preview')}</span></header>
                  <div class="panel-body">
                    {#if cfg.display.length > 1}
                      <label class="field">
                        <span>{t('field.previewDisplay')}</span>
                        <select bind:value={previewDisplayName} on:change={schedulePreview}>
                          {#each cfg.display as d}<option value={d.name}>{d.name}</option>{/each}
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

                    {#if cfg.actions.length > 0}
                      <label class="field">
                        <span>{t('field.previewAction')}</span>
                        <select bind:value={previewActionIdx} on:change={previewSelectedAction}>
                          <option value={-1}>{t('option.none')}</option>
                          {#each cfg.actions as a, i}<option value={i}>{a.title || a.id}</option>{/each}
                        </select>
                      </label>
                      {#if actionPreview}
                        {#if actionPreview.error}
                          <div class="validation-issue validation-error">{actionPreview.error}</div>
                        {/if}
                        <p class="preview-label">{t('hint.commandLabel')}</p>
                        <pre class="cmd-preview">{actionPreview.cmd}</pre>
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
        {:else if section === 'messages'}
          <div class="messages-target-row">
            <span class="messages-target-label">{t('messagesEditor.targetLabel')}</span>
            <div class="view-mode-group">
              <button
                class="btn"
                class:active={messagesTarget === 'gui'}
                type="button"
                on:click={() => (messagesTarget = 'gui')}>{t('messagesEditor.targetGui')}</button
              >
              <button
                class="btn"
                class:active={messagesTarget === 'configedit'}
                type="button"
                on:click={() => (messagesTarget = 'configedit')}>{t('messagesEditor.targetConfigEdit')}</button
              >
            </div>
            <button class="btn btn-primary" type="button" on:click={saveMessagesSection}
              >{t('messagesEditor.saveButton')}</button
            >
          </div>
          {#if messagesError}
            <div class="validation-issue validation-error">{messagesError}</div>
          {:else}
            <div class="messages-rows">
              {#each messagesGroups as group (group.category)}
                <div class="messages-group">
                  <h3 class="messages-group-title">{group.category}</h3>
                  {#each group.rows as row (row.key)}
                    <label class="field messages-row">
                      <span class="messages-row-key">{row.key}</span>
                      <input type="text" bind:value={row.value} />
                    </label>
                  {/each}
                </div>
              {/each}
            </div>
          {/if}
        {/if}
      </div>
    </section>
  </div>

  <Toast message={toast} />
</main>

<style>
  .app-shell {
    display: flex;
    flex-direction: column;
    height: 100vh;
    box-sizing: border-box;
    padding: 8px;
    gap: 8px;
    text-align: left;
  }

  .toolbar {
    flex: none;
    display: flex;
    align-items: center;
    gap: 8px;
  }

  .toolbar-path {
    margin-left: 8px;
    color: var(--sm-text-muted);
    font-size: 0.85rem;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
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
    color: var(--sm-accent-warm);
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

  /* Items/Action Groups/Actions: .master-detail's height:100% only works
     out if it's the sole child filling panel-body — with .list-toolbar as
     a sibling above it, "100%" of the same box overflows by the toolbar's
     own height, forcing a scrollbar on panel-body that a user could never
     actually need (master/detail already scroll internally). Making
     panel-body a column flex here — toolbar fixed-height, master-detail
     filling exactly what's left — removes that spurious overflow and, as a
     side effect, keeps the toolbar permanently visible above the list
     without needing script-manager-gui's position:sticky trick (nothing
     here scrolls at the panel-body level to begin with). The extra
     specificity over the plain .panel-body rule is deliberate so this
     doesn't depend on CSS source order between the shared theme and this
     component's scoped styles. */
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

  .action-desc-preview {
    white-space: pre-wrap;
  }

  .hint code {
    background: var(--sm-bg-deep);
    padding: 1px 4px;
    border-radius: 3px;
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
  .field select,
  .field textarea {
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
     dark theme. appearance: none removes that entirely (including the
     arrow itself), so a plain custom chevron replaces it here instead. */
  .field select {
    appearance: none;
    -webkit-appearance: none;
    -moz-appearance: none;
    background-image: url("data:image/svg+xml,%3Csvg xmlns='http://www.w3.org/2000/svg' width='10' height='6' viewBox='0 0 10 6'%3E%3Cpath d='M1 1l4 4 4-4' fill='none' stroke='%23a9b6c8' stroke-width='1.4' stroke-linecap='round' stroke-linejoin='round'/%3E%3C/svg%3E");
    background-repeat: no-repeat;
    background-position: right 10px center;
    padding-right: 28px;
  }

  .radio-group {
    display: flex;
    gap: 16px;
    margin-bottom: 10px;
    font-size: 0.85rem;
  }

  .master-detail {
    display: flex;
    gap: 10px;
    flex: 1 1 auto;
    min-height: 0;
  }

  .master {
    flex: 0 0 200px;
    overflow-y: auto;
  }

  .detail {
    flex: 1 1 auto;
    min-width: 0;
    overflow-y: auto;
    padding-right: 4px;
  }

  .master.reorder-mode .row {
    cursor: grab;
  }

  /* Displays has no master-list sidebar (a combobox picks the display
     instead), so it doesn't use .master-detail/.detail at all. Its
     edit/preview split still needs a real, bounded height to resize
     within, so display-section fills the available height exactly and
     lets display-edit-preview's two panes scroll internally instead. */
  .display-section {
    display: flex;
    flex-direction: column;
    gap: 10px;
    height: 100%;
    min-height: 0;
    overflow: hidden;
  }

  .display-select-row {
    flex: none;
    display: flex;
    align-items: flex-end;
    gap: 12px;
  }

  .display-select-field {
    flex: 0 1 320px;
    margin-bottom: 0;
  }

  .display-toolbar {
    display: flex;
    align-items: flex-end;
    justify-content: space-between;
    gap: 12px;
    flex-wrap: wrap;
  }

  .view-mode-group {
    display: flex;
    gap: 4px;
  }

  .icon-btn {
    display: flex;
    align-items: center;
    justify-content: center;
    padding: 6px 9px;
  }

  .icon-btn:disabled {
    opacity: 0.35;
    cursor: default;
  }

  .list-toolbar {
    flex: none;
    display: flex;
    gap: 4px;
    margin-bottom: 8px;
  }

  .view-mode-group .btn.active,
  .list-toolbar .btn.active {
    background: var(--sm-accent-warm);
    border-color: var(--sm-accent-warm);
    color: var(--sm-bg);
    font-weight: 700;
  }

  .preview-item-picker {
    flex: 0 0 220px;
    margin-bottom: 0;
  }

  /* Row by default (also covers single-pane Edit-only/Preview-only modes,
     where whichever one pane is present just fills 100% via flex:1 below);
     .split-h switches to a column so the panes stack instead. */
  .display-edit-preview {
    display: flex;
    flex: 1 1 auto;
    min-height: 0;
    overflow: hidden;
  }

  .display-edit-preview.split-h {
    flex-direction: column;
  }

  /* Base flex-basis is 0, not auto: with auto, an unconstrained pane's basis
     is its max-content size — for the Preview pane that means the table's
     *unwrapped* natural width, which can be huge regardless of the
     word-wrap CSS below (max-content sizing ignores wrapping opportunities
     by definition). That huge implicit basis was swamping the edit pane's
     explicit pixel basis during flex-shrink, collapsing it to ~2px even
     though its own flex-basis said otherwise. flex-basis:0 makes both
     panes' share of space depend only on flex-grow/shrink and the explicit
     pixel size below, never on content. A single visible pane (Edit-only/
     Preview-only) still fills 100% via flex-grow regardless of basis. */
  .edit-pane,
  .preview-pane-inline {
    flex: 1 1 0;
    min-width: 0;
    min-height: 0;
  }

  .edit-pane-body {
    display: flex;
    flex-direction: column;
    gap: 8px;
    min-height: 0;
  }

  .list-template-field {
    flex: none;
  }

  .details-template-field {
    flex: 1 1 auto;
    min-height: 0;
    margin-bottom: 0;
  }

  .details-template-field textarea {
    flex: 1 1 auto;
    min-height: 60px;
    resize: none;
    font-family: "SF Mono", Consolas, monospace;
    font-size: 0.8rem;
  }

  .details-helper-toolbar {
    flex: none;
    display: flex;
    align-items: center;
    gap: 4px;
    margin-bottom: 4px;
  }

  .details-helper-toolbar .icon-btn {
    font-family: "SF Mono", Consolas, monospace;
    font-size: 0.8rem;
    line-height: 1;
  }

  .env-insert-select {
    max-width: 160px;
  }

  .checkbox-list {
    display: flex;
    flex-wrap: wrap;
    gap: 6px;
  }

  .checkbox-chip {
    display: flex;
    align-items: center;
    gap: 4px;
    background: var(--sm-bg-deep);
    border: 1px solid var(--sm-border);
    border-radius: 999px;
    padding: 2px 9px;
    font-size: 0.75rem;
    cursor: pointer;
  }

  .group-swatch {
    display: inline-block;
    width: 10px;
    height: 10px;
    border-radius: 50%;
    margin-right: 6px;
    vertical-align: middle;
    border: 1px solid var(--sm-border);
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

  .preview-label {
    margin: 4px 0;
    font-size: 0.85rem;
  }

  /* Mirrors script-manager-gui's .details-content rules for the same
     goldmark-rendered HTML. table-layout: fixed (gui's own table rule
     doesn't need this — it only ever resizes panes by height, never width)
     is the important addition here: without it, an unstyled <table> sizes
     itself to its widest cell and refuses to shrink below that, fighting
     the drag-to-resize divider in split-v mode — this pane, unlike gui's,
     genuinely needs to shrink to arbitrary widths. */
  .details-preview {
    font-size: 0.85rem;
    line-height: 1.5;
    margin-bottom: 8px;
    min-width: 0;
    overflow-wrap: break-word;
  }

  .details-preview :global(h1),
  .details-preview :global(h2),
  .details-preview :global(h3) {
    color: var(--sm-accent);
    margin: 0.6em 0 0.3em;
  }

  .details-preview :global(table) {
    table-layout: fixed;
    border-collapse: collapse;
    width: 100%;
  }

  .details-preview :global(td),
  .details-preview :global(th) {
    border: 1px solid var(--sm-border);
    padding: 4px 8px;
    text-align: left;
    overflow-wrap: break-word;
  }

  .details-preview :global(code) {
    background: var(--sm-bg-deep);
    color: #6ee7d8;
    padding: 1px 5px;
    border-radius: 3px;
    font-family: "SF Mono", Consolas, monospace;
    overflow-wrap: break-word;
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

  .messages-target-row {
    flex: none;
    display: flex;
    align-items: center;
    gap: 8px;
    margin-bottom: 10px;
  }

  .messages-target-label {
    color: var(--sm-text-muted);
    font-size: 0.8rem;
  }

  .messages-rows {
    flex: 1 1 auto;
    overflow-y: auto;
  }

  .messages-group {
    margin-bottom: 14px;
  }

  .messages-group-title {
    margin: 0 0 6px;
    font-size: 0.75rem;
    text-transform: uppercase;
    letter-spacing: 0.04em;
    color: var(--sm-text-muted);
  }

  .messages-row-key {
    font-family: "SF Mono", Consolas, monospace;
    font-size: 0.75rem;
  }
</style>
