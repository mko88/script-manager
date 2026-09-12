<script lang="ts">
  import CollapseToggle from '@shared/components/CollapseToggle.svelte'

  export let collapsed: boolean
  export let title: string
  export let titleWrap = false
  export let expandTitle: string
  export let collapseTitle: string
  export let onToggle: (() => void) | undefined = undefined
  export let style = ''
  let className = ''
  export { className as class }

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
