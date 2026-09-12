<script lang="ts">
  import { onMount } from 'svelte'
  import Toast from '@shared/components/Toast.svelte'
  import { flash } from '@shared/toast'
  import { loadPersisted, savePersisted } from '@shared/persist'
  import { watchTheme } from '@shared/theme'
  import { applyUIPrefs, clampScale, scaleFromKeydown, UI_SCALE } from '@shared/uiprefs'
  import Icon from '@shared/components/Icon.svelte'
  import CollapseToggle from '@shared/components/CollapseToggle.svelte'
  import IconButton from '@shared/components/IconButton.svelte'
  import RecentMenu from '@shared/components/RecentMenu.svelte'
  import PinDialog from '@shared/components/PinDialog.svelte'
  import CodeMirror from '@shared/components/CodeMirror.svelte'
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
    RecentConfigs,
    GetUIPrefs,
    SetUIScale,
    LoadRecentConfig,
    ClearRecentConfigs,
    ActionNeedsUnlock,
    OpenScriptInEditor,
    WatchScript,
    UnlockSecrets,
  } from '../wailsjs/go/gui/App.js'
  import type { gui } from '../wailsjs/go/models'

  let items: gui.ItemDTO[] = []
  let actions: gui.ActionDTO[] = []
  let actionGroupCatalog: gui.ActionGroupDTO[] = []
  let details: gui.DetailsDTO | null = null
  let actionDetail: gui.ActionDetailDTO | null = null

  let selectedItem = -1
  let selectedActionIndex = -1
  let selectedGroups = new Set<string>()

  onMount(() => watchTheme(EventsOn, () => {}))
  onMount(() => EventsOn('config:changed', onConfigFileChanged))
  onMount(() => EventsOn('script:changed', reloadActionDetail))

  // Follows whatever script the Command pane is showing, so an edit made
  // outside the app — including from the button beside the path — refreshes
  // it in place.
  $: WatchScript(actionDetail?.script ?? '')

  async function reloadActionDetail() {
    if (selectedItem < 0 || selectedActionIndex < 0) return
    actionDetail = await GetActionDetail(selectedItem, selectedActionIndex)
  }

  let uiScale: number = UI_SCALE.default

  onMount(async () => {
    const prefs = await GetUIPrefs()
    uiScale = clampScale(prefs.scalePercent)
    applyUIPrefs(prefs)
  })

  async function onScaleKeydown(e: KeyboardEvent) {
    const next = scaleFromKeydown(e, uiScale)
    if (next === null) return
    e.preventDefault()
    if (next === uiScale) return
    uiScale = next
    applyUIPrefs(await SetUIScale(next))
  }

  $: currentInline = selectedItem >= 0 && selectedActionIndex >= 0 ? $inlineStates[inlineKey(selectedItem, selectedActionIndex)] : undefined
  $: inlineRunning = currentInline?.running ?? false
  $: inlineOutput = currentInline?.output ?? ''
  $: inlineExitCode = currentInline?.exitCode ?? null

  $: runningItemIndices = new Set(Object.values($inlineStates).filter((s) => s.running).map((s) => s.itemIndex))
  $: runningActionIndicesForSelectedItem = new Set(
    Object.values($inlineStates)
      .filter((s) => s.running && s.itemIndex === selectedItem)
      .map((s) => s.actionIndex),
  )
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

  // The OUTPUT section belongs to inline running, so it is shown for any
  // action that can run inline — before the first run too, where it stands
  // collapsed as a reminder that the pane is there.
  $: canRunInline = !!actionDetail && !actionDetail.interactive && !!(actionDetail.cmd || actionDetail.script)
  $: hasInlineOutput = inlineRunning || !!inlineOutput || inlineExitCode !== null
  $: if (actionDetail && !hasInlineOutput) outputSectionCollapsed = true

  $: selectedItemLabel = items.find((i) => i.index === selectedItem)?.label ?? ''
  $: selectedActionLabel = actions.find((a) => a.index === selectedActionIndex)?.title ?? ''
  $: selectedActionGroups = actions.find((a) => a.index === selectedActionIndex)?.groups ?? []

  $: groupColors = buildGroupColors(actionGroupCatalog)

  onMount(async () => {
    const loadErr = await LoadError()
    if (loadErr) flash(t('toast.configLoadFailed', { error: loadErr }))
    items = await GetItems()
    actionGroupCatalog = await GetActionGroups()
    await refreshRecents()
    if (items.length > 0) selectItem(0)
  })

  async function selectItem(index: number) {
    selectedItem = index
    selectedActionIndex = -1
    selectedGroups = new Set()
    actionDetail = null
    detailsCollapsed = false
    commandCollapsed = true
    saveLayout()
    actions = await GetActions(index)
    details = await GetItemDetails(index)
  }

  function onGroupFilterChange() {
    selectedActionIndex = -1
    actionDetail = null
  }

  async function selectAction(index: number) {
    if (selectedItem < 0) return
    selectedActionIndex = index
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

  // What the pane is showing: a script's contents when they could be read,
  // the command otherwise, and the path only when the file wouldn't open —
  // where the path is the useful thing to have.
  async function openScriptInEditor() {
    if (!actionDetail?.script) return
    try {
      await OpenScriptInEditor(actionDetail.script)
    } catch (err) {
      flash(t('toast.openScriptFailed', { error: String(err) }))
    }
  }

  function copyCmd() {
    const value = actionDetail?.scriptContent || actionDetail?.cmd || actionDetail?.script
    if (!value) return
    copyToClipboard(value)
  }

  let pinDialogOpen = false
  let pinError = ''
  let pinPending: (() => void) | null = null

  async function withUnlocked(run: () => void) {
    if (selectedItem >= 0 && selectedActionIndex >= 0 && (await ActionNeedsUnlock(selectedItem, selectedActionIndex))) {
      pinError = ''
      pinPending = run
      pinDialogOpen = true
      return
    }
    run()
  }

  async function submitPin(pin: string) {
    try {
      await UnlockSecrets(pin)
    } catch {
      pinError = t('tooltip.pinWrong')
      return
    }
    pinDialogOpen = false
    pinError = ''
    const run = pinPending
    pinPending = null
    if (selectedItem >= 0) details = await GetItemDetails(selectedItem)
    run?.()
  }

  function cancelPin() {
    pinDialogOpen = false
    pinError = ''
    pinPending = null
  }

  function runAction() {
    if (selectedItem < 0 || selectedActionIndex < 0) return
    withUnlocked(async () => {
      try {
        await RunAction(selectedItem, selectedActionIndex)
        flash(t('toast.runningInTerminal'))
      } catch (err) {
        flash(t('toast.runFailed', { error: String(err) }))
      }
    })
  }

  function runActionInline() {
    if (selectedItem < 0 || selectedActionIndex < 0) return
    withUnlocked(() => {
      // The output is what you want to watch once it starts, and the command
      // is what you just read to decide to run it.
      cmdSectionCollapsed = true
      outputSectionCollapsed = false
      saveLayout()
      startInlineRun(selectedItem, selectedActionIndex)
    })
  }

  function cancelInlineAction() {
    if (selectedItem < 0 || selectedActionIndex < 0) return
    cancelInlineRun(selectedItem, selectedActionIndex)
  }

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

  async function onConfigFileChanged() {
    let warning = ''
    try {
      warning = await ReloadConfig()
    } catch {
      return
    }
    await refreshAfterConfigChange()
    if (warning) flash(t('toast.configReloadedWithWarning', { warning }))
  }

  async function browseConfig() {
    let path = ''
    try {
      path = await BrowseConfig()
    } catch (err) {
      flash(t('toast.loadFailed', { error: String(err) }))
      return
    }
    if (!path) return
    await refreshAfterConfigChange()
    await refreshRecents()
    flash(t('toast.loaded', { path }))
  }

  let recents: string[] = []

  async function refreshRecents() {
    recents = await RecentConfigs()
  }

  async function loadRecent(path: string) {
    try {
      await LoadRecentConfig(path)
    } catch (err) {
      await refreshRecents()
      flash(t('toast.loadFailed', { error: String(err) }))
      return
    }
    await refreshAfterConfigChange()
    await refreshRecents()
    flash(t('toast.loaded', { path }))
  }

  async function clearRecents() {
    recents = await ClearRecentConfigs()
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

  const WINDOW_KEY = 'script-manager-gui:window'
  const MIN_OPACITY = 20

  let alwaysOnTop = false
  let opacity = 100
  let opacityPopoverOpen = false
  let opacityControlEl: HTMLElement

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
  let popProgress = 0
  let popAnimFrame: number | null = null
  let savedGeometry: { x: number; y: number; w: number; h: number } | null = null

  function applyPopProgress() {
    const width = Math.round(NUB_WIDTH + (BADGE_WIDTH - NUB_WIDTH) * popProgress)
    const badgeOpacity = Math.round(SHRUNK_OPACITY + (opacity - SHRUNK_OPACITY) * popProgress)
    WindowSetSize(width, SHRUNK_HEIGHT)
    WindowSetPosition(Math.max(0, shrunkRightEdgeX - width), shrunkY)
    SetWindowOpacity(badgeOpacity)
  }

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

  async function onWindowBlur() {
    if (!alwaysOnTop || !shrinkOnBlur || isShrunk) return
    isShrunk = true
    isPoppedOut = false
    popProgress = 0
    const [pos, size] = await Promise.all([WindowGetPosition(), WindowGetSize()])
    savedGeometry = { x: pos.x, y: pos.y, w: size.w, h: size.h }

    const screens = await ScreenGetAll()
    const screen = screens.find((s) => s.isCurrent) ?? screens.find((s) => s.isPrimary) ?? screens[0]
    const screenW = screen?.width ?? size.w
    shrunkRightEdgeX = screenW - SHRUNK_MARGIN
    shrunkY = Math.max(0, ((screen?.height ?? size.h) - SHRUNK_HEIGHT) / 2)

    WindowSetMinSize(NUB_WIDTH, SHRUNK_HEIGHT)
    applyPopProgress()
  }

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
    WindowSetMinSize(0, 0)
    if (savedGeometry) {
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

  const GITHUB_URL = 'https://github.com/mko88/script-manager'

  let appVersion = ''
  let aboutPopoverOpen = false
  let aboutControlEl: HTMLElement

  onMount(async () => {
    appVersion = (await GetVersion()).version ?? ''
  })

  function toggleAboutPopover() {
    aboutPopoverOpen = !aboutPopoverOpen
    opacityPopoverOpen = false
  }

  function openGithub() {
    BrowserOpenURL(GITHUB_URL)
  }

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

<svelte:window on:keydown|capture={onScaleKeydown} on:keydown={onKeyDown} on:click={onWindowClick} on:blur={onWindowBlur} />

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
    <RecentMenu
      title={t('tooltip.loadConfig')}
      {recents}
      recentsHeading={t('tooltip.recentHeading')}
      emptyLabel={t('tooltip.recentEmpty')}
      clearLabel={t('tooltip.recentClear')}
      browseLabel={t('tooltip.recentBrowse')}
      onOpen={loadRecent}
      onBrowse={browseConfig}
      onClear={clearRecents}><Icon name="load" /></RecentMenu
    >
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
            <div class="messages-group cmd-section" class:cmd-section-open={!cmdSectionCollapsed}>
              <button class="messages-group-header" type="button" on:click={() => { cmdSectionCollapsed = !cmdSectionCollapsed; saveLayout() }}>
                <span class="messages-group-title">{t('section.command')}</span>
                <span class="collapse-glyph">{cmdSectionCollapsed ? '▸' : '▾'}</span>
              </button>
              {#if !cmdSectionCollapsed}
                <div class="cmd-section-body">
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
                  <p class="cmd-desc script-path-line">
                    <span class="script-path">{t('text.scriptLabel')}{actionDetail.script}</span>
                    <IconButton
                      class="btn icon-btn script-edit-btn"
                      title={t('tooltip.openScriptInEditor', { path: actionDetail.script })}
                      on:click={openScriptInEditor}><Icon name="edit" /></IconButton
                    >
                  </p>
                  {#if actionDetail.scriptError}
                    <p class="cmd-error">{actionDetail.scriptError}</p>
                  {:else}
                    <div class="code-block">
                      <CodeMirror
                        value={actionDetail.scriptContent}
                        language={actionDetail.language}
                        readOnly
                      />
                      <IconButton class="cmd-copy-btn cmd-line-copy-btn" title={t('tooltip.copyCommand')} on:click={copyCmd}><Icon name="copy" /></IconButton>
                    </div>
                  {/if}
                {:else if actionDetail.cmd}
                  <div class="code-block">
                    <CodeMirror value={actionDetail.cmd} language={actionDetail.language} readOnly />
                    <IconButton class="cmd-copy-btn cmd-line-copy-btn" title={t('tooltip.copyCommand')} on:click={copyCmd}><Icon name="copy" /></IconButton>
                  </div>
                {/if}
                </div>
              {/if}
            </div>
            {#if canRunInline}
              <div class="messages-group cmd-section" class:cmd-section-open={!outputSectionCollapsed && inlineOutput}>
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
                {#if !outputSectionCollapsed && !hasInlineOutput}
                  <p class="cmd-desc cmd-output-empty">{t('empty.noOutputYet')}</p>
                {/if}
                {#if !outputSectionCollapsed && inlineOutput}
                  <div class="cmd-output">
                    <IconButton
                      class="cmd-copy-btn cmd-output-copy-btn"
                      title={t('tooltip.copyOutput')}
                      on:click={() => copyToClipboard(inlineOutput)}><Icon name="copy" /></IconButton
                    >
                    <div class="cmd-output-body">
                      <CodeMirror value={inlineOutput} readOnly followTail showLineNumbers={false} maxHeight="100%" />
                    </div>
                  </div>
                {/if}
              </div>
            {/if}
          {:else}
            <div class="empty">{t('empty.selectActionToPreview')}</div>
          {/if}
        </div>
    </Panel>
  </div>

    <Toast />

    <PinDialog
    open={pinDialogOpen}
    title={t('tooltip.pinTitle')}
    message={t('tooltip.pinMessage')}
    pinLabel={t('tooltip.pinLabel')}
    confirmLabel={t('tooltip.pinConfirmButton')}
    cancelLabel={t('tooltip.pinCancelButton')}
    error={pinError}
    onSubmit={submitPin}
    onCancel={cancelPin}
  />
</main>
</div>
{/if}

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
    gap: 6px;
    padding: 6px 8px;
    background: var(--sm-panel-header);
    border-bottom: 1px solid var(--sm-border);
  }

  .toolbar :global(.btn),
  .toolbar .opacity-control,
  .toolbar .about-control {
    --wails-draggable: no-drag;
  }

  :global(.toolbar-right-start) {
    margin-left: auto;
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
    font-size: var(--sm-type-sm);
    color: var(--sm-text-muted);
    min-width: 2.4em;
    text-align: right;
  }

  .shrink-option {
    display: flex;
    align-items: center;
    gap: 6px;
    font-size: var(--sm-type-sm);
    color: var(--sm-text-muted);
    white-space: nowrap;
  }

  .shrink-option:has(input:disabled) {
    opacity: 0.6;
  }

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
    font-size: var(--sm-type-sm);
    color: var(--sm-text-muted);
  }

  .about-description {
    margin: 4px 0 0;
    font-size: var(--sm-type-sm);
    color: var(--sm-text);
    line-height: 1.4;
  }

  .about-github-link {
    margin-top: 4px;
    color: var(--sm-text-heading);
    font-size: var(--sm-type-sm);
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

  .details-warning {
    flex: none;
    display: flex;
    flex-direction: column;
    gap: 4px;
    padding: 4px 6px;
    background: var(--sm-warning-tint);
    border-bottom: 1px solid var(--sm-border);
  }

  .details-warning-header {
    display: flex;
    align-items: flex-start;
    gap: 4px;
  }

  :global(.warning-toggle) {
    flex: none;
    padding: 2px 4px;
    color: var(--sm-warning);
  }

  .warning-summary {
    color: var(--sm-warning);
    font-size: var(--sm-type-sm);
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
    font-size: var(--sm-type-lg);
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
    font-family: var(--sm-font-mono);
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

  .cmd-desc {
    margin: 0 0 8px;
    color: var(--sm-text-muted);
    white-space: pre-wrap;
  }

  .cmd-error {
    margin: 0 0 8px;
    color: var(--sm-error);
  }

  .cmd-output-empty {
    margin: 0;
    padding: 2px 2px 4px;
    font-style: italic;
    color: var(--sm-text-faint);
  }

  .cmd-output {
    position: relative;
    background: var(--sm-bg-deep);
    border-radius: 4px;
    margin: 0;
    flex: 1 1 auto;
    min-height: 0;
    display: flex;
  }
  :global(.cmd-output-copy-btn) {
    position: absolute;
    top: 4px;
    right: 4px;
    z-index: 1;
  }
  .list .row {
    display: flex;
    align-items: center;
  }
  .running-indicator,
  .exit-indicator {
    margin-left: auto;
    padding-left: 8px;
    font-size: var(--sm-type-xl);
    line-height: 1;
  }
  .running-indicator {
    color: var(--sm-run-active);
    animation: running-pulse 1.5s ease-in-out infinite;
  }
  @keyframes running-pulse {
    0%, 100% { opacity: 1; }
    50% { opacity: 0.3; }
  }
  @media (prefers-reduced-motion: reduce) {
    .running-indicator {
      animation: none;
    }
  }
  .cmd-output-body {
    margin: 0;
    padding: 6px;
    flex: 1 1 auto;
    min-height: 0;
    overflow: hidden;
    display: flex;
  }

  /* The pane is a column of sections: the run buttons take what they need,
     and every open section shares what is left rather than the whole pane
     scrolling as one long strip. */
  .command-content {
    font-size: var(--sm-type-base);
    display: flex;
    flex-direction: column;
    gap: 6px;
    min-height: 0;
    overflow: hidden;
  }

  .cmd-actions {
    flex: none;
  }

  .command-content .cmd-section {
    flex: none;
    display: flex;
    flex-direction: column;
    min-height: 0;
    margin: 0;
  }

  /* An open section gets an equal share, but never less than about three
     lines plus its header — below that it is a title bar with a sliver. */
  .command-content .cmd-section-open {
    flex: 1 1 0;
    min-height: 104px;
  }

  .command-content :global(.messages-group-header) {
    flex: none;
  }

  .cmd-section-body {
    flex: 1 1 auto;
    min-height: 0;
    overflow-y: auto;
  }

  .cmd-section-body :global(.sm-code) {
    max-height: none;
  }

  .cmd-output-body :global(.sm-code) {
    flex: 1;
    min-width: 0;
  }

  .code-block {
    position: relative;
    margin-bottom: 8px;
  }

  .script-path-line {
    display: flex;
    align-items: center;
    gap: 6px;
  }

  .script-path {
    min-width: 0;
    overflow-wrap: anywhere;
  }

  :global(.script-edit-btn) {
    flex: none;
    padding: 2px 5px;
  }

  .cmd-groups {
    display: flex;
    flex-wrap: wrap;
    gap: 4px;
    margin: 0 0 8px;
  }

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
