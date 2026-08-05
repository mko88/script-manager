<script lang="ts">
  import { onMount, tick } from 'svelte'
  import Toast from '@shared/components/Toast.svelte'
  import { flash } from '@shared/toast'
  import { loadPersisted, savePersisted } from '@shared/persist'
  import { watchTheme } from '@shared/theme'
  import Icon from '@shared/components/Icon.svelte'
  import CollapseToggle from '@shared/components/CollapseToggle.svelte'
  import IconButton from '@shared/components/IconButton.svelte'
  import ScriptSource from '@shared/components/ScriptSource.svelte'
  import Panel from './components/Panel.svelte'
  import GroupFilter from './components/GroupFilter.svelte'
  import { t } from './messages'
  import { buildGroupColors, groupChipStyle } from './lib/groupColors'
  import { inlineStates, inlineKey, startInlineRun, cancelInlineRun } from './lib/inlineRuns'
  import { dragColumn, dragRow, topStyle, bottomStyle } from './lib/panelLayout'
  import {
    EventsOn,
    WindowMinimise,
    WindowToggleMaximise,
    Quit,
    BrowserOpenURL,
    WindowGetPosition,
    WindowGetSize,
    WindowSetPosition,
    WindowSetSize,
    WindowSetMinSize,
    ScreenGetAll,
  } from '../wailsjs/runtime'
  import {
    GetItems,
    GetActions,
    GetActionDetail,
    GetItemDetails,
    GetActionGroups,
    CopyToClipboard,
    ReloadConfig,
    BrowseConfig,
    LaunchConfigEditor,
    RunAction,
    LoadError,
    SetAlwaysOnTop,
    SetWindowOpacity,
    GetVersion,
  } from '../wailsjs/go/gui/App.js'
  import type { gui } from '../wailsjs/go/models'

  let items: gui.ItemDTO[] = []
  let actions: gui.ActionDTO[] = []
  let actionGroupCatalog: gui.ActionGroupDTO[] = []
  let details: gui.DetailsDTO | null = null
  let actionDetail: gui.ActionDetailDTO | null = null

  let selectedItem = -1
  let selectedActionIndex = -1
  // Two-way bound into GroupFilter; empty set means "All" — no filter.
  let selectedGroups = new Set<string>()

  // Live reload: internal/gui/themewatch.go watches sm-theme.json and
  // pushes a Wails event whenever sm-config-edit changes it, so a theme
  // switched or saved there shows up here without needing to relaunch.
  // This app never switches themes itself, so there's no reactive state to
  // update here — watchTheme applies the change internally regardless.
  onMount(() => watchTheme(EventsOn, () => {}))

  let inlineOutputEl: HTMLElement | undefined

  // Autoscrolls the inline output box to the newest line as it streams in.
  // Called directly from the poll loop below when it's updating the
  // currently-viewed action, not from a `$:` reactive statement watching
  // output/inlineOutputEl together — that shape was tried first and
  // reliably broke Wails' own bound-method delivery (and, separately, an
  // EventSource-based version of this feature) in WebKitGTK:
  // inlineOutputEl is only bound once the <pre> below actually renders
  // (there must already be output for that), so the reactive statement's
  // own dependency on both variables together, right as new output
  // arrived, was the trigger. Root-caused by bisection, not fully
  // understood at the WebKitGTK level.
  async function scrollInlineOutputToEnd() {
    await tick()
    if (inlineOutputEl) inlineOutputEl.scrollTop = inlineOutputEl.scrollHeight
  }

  // What the Command pane actually displays — derived from the shared
  // inlineRuns store (see lib/inlineRuns) for whatever's currently selected,
  // defaulting to "never run" (blank, not running) when there's no entry yet.
  $: currentInline = selectedItem >= 0 && selectedActionIndex >= 0 ? $inlineStates[inlineKey(selectedItem, selectedActionIndex)] : undefined
  $: inlineRunning = currentInline?.running ?? false
  $: inlineOutput = currentInline?.output ?? ''
  $: inlineExitCode = currentInline?.exitCode ?? null

  // Which items/actions to show a running indicator for — every entry
  // still running, cross-referenced by itemIndex for the Items list and by
  // actionIndex (within the selected item) for the Actions list. Pure data
  // derivations, no DOM access — safe alongside the bug described above,
  // which was specifically about a reactive statement that touched the DOM.
  $: runningItemIndices = new Set(Object.values($inlineStates).filter((s) => s.running).map((s) => s.itemIndex))
  $: runningActionIndicesForSelectedItem = new Set(
    Object.values($inlineStates)
      .filter((s) => s.running && s.itemIndex === selectedItem)
      .map((s) => s.actionIndex),
  )
  // Last finished exit code per action of the selected item, for the
  // persistent green/red dot on action rows — the store keeps every pair
  // ever run this session, so this survives switching items/actions. While
  // a pair is running again its exitCode is null, so the row falls back to
  // the pulsing running indicator until the new result replaces the old.
  $: lastExitCodeByActionForSelectedItem = new Map(
    Object.values($inlineStates)
      .filter((s) => !s.running && s.exitCode !== null && s.itemIndex === selectedItem)
      .map((s) => [s.actionIndex, s.exitCode as number]),
  )

  $: filteredActions =
    selectedGroups.size === 0
      ? actions
      : actions.filter((a) => [...selectedGroups].every((g) => (a.groups ?? []).includes(g)))

  $: missingFields = details?.missingFields ?? []

  $: selectedItemLabel = items.find((i) => i.index === selectedItem)?.label ?? ''
  $: selectedActionLabel = actions.find((a) => a.index === selectedActionIndex)?.title ?? ''
  $: selectedActionGroups = actions.find((a) => a.index === selectedActionIndex)?.groups ?? []

  $: groupColors = buildGroupColors(actionGroupCatalog)

  onMount(async () => {
    const loadErr = await LoadError()
    if (loadErr) flash(t('toast.configLoadFailed', { error: loadErr }))
    items = await GetItems()
    actionGroupCatalog = await GetActionGroups()
    if (items.length > 0) selectItem(0)
  })

  async function selectItem(index: number) {
    selectedItem = index
    selectedActionIndex = -1
    selectedGroups = new Set()
    actionDetail = null
    // Picking an item is about inspecting it, not the command from
    // whatever action happened to be selected before — so Details takes
    // over the space and Command steps aside until an action is chosen.
    detailsCollapsed = false
    commandCollapsed = true
    saveLayout()
    actions = await GetActions(index)
    details = await GetItemDetails(index)
  }

  // A group-filter change can hide the selected action, so the selection is
  // always reset alongside it.
  function onGroupFilterChange() {
    selectedActionIndex = -1
    actionDetail = null
  }

  async function selectAction(index: number) {
    if (selectedItem < 0) return
    selectedActionIndex = index
    // Mirrors selectItem above: picking an action is about running or
    // inspecting its command, so Command takes over and Details steps
    // aside.
    detailsCollapsed = true
    commandCollapsed = false
    saveLayout()
    actionDetail = await GetActionDetail(selectedItem, index)
  }

  async function copyToClipboard(value: string) {
    try {
      await CopyToClipboard(value)
      flash(t('toast.copiedToClipboard'))
    } catch (err) {
      flash(t('toast.clipboardUnavailable', { error: String(err) }))
    }
  }

  function copyValue(idx: number) {
    if (!details) return
    const value = details.copyValues[idx]
    if (value === undefined) return
    copyToClipboard(value)
  }

  function onDetailsClick(e: MouseEvent) {
    const target = (e.target as HTMLElement).closest('[data-copy-idx]') as HTMLElement | null
    if (!target) return
    copyValue(Number(target.dataset.copyIdx))
  }

  function copyCmd() {
    const value = actionDetail?.cmd || actionDetail?.script
    if (!value) return
    copyToClipboard(value)
  }

  async function runAction() {
    if (selectedItem < 0 || selectedActionIndex < 0) return
    try {
      await RunAction(selectedItem, selectedActionIndex)
      flash(t('toast.runningInTerminal'))
    } catch (err) {
      flash(t('toast.runFailed', { error: String(err) }))
    }
  }

  // The run/poll mechanics live in lib/inlineRuns — these wrappers just tie
  // them to the current selection, plus the scroll side effect for whichever
  // run is on screen right now (a DOM concern that stays in this component;
  // see scrollInlineOutputToEnd's doc comment above).
  function runActionInline() {
    if (selectedItem < 0 || selectedActionIndex < 0) return
    startInlineRun(selectedItem, selectedActionIndex, (itemIndex, actionIndex) => {
      if (selectedItem === itemIndex && selectedActionIndex === actionIndex) {
        scrollInlineOutputToEnd()
      }
    })
  }

  function cancelInlineAction() {
    if (selectedItem < 0 || selectedActionIndex < 0) return
    cancelInlineRun(selectedItem, selectedActionIndex)
  }

  // Shared by reloadConfig (F5 / Refresh config) and browseConfig (Load
  // config) — both swap the backend's in-memory config out from under the
  // frontend, so both need the same items/actions/details re-fetch and
  // reselect-something-sane dance afterward.
  async function refreshAfterConfigChange() {
    actionGroupCatalog = await GetActionGroups()
    const newItems = await GetItems()
    items = newItems
    if (newItems.length === 0) {
      selectedItem = -1
      actions = []
      details = null
      actionDetail = null
    } else {
      await selectItem(Math.min(selectedItem < 0 ? 0 : selectedItem, newItems.length - 1))
    }
  }

  async function reloadConfig() {
    let warning = ''
    try {
      warning = await ReloadConfig()
    } catch (err) {
      flash(t('toast.reloadFailed', { error: String(err) }))
      return
    }
    await refreshAfterConfigChange()
    flash(warning ? t('toast.configReloadedWithWarning', { warning }) : t('toast.configReloaded'))
  }

  async function browseConfig() {
    let path = ''
    try {
      path = await BrowseConfig()
    } catch (err) {
      flash(t('toast.loadFailed', { error: String(err) }))
      return
    }
    if (!path) return // dialog cancelled
    await refreshAfterConfigChange()
    flash(t('toast.loaded', { path }))
  }

  async function launchConfigEditor() {
    try {
      const alreadyRunning = await LaunchConfigEditor()
      if (alreadyRunning) {
        flash(t('toast.configEditorAlreadyOpen'))
      }
    } catch (err) {
      flash(t('toast.openConfigEditorFailed', { error: String(err) }))
    }
  }

  function onKeyDown(e: KeyboardEvent) {
    if (e.key === 'F5') {
      e.preventDefault()
      reloadConfig()
    } else if ((e.ctrlKey || e.metaKey) && e.key.toLowerCase() === 'e') {
      e.preventDefault()
      launchConfigEditor()
    } else if (e.key === 'Escape' && (opacityPopoverOpen || aboutPopoverOpen)) {
      opacityPopoverOpen = false
      aboutPopoverOpen = false
    }
  }

  // --- Window controls (pin-on-top / transparency toolbar buttons) ---
  const WINDOW_KEY = 'script-manager-gui:window'
  const MIN_OPACITY = 20

  let alwaysOnTop = false
  let opacity = 100
  let opacityPopoverOpen = false
  let opacityControlEl: HTMLElement

  // --- Shrink-on-blur: while pinned always-on-top, fades the window down
  // to a tiny nub parked at the vertical center of the screen's right edge
  // whenever it loses OS focus, so it stays out of the way without
  // vanishing entirely. Hovering it pops it out to a bigger, fully-opaque
  // badge that's easy to see and click; moving away shrinks it back to the
  // nub after a short delay so it doesn't flicker shut mid-hover. Clicking
  // it in either state restores the saved size, position, and opacity.
  // Gated on alwaysOnTop itself (not just this checkbox) since without
  // pinning, a shrunk window would just vanish behind whatever's focused.
  //
  // The nub and badge share the same height and right edge — only the
  // width animates between them — so the pop only ever needs to grow/shrink
  // leftward from a fixed edge, never risking an overflow past the screen
  // the way animating height (or a right margin that itself moves) would.
  //
  // Both widths sit below Windows' minimum tracking width for a resizable
  // frameless window (SM_CXMINTRACK, ~120px), which would otherwise clamp
  // WindowSetSize so the window stayed wider than requested, overhung the
  // fixed right edge, and pushed the flex-centred icon off to the right.
  // We lower the window's min size to the nub's dimensions on the way in
  // (and restore it on the way out) so the requested widths are honoured.
  // The height (SHRUNK_HEIGHT) has always been above SM_CYMINTRACK, which
  // is why only the horizontal axis was ever affected.
  const NUB_WIDTH = 36
  const BADGE_WIDTH = 100
  const SHRUNK_HEIGHT = 100
  const SHRUNK_MARGIN = 16
  const HIDE_DELAY_MS = 300
  const POP_ANIM_MS = 180
  const SHRUNK_OPACITY = 80

  let shrinkOnBlur = false
  let isShrunk = false
  let isPoppedOut = false
  let shrunkRightEdgeX = 0
  let shrunkY = 0
  let hideShrunkTimer: ReturnType<typeof setTimeout> | null = null
  let popProgress = 0 // 0 = nub width/opacity, 1 = badge width/opacity
  let popAnimFrame: number | null = null
  let savedGeometry: { x: number; y: number; w: number; h: number } | null = null

  function applyPopProgress() {
    const width = Math.round(NUB_WIDTH + (BADGE_WIDTH - NUB_WIDTH) * popProgress)
    const badgeOpacity = Math.round(SHRUNK_OPACITY + (opacity - SHRUNK_OPACITY) * popProgress)
    WindowSetSize(width, SHRUNK_HEIGHT)
    WindowSetPosition(Math.max(0, shrunkRightEdgeX - width), shrunkY)
    SetWindowOpacity(badgeOpacity)
  }

  // Animates popProgress toward 0 (nub) or 1 (badge) with an ease-out curve.
  // Reads from whatever popProgress currently is, so reversing direction
  // mid-animation (e.g. the pointer leaves before the pop-out finishes)
  // continues smoothly from there instead of jumping.
  function animatePopTo(target: number) {
    if (popAnimFrame !== null) cancelAnimationFrame(popAnimFrame)
    const start = popProgress
    const startTime = performance.now()
    const step = (now: number) => {
      const t = Math.min(1, (now - startTime) / POP_ANIM_MS)
      const eased = 1 - Math.pow(1 - t, 3)
      popProgress = start + (target - start) * eased
      applyPopProgress()
      popAnimFrame = t < 1 ? requestAnimationFrame(step) : null
    }
    popAnimFrame = requestAnimationFrame(step)
  }

  onMount(() => {
    ;({ alwaysOnTop, opacity, shrinkOnBlur } = loadPersisted(WINDOW_KEY, {
      alwaysOnTop: false,
      opacity: 100,
      shrinkOnBlur: false,
    }))
    SetAlwaysOnTop(alwaysOnTop)
    SetWindowOpacity(opacity)
  })

  function toggleAlwaysOnTop() {
    alwaysOnTop = !alwaysOnTop
    SetAlwaysOnTop(alwaysOnTop)
    savePersisted(WINDOW_KEY, { alwaysOnTop, opacity, shrinkOnBlur })
    if (!alwaysOnTop && isShrunk) restoreFromShrink()
  }

  function toggleOpacityPopover() {
    opacityPopoverOpen = !opacityPopoverOpen
    aboutPopoverOpen = false
  }

  // Applies opacity live as the slider is dragged (on:input fires on every
  // tick); commitOpacity below only persists once the drag settles, so
  // dragging through several values doesn't spam localStorage writes.
  function applyOpacity() {
    SetWindowOpacity(opacity)
  }

  function commitOpacity() {
    SetWindowOpacity(opacity)
    savePersisted(WINDOW_KEY, { alwaysOnTop, opacity, shrinkOnBlur })
  }

  function toggleShrinkOnBlur() {
    savePersisted(WINDOW_KEY, { alwaysOnTop, opacity, shrinkOnBlur })
    if (!shrinkOnBlur && isShrunk) restoreFromShrink()
  }

  // Native window blur fires when the OS moves focus to another window —
  // reliable here because Wails hosts the whole UI in one real OS window,
  // unlike a browser tab where "blur" can mean other things.
  async function onWindowBlur() {
    if (!alwaysOnTop || !shrinkOnBlur || isShrunk) return
    isShrunk = true
    isPoppedOut = false
    popProgress = 0
    const [pos, size] = await Promise.all([WindowGetPosition(), WindowGetSize()])
    savedGeometry = { x: pos.x, y: pos.y, w: size.w, h: size.h }

    // WindowSetPosition is relative to the monitor the window is currently
    // on, so the nub only needs that monitor's own width/height, not its
    // absolute desktop offset.
    const screens = await ScreenGetAll()
    const screen = screens.find((s) => s.isCurrent) ?? screens.find((s) => s.isPrimary) ?? screens[0]
    const screenW = screen?.width ?? size.w
    shrunkRightEdgeX = screenW - SHRUNK_MARGIN
    shrunkY = Math.max(0, ((screen?.height ?? size.h) - SHRUNK_HEIGHT) / 2)

    // Drop the OS min-size floor so the nub/badge widths aren't clamped
    // wider than requested; restoreFromShrink puts it back.
    WindowSetMinSize(NUB_WIDTH, SHRUNK_HEIGHT)
    applyPopProgress()
  }

  // Pops the nub out to the bigger badge width on hover. Cancels any
  // pending shrink-back from a previous mouseleave.
  function revealPopOut() {
    if (!isShrunk) return
    if (hideShrunkTimer) {
      clearTimeout(hideShrunkTimer)
      hideShrunkTimer = null
    }
    if (isPoppedOut) return
    isPoppedOut = true
    animatePopTo(1)
  }

  // Shrinks back to the nub width after a short delay, so briefly crossing
  // the pointer off it (e.g. moving toward its edge) doesn't snap it shut.
  function scheduleShrinkBack() {
    if (!isShrunk) return
    if (hideShrunkTimer) clearTimeout(hideShrunkTimer)
    hideShrunkTimer = setTimeout(() => {
      hideShrunkTimer = null
      isPoppedOut = false
      animatePopTo(0)
    }, HIDE_DELAY_MS)
  }

  function restoreFromShrink() {
    if (!isShrunk) return
    if (hideShrunkTimer) {
      clearTimeout(hideShrunkTimer)
      hideShrunkTimer = null
    }
    if (popAnimFrame !== null) {
      cancelAnimationFrame(popAnimFrame)
      popAnimFrame = null
    }
    isShrunk = false
    isPoppedOut = false
    // Undo the shrunk-mode min-size floor (0 = no minimum, as configured).
    WindowSetMinSize(0, 0)
    if (savedGeometry) {
      // Growing back to the saved (larger) size, so reposition first — a
      // resize keeps the top-left corner fixed and grows rightward/downward,
      // so sizing up before moving would briefly balloon the window past
      // the screen from its right-edge-hugging shrunk position.
      WindowSetPosition(savedGeometry.x, savedGeometry.y)
      WindowSetSize(savedGeometry.w, savedGeometry.h)
    }
    savedGeometry = null
    SetWindowOpacity(opacity)
  }

  function onWindowClick(e: MouseEvent) {
    if (opacityPopoverOpen && opacityControlEl && !opacityControlEl.contains(e.target as Node)) {
      opacityPopoverOpen = false
    }
    if (aboutPopoverOpen && aboutControlEl && !aboutControlEl.contains(e.target as Node)) {
      aboutPopoverOpen = false
    }
  }

  // --- About popover ---
  const GITHUB_URL = 'https://github.com/mko88/script-manager'

  let appVersion = ''
  let aboutPopoverOpen = false
  let aboutControlEl: HTMLElement

  onMount(async () => {
    appVersion = await GetVersion()
  })

  function toggleAboutPopover() {
    aboutPopoverOpen = !aboutPopoverOpen
    opacityPopoverOpen = false
  }

  function openGithub() {
    BrowserOpenURL(GITHUB_URL)
  }

  // --- Resizable / collapsible panel layout (geometry: lib/panelLayout) ---
  const LAYOUT_KEY = 'script-manager-gui:layout'

  let shellEl: HTMLElement
  let colLeftEl: HTMLElement
  let colRightEl: HTMLElement

  let leftWidth = 320
  let itemsHeight = 340
  let detailsHeight = 420

  let itemsCollapsed = false
  let actionsCollapsed = false
  let detailsCollapsed = false
  let commandCollapsed = false
  let groupChipsCollapsed = true
  let detailsWarningCollapsed = true
  let cmdSectionCollapsed = false
  let outputSectionCollapsed = false

  onMount(() => {
    // Defaults here are the effective first-run values, not necessarily the
    // `let` initializers above (e.g. group chips start expanded on a fresh
    // profile, matching the pre-loadPersisted `!!saved.groupChipsCollapsed`
    // coercion this replaced).
    ;({
      leftWidth,
      itemsHeight,
      detailsHeight,
      itemsCollapsed,
      actionsCollapsed,
      detailsCollapsed,
      commandCollapsed,
      groupChipsCollapsed,
      detailsWarningCollapsed,
      cmdSectionCollapsed,
      outputSectionCollapsed,
    } = loadPersisted(LAYOUT_KEY, {
      leftWidth: 320,
      itemsHeight: 340,
      detailsHeight: 420,
      itemsCollapsed: false,
      actionsCollapsed: false,
      detailsCollapsed: false,
      commandCollapsed: false,
      groupChipsCollapsed: false,
      detailsWarningCollapsed: true,
      cmdSectionCollapsed: false,
      outputSectionCollapsed: false,
    }))
  })

  function saveLayout() {
    savePersisted(LAYOUT_KEY, {
      leftWidth,
      itemsHeight,
      detailsHeight,
      itemsCollapsed,
      actionsCollapsed,
      detailsCollapsed,
      commandCollapsed,
      groupChipsCollapsed,
      detailsWarningCollapsed,
      cmdSectionCollapsed,
      outputSectionCollapsed,
    })
  }

  // The drag/flex geometry lives in lib/panelLayout — these wrappers just
  // bind it to this window's panels and persist the result.
  function dragLeftColumn(e: MouseEvent) {
    dragColumn(e, {
      getTotal: () => shellEl.getBoundingClientRect().width,
      get: () => leftWidth,
      set: (v) => (leftWidth = v),
      onDone: saveLayout,
    })
  }

  function dragItemsRow(e: MouseEvent) {
    if (itemsCollapsed || actionsCollapsed) return
    dragRow(e, {
      getTotal: () => colLeftEl.getBoundingClientRect().height,
      get: () => itemsHeight,
      set: (v) => (itemsHeight = v),
      onDone: saveLayout,
    })
  }

  function dragDetailsRow(e: MouseEvent) {
    if (detailsCollapsed || commandCollapsed) return
    dragRow(e, {
      getTotal: () => colRightEl.getBoundingClientRect().height,
      get: () => detailsHeight,
      set: (v) => (detailsHeight = v),
      onDone: saveLayout,
    })
  }

</script>

<svelte:window on:keydown={onKeyDown} on:click={onWindowClick} on:blur={onWindowBlur} />

{#if isShrunk}
  <button
    class="shrunk-widget"
    on:click={restoreFromShrink}
    on:mouseenter={revealPopOut}
    on:mouseleave={scheduleShrinkBack}
    title={t('tooltip.restoreWindow')}
  >
    <Icon name="restore" />
  </button>
{:else}
<div class="app-root">
  <header class="toolbar">
    <IconButton title={t('tooltip.loadConfig')} on:click={browseConfig}><Icon name="load" /></IconButton>
    <IconButton title={t('tooltip.refreshConfigTitle')} aria={t('tooltip.refreshConfigAria')} on:click={reloadConfig}><Icon name="refresh" /></IconButton>
    <IconButton
      class="btn icon-btn toolbar-right-start"
      title={t('tooltip.pinWindow')}
      active={alwaysOnTop}
      on:click={toggleAlwaysOnTop}><Icon name="pin" /></IconButton
    >
    <div class="opacity-control" bind:this={opacityControlEl}>
      <IconButton
        title={t('tooltip.windowTransparency')}
        active={opacity < 100}
        on:click={toggleOpacityPopover}><Icon name="transparency" /></IconButton
      >
      {#if opacityPopoverOpen}
        <div class="opacity-popover">
          <div class="opacity-slider-row">
            <input
              type="range"
              min={MIN_OPACITY}
              max="100"
              step="5"
              bind:value={opacity}
              on:input={applyOpacity}
              on:change={commitOpacity}
              aria-label={t('tooltip.opacityLevel')}
            />
            <span class="opacity-value">{opacity}%</span>
          </div>
          <label class="shrink-option" title={alwaysOnTop ? '' : t('tooltip.shrinkOnBlurRequiresPin')}>
            <input type="checkbox" bind:checked={shrinkOnBlur} disabled={!alwaysOnTop} on:change={toggleShrinkOnBlur} />
            {t('option.shrinkOnBlur')}
          </label>
        </div>
      {/if}
    </div>
    <div class="about-control" bind:this={aboutControlEl}>
      <IconButton title={t('tooltip.aboutButton')} on:click={toggleAboutPopover}><Icon name="info" /></IconButton>
      {#if aboutPopoverOpen}
        <div class="about-popover">
          <div class="about-title">{t('about.title')}</div>
          <div class="about-version">{t('about.version', { version: appVersion })}</div>
          <p class="about-description">{t('about.description')}</p>
          <a class="about-github-link" href={GITHUB_URL} on:click|preventDefault={openGithub}>{t('about.githubLink')}</a>
        </div>
      {/if}
    </div>
    <IconButton
      class="btn icon-btn settings-btn"
      title={t('tooltip.openConfigEditorTitle')}
      aria={t('tooltip.openConfigEditorAria')}
      on:click={launchConfigEditor}><Icon name="settings" /></IconButton
    >
    <div class="window-controls">
      <IconButton title={t('tooltip.minimizeWindow')} on:click={() => WindowMinimise()}><Icon name="minimize" /></IconButton>
      <IconButton title={t('tooltip.maximizeWindow')} on:click={() => WindowToggleMaximise()}><Icon name="maximize" /></IconButton>
      <IconButton class="btn icon-btn window-close-btn" title={t('tooltip.closeWindow')} on:click={() => Quit()}><Icon name="cancel" /></IconButton>
    </div>
  </header>
  <main class="app-shell" bind:this={shellEl}>
  <div class="col col-left" style="flex: 0 0 {leftWidth}px" bind:this={colLeftEl}>
    <Panel
      bind:collapsed={itemsCollapsed}
      title={t('panel.items')}
      titleWrap={itemsCollapsed}
      expandTitle={t('tooltip.expand')}
      collapseTitle={t('tooltip.collapse')}
      onToggle={saveLayout}
      style={topStyle(itemsCollapsed, actionsCollapsed, itemsHeight, true)}
      class="panel-items"
    >
      <svelte:fragment slot="title-extra">
        {#if itemsCollapsed && selectedItemLabel}<span class="panel-title-selected">{t('text.separator')}{selectedItemLabel}</span>{/if}
      </svelte:fragment>
      <div class="panel-body list">
        {#each items as item (item.index)}
          <button
            class="row"
            class:selected={item.index === selectedItem}
            on:click={() => selectItem(item.index)}
          >{item.label}{#if runningItemIndices.has(item.index)}<span class="running-indicator" title={t('tooltip.actionRunningItem')}>●</span>{/if}</button>
        {/each}
        {#if items.length === 0}
          <div class="empty">{t('empty.noItems')}</div>
        {/if}
      </div>
    </Panel>

    <!-- svelte-ignore a11y-no-static-element-interactions -->
    <div class="resizer horizontal" class:disabled={itemsCollapsed || actionsCollapsed} on:mousedown={dragItemsRow}></div>

    <Panel
      bind:collapsed={actionsCollapsed}
      title={t('panel.actions')}
      titleWrap={actionsCollapsed}
      expandTitle={t('tooltip.expand')}
      collapseTitle={t('tooltip.collapse')}
      onToggle={saveLayout}
      style={bottomStyle(actionsCollapsed, true)}
      class="panel-actions"
    >
      <svelte:fragment slot="title-extra">
        {#if actionsCollapsed && selectedActionLabel}<span class="panel-title-selected">{t('text.separator')}{selectedActionLabel}</span>{/if}
      </svelte:fragment>
      <GroupFilter
        {actions}
        {groupColors}
        bind:selectedGroups
        bind:collapsed={groupChipsCollapsed}
        onCollapseChange={saveLayout}
        onSelectionChange={onGroupFilterChange}
      />
      <div class="panel-body list">
        {#each filteredActions as action (action.index)}
          <button
            class="row"
            class:selected={action.index === selectedActionIndex}
            on:click={() => selectAction(action.index)}
          >{action.title}{#if runningActionIndicesForSelectedItem.has(action.index)}<span class="running-indicator" title={t('tooltip.actionRunningAction')}>●</span>{:else if lastExitCodeByActionForSelectedItem.has(action.index)}<span
              class="exit-indicator"
              class:status-ok={lastExitCodeByActionForSelectedItem.get(action.index) === 0}
              class:status-fail={lastExitCodeByActionForSelectedItem.get(action.index) !== 0}
              title={t('tooltip.actionLastExitCode', { code: String(lastExitCodeByActionForSelectedItem.get(action.index)) })}>●</span
            >{/if}</button>
        {/each}
        {#if selectedItem >= 0 && filteredActions.length === 0}
          <div class="empty">
            {selectedGroups.size > 0
              ? t('empty.noActionsForGroups', { plural: selectedGroups.size > 1 ? 's' : '' })
              : t('empty.noActionsForItem')}
          </div>
        {/if}
      </div>
    </Panel>
  </div>

  <!-- svelte-ignore a11y-no-static-element-interactions -->
  <div class="resizer vertical" on:mousedown={dragLeftColumn}></div>

  <div class="col col-right" bind:this={colRightEl}>
    <Panel
      bind:collapsed={detailsCollapsed}
      title={t('panel.details')}
      expandTitle={t('tooltip.expand')}
      collapseTitle={t('tooltip.collapse')}
      onToggle={saveLayout}
      style={topStyle(detailsCollapsed, commandCollapsed, detailsHeight)}
      class="panel-details"
    >
      {#if missingFields.length > 0}
        <div class="details-warning">
          <div class="details-warning-header">
            <CollapseToggle
              bind:collapsed={detailsWarningCollapsed}
              onToggle={saveLayout}
              class="warning-toggle"
              expandTitle={t('tooltip.expandMissingWarning')}
              collapseTitle={t('tooltip.collapseMissingWarning')}
            />
            <span class="warning-summary">
              {t('warning.missingFields', { count: missingFields.length, plural: missingFields.length > 1 ? 's' : '' })}
            </span>
          </div>
          {#if !detailsWarningCollapsed}
            <div class="warning-chips">
              {#each missingFields as field (field)}
                <span class="chip chip-static warning-chip">{field}</span>
              {/each}
            </div>
          {/if}
        </div>
      {/if}
      <!-- svelte-ignore a11y-click-events-have-key-events -->
      <!-- svelte-ignore a11y-no-static-element-interactions -->
      <div class="panel-body details-content" on:click={onDetailsClick}>
        {#if details?.html}
          {@html details.html}
        {:else}
          <div class="empty">{t('empty.noItemSelected')}</div>
        {/if}
      </div>
    </Panel>

    <!-- svelte-ignore a11y-no-static-element-interactions -->
    <div class="resizer horizontal" class:disabled={detailsCollapsed || commandCollapsed} on:mousedown={dragDetailsRow}></div>

    <Panel
      bind:collapsed={commandCollapsed}
      title={t('panel.command')}
      expandTitle={t('tooltip.expand')}
      collapseTitle={t('tooltip.collapse')}
      onToggle={saveLayout}
      style={bottomStyle(commandCollapsed)}
      class="panel-command"
    >
        <div class="panel-body command-content">
          {#if actionDetail}
            {#if actionDetail.cmd || actionDetail.script}
              <div class="cmd-actions">
                {#if !actionDetail.interactive}
                  <IconButton
                    class="run-cmd-btn icon-btn"
                    title={t('tooltip.runHere')}
                    disabled={inlineRunning}
                    on:click={runActionInline}><Icon name="run-here" /></IconButton
                  >
                {/if}
                <IconButton class="run-cmd-btn icon-btn" title={t('tooltip.run')} on:click={runAction}><Icon name="run" /></IconButton>
                {#if inlineRunning}
                  <IconButton class="copy-cmd-btn icon-btn" title={t('tooltip.cancel')} on:click={cancelInlineAction}><Icon name="cancel" /></IconButton>
                {/if}
              </div>
            {/if}
            {#if inlineRunning || inlineOutput || inlineExitCode !== null}
              <div class="messages-group">
                <button class="messages-group-header" type="button" on:click={() => { outputSectionCollapsed = !outputSectionCollapsed; saveLayout() }}>
                  <span class="messages-group-title">{t('section.output')}</span>
                  <span class="output-status">
                    {#if inlineRunning}
                      {t('text.running')}<span class="status-dot status-running">●</span>
                    {:else if inlineExitCode !== null}
                      {t('text.exitCode', { code: String(inlineExitCode) })}<span
                        class="status-dot"
                        class:status-ok={inlineExitCode === 0}
                        class:status-fail={inlineExitCode !== 0}>●</span
                      >
                    {/if}
                  </span>
                  <span class="collapse-glyph">{outputSectionCollapsed ? '▸' : '▾'}</span>
                </button>
                {#if !outputSectionCollapsed && inlineOutput}
                  <div class="cmd-output">
                    <IconButton
                      class="cmd-copy-btn cmd-output-copy-btn"
                      title={t('tooltip.copyOutput')}
                      on:click={() => copyToClipboard(inlineOutput)}><Icon name="copy" /></IconButton
                    >
                    <pre class="cmd-output-body" bind:this={inlineOutputEl}>{inlineOutput}</pre>
                  </div>
                {/if}
              </div>
            {/if}
            <div class="messages-group">
              <button class="messages-group-header" type="button" on:click={() => { cmdSectionCollapsed = !cmdSectionCollapsed; saveLayout() }}>
                <span class="messages-group-title">{t('section.command')}</span>
                <span class="collapse-glyph">{cmdSectionCollapsed ? '▸' : '▾'}</span>
              </button>
              {#if !cmdSectionCollapsed}
                {#if actionDetail.description}
                  <p class="cmd-desc">{actionDetail.description}</p>
                {/if}
                {#if selectedActionGroups.length > 0}
                  <div class="cmd-groups">
                    {#each selectedActionGroups as group (group)}
                      <span class="chip chip-static" style={groupChipStyle(groupColors, group, false)}>{group}</span>
                    {/each}
                  </div>
                {/if}
                {#if actionDetail.script}
                  <p class="cmd-desc">{t('text.scriptLabel')}{actionDetail.script}</p>
                  {#if actionDetail.scriptError}
                    <p class="cmd-error">{actionDetail.scriptError}</p>
                  {:else}
                    <ScriptSource content={actionDetail.scriptContent}>
                      <IconButton class="cmd-copy-btn cmd-line-copy-btn" title={t('tooltip.copyCommand')} on:click={copyCmd}><Icon name="copy" /></IconButton>
                    </ScriptSource>
                  {/if}
                {:else if actionDetail.cmd}
                  <ScriptSource content={actionDetail.cmd}>
                    <IconButton class="cmd-copy-btn cmd-line-copy-btn" title={t('tooltip.copyCommand')} on:click={copyCmd}><Icon name="copy" /></IconButton>
                  </ScriptSource>
                {/if}
              {/if}
            </div>
          {:else}
            <div class="empty">{t('empty.selectActionToPreview')}</div>
          {/if}
        </div>
    </Panel>
  </div>

    <Toast />
  </main>
</div>
{/if}

<style>
  .app-root {
    display: flex;
    flex-direction: column;
    height: 100vh;
  }

  .toolbar {
    /* The window is frameless (main.go) — this bar is its only drag
       handle. --wails-draggable is a CSS custom property Wails itself
       watches for (not app-defined), and it inherits to every descendant
       by default, so every clickable child below has to opt back out with
       --wails-draggable: no-drag or it'd drag the window instead of
       responding to clicks. */
    --wails-draggable: drag;
    flex: none;
    display: flex;
    align-items: center;
    gap: 6px;
    padding: 6px 8px;
    background: var(--sm-panel-header);
    border-bottom: 1px solid var(--sm-border);
  }

  /* .icon-btn comes from the shared design system (@shared/theme.css),
     same as .btn. */

  /* Opts every toolbar button (and the opacity/about popovers, which nest
     a range input and a link respectively) back out of the .toolbar drag
     region above — otherwise clicking any of them would drag the window
     instead of activating it. */
  .toolbar :global(.btn),
  .toolbar .opacity-control,
  .toolbar .about-control {
    --wails-draggable: no-drag;
  }

  /* Takes over the slot the theme dropdown used to occupy at the far
     right of the toolbar: pushes the pin/transparency/settings group as a
     whole to the right edge, leaving load/refresh at the left. */
  /* :global — this class now renders inside IconButton's own template
     (via its class prop), which Svelte's per-component CSS scoping
     wouldn't otherwise reach. */
  :global(.toolbar-right-start) {
    margin-left: auto;
  }

  /* Highlights the pin/transparency buttons while toggled on — .toolbar
     isn't .list-toolbar, so it doesn't already get that shared rule's
     .btn.active styling. */
  .toolbar :global(.btn.active) {
    background: var(--sm-bg-primary);
    border-color: var(--sm-bg-primary);
    color: var(--sm-text-primary);
  }

  .opacity-control {
    position: relative;
    display: flex;
  }

  .opacity-popover {
    position: absolute;
    top: calc(100% + 4px);
    right: 0;
    display: flex;
    flex-direction: column;
    gap: 8px;
    padding: 8px 10px;
    background: var(--sm-panel-header);
    border: 1px solid var(--sm-border);
    border-radius: 6px;
    box-shadow: 0 4px 12px var(--sm-shadow);
    z-index: 20;
  }

  .opacity-slider-row {
    display: flex;
    align-items: center;
    gap: 8px;
  }

  .opacity-popover input[type='range'] {
    accent-color: var(--sm-text-heading);
  }

  .opacity-value {
    font-size: 0.8rem;
    color: var(--sm-text-muted);
    min-width: 2.4em;
    text-align: right;
  }

  .shrink-option {
    display: flex;
    align-items: center;
    gap: 6px;
    font-size: 0.8rem;
    color: var(--sm-text-muted);
    white-space: nowrap;
  }

  .shrink-option:has(input:disabled) {
    opacity: 0.6;
  }

  /* The shrunk-on-blur nub/badge: at this point the native window itself
     has been animated between NUB_WIDTH and BADGE_WIDTH (always at
     SHRUNK_HEIGHT — see applyPopProgress/animatePopTo), so this replaces
     .app-root entirely rather than overlaying it. box-sizing: border-box
     keeps the 1px border inside those dimensions rather than added on top
     of them, which would otherwise push the box a couple of pixels past
     the window's actual bounds and throw off the icon's centering. */
  .shrunk-widget {
    display: flex;
    align-items: center;
    justify-content: center;
    box-sizing: border-box;
    width: 100vw;
    height: 100vh;
    margin: 0;
    padding: 0;
    background: var(--sm-panel-header);
    border: 1px solid var(--sm-border);
    color: var(--sm-text-heading);
    font: inherit;
    cursor: pointer;
  }

  .shrunk-widget :global(svg) {
    width: 28px;
    height: 28px;
  }

  .about-control {
    position: relative;
    display: flex;
  }

  .about-popover {
    position: absolute;
    top: calc(100% + 4px);
    right: 0;
    display: flex;
    flex-direction: column;
    gap: 4px;
    width: 220px;
    padding: 10px 12px;
    background: var(--sm-panel-header);
    border: 1px solid var(--sm-border);
    border-radius: 6px;
    box-shadow: 0 4px 12px var(--sm-shadow);
    z-index: 20;
  }

  .about-title {
    font-weight: 700;
    color: var(--sm-text-heading);
  }

  .about-version {
    font-size: 0.75rem;
    color: var(--sm-text-muted);
  }

  .about-description {
    margin: 4px 0 0;
    font-size: 0.8rem;
    color: var(--sm-text);
    line-height: 1.4;
  }

  .about-github-link {
    margin-top: 4px;
    color: var(--sm-text-heading);
    font-size: 0.8rem;
  }

  /* Stands in for the native title-bar buttons the frameless window no
     longer has — tighter gap than the rest of the toolbar, matching the
     conventional Windows minimize/maximize/close grouping. */
  .window-controls {
    display: flex;
    gap: 2px;
    margin-left: 8px;
  }

  :global(.window-close-btn:hover) {
    background: var(--sm-error);
    border-color: var(--sm-error);
    color: #fff;
  }

  .app-shell {
    display: flex;
    flex-direction: row;
    flex: 1 1 auto;
    min-height: 0;
    box-sizing: border-box;
    padding: 8px;
    text-align: left;
  }

  .col {
    display: flex;
    flex-direction: column;
    min-height: 0;
    min-width: 0;
  }

  .col-right {
    flex: 1 1 auto;
  }

  /* .resizer(.vertical/.horizontal/.disabled), .panel, .panel-title(-text/-selected), .collapse-btn, .panel-body, .list,
     .row, .chip, .empty, .toast, .copy-cmd-btn, .run-cmd-btn come from the
     shared design system (@shared/theme.css, imported via style.css) — not
     redefined here. */

  .details-warning {
    flex: none;
    display: flex;
    flex-direction: column;
    gap: 4px;
    padding: 4px 6px;
    background: rgba(232, 163, 61, 0.1);
    border-bottom: 1px solid var(--sm-border);
  }

  .details-warning-header {
    display: flex;
    align-items: flex-start;
    gap: 4px;
  }

  /* :global, not scoped — the button this styles now renders inside
     CollapseToggle's own template (a different component), which
     Svelte's per-component CSS scoping wouldn't otherwise reach. */
  :global(.warning-toggle) {
    flex: none;
    padding: 2px 4px;
    color: var(--sm-warning);
  }

  .warning-summary {
    color: var(--sm-warning);
    font-size: 0.78rem;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .warning-chips {
    display: flex;
    flex-wrap: wrap;
    gap: 4px;
    padding-bottom: 2px;
  }

  .warning-chip {
    border-color: var(--sm-warning);
    color: var(--sm-warning);
  }

  .details-content {
    font-size: 0.9rem;
    line-height: 1.5;
  }

  .details-content :global(h1),
  .details-content :global(h2),
  .details-content :global(h3) {
    color: var(--sm-text-heading);
    margin: 0.6em 0 0.3em;
  }

  .details-content :global(table) {
    border-collapse: collapse;
    width: 100%;
  }

  .details-content :global(td),
  .details-content :global(th) {
    border: 1px solid var(--sm-border);
    padding: 4px 8px;
    text-align: left;
  }

  .details-content :global(code) {
    background: var(--sm-bg-deep);
    color: var(--sm-text-highlight);
    padding: 1px 5px;
    border-radius: 3px;
    font-family: "SF Mono", Consolas, monospace;
  }

  .details-content :global(code.copy-value) {
    cursor: pointer;
  }

  .details-content :global(code.copy-value:hover) {
    background: var(--sm-tint-hover);
    outline: 1px solid var(--sm-text-highlight);
  }

  .details-content :global(code.copy-value-masked) {
    color: var(--sm-masked);
  }

  .command-content {
    font-size: 0.85rem;
  }

  .cmd-desc {
    margin: 0 0 8px;
    color: var(--sm-text-muted);
    white-space: pre-wrap;
  }

  .cmd-error {
    margin: 0 0 8px;
    color: var(--sm-error);
  }

  .cmd-output {
    position: relative;
    background: var(--sm-bg-deep);
    border-radius: 4px;
    margin: 0 0 8px;
  }
  /* :global — these buttons now render inside IconButton's own template
     (via its class prop), which Svelte's per-component CSS scoping
     wouldn't otherwise reach. */
  :global(.cmd-output-copy-btn) {
    position: absolute;
    top: 4px;
    right: 4px;
  }
  /* The list dots (running pulse / last exit code) sit at the row's right
     edge: rows go flex here — the shared .row stays display:block for the
     config editor's use — so margin-left:auto can push the dot over
     regardless of label length. The exit dot's ok/fail colors come from
     the shared .status-ok/.status-fail classes (@shared/theme.css). */
  .list .row {
    display: flex;
    align-items: center;
  }
  .running-indicator,
  .exit-indicator {
    margin-left: auto;
    padding-left: 8px;
    font-size: 1rem;
    line-height: 1;
  }
  .running-indicator {
    color: var(--sm-run-active);
    animation: running-pulse 1.5s ease-in-out infinite;
  }
  /* .output-status, .status-dot, .status-running/.status-ok/.status-fail
     (the OUTPUT section header's run status) come from the shared design
     system (@shared/theme.css) — also reused by the config editor's theme
     preview. */
  @keyframes running-pulse {
    0%, 100% { opacity: 1; }
    50% { opacity: 0.3; }
  }
  .cmd-output-body {
    margin: 0;
    padding: 10px;
    font-family: "SF Mono", Consolas, monospace;
    font-size: 0.8rem;
    white-space: pre-wrap;
    word-break: break-all;
    max-height: 260px;
    overflow-y: auto;
  }

  .cmd-groups {
    display: flex;
    flex-wrap: wrap;
    gap: 4px;
    margin: 0 0 8px;
  }

  /* Minimal, borderless copy button meant to sit inside a code block —
     .cmd-line-copy-btn and .cmd-output-copy-btn both float it in the
     top-right corner, the placement docs sites commonly use for a code
     block's copy action. .cmd-line-copy-btn positions against
     ScriptSource's own position:relative card (@shared/components/
     ScriptSource.svelte) — it's passed into that component's default slot,
     not rendered as a sibling here, but stays :global() since Svelte scopes
     slotted content to the component that fills the slot (this one), not
     the one that declares it. */
  :global(.cmd-copy-btn) {
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
  :global(.cmd-copy-btn:hover) {
    background: var(--sm-overlay-soft);
    color: var(--sm-text);
  }
  :global(.cmd-line-copy-btn) {
    position: absolute;
    top: 4px;
    right: 4px;
  }

  .cmd-actions {
    position: sticky;
    top: 0;
    z-index: 2;
    display: flex;
    gap: 8px;
    margin: -6px -6px 8px;
    padding: 6px 6px 8px;
    background: var(--sm-bg-alt);
    box-shadow: 0 4px 6px -4px var(--sm-shadow);
  }

</style>
