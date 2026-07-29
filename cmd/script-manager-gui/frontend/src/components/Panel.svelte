<script lang="ts">
  import CollapseToggle from '@shared/components/CollapseToggle.svelte'

  // The collapsible/resizable panel header+body shape shared by all four of
  // this app's panels (Items, Actions, Details, Command) — same title bar,
  // collapse toggle, and conditional-body-when-expanded structure, so a
  // future layout tweak touches this one file instead of all four.
  export let collapsed: boolean
  export let title: string
  export let titleWrap = false
  export let expandTitle: string
  export let collapseTitle: string
  export let onToggle: (() => void) | undefined = undefined
  export let style = ''
  let className = ''
  export { className as class }

  // Clicking anywhere on the header toggles collapse, not just the
  // chevron — except a click on the chevron itself, which CollapseToggle
  // below already handles on its own; toggling again here would just flip
  // it straight back.
  function onHeaderClick(e: MouseEvent) {
    if ((e.target as HTMLElement).closest('.collapse-btn')) return
    collapsed = !collapsed
    onToggle?.()
  }
</script>

<section class="panel {className}" {style}>
  <!-- svelte-ignore a11y-click-events-have-key-events -->
  <!-- svelte-ignore a11y-no-static-element-interactions -->
  <header class="panel-title" on:click={onHeaderClick}>
    <span class="panel-title-text" class:wrap={titleWrap}>
      {title}<slot name="title-extra" />
    </span>
    <CollapseToggle bind:collapsed {expandTitle} {collapseTitle} {onToggle} />
  </header>
  {#if !collapsed}
    <slot />
  {/if}
</section>

<style>
  .panel-title {
    cursor: pointer;
  }
</style>
