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
</script>

<section class="panel {className}" {style}>
  <header class="panel-title">
    <span class="panel-title-text" class:wrap={titleWrap}>
      {title}<slot name="title-extra" />
    </span>
    <CollapseToggle bind:collapsed {expandTitle} {collapseTitle} {onToggle} />
  </header>
  {#if !collapsed}
    <slot />
  {/if}
</section>
