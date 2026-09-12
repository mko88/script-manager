<script lang="ts">
  // The standalone ▾/▸ toggle button used by both apps' collapsible
  // panels/sections. collapsed is bindable for the plain case; onToggle
  // additionally fires on every flip for callers that need a side effect
  // (e.g. persisting layout state). A plain callback prop, not a dispatched
  // event: frontend-shared sits outside both apps' node_modules, so a bare
  // `svelte` import (needed for createEventDispatcher) doesn't resolve from
  // here — see toast.ts for the same constraint.
  export let collapsed: boolean
  export let expandTitle: string
  export let collapseTitle: string
  export let onToggle: (() => void) | undefined = undefined
  let className = ''
  export { className as class }

  function toggle() {
    collapsed = !collapsed
    onToggle?.()
  }
</script>

<button class="collapse-btn {className}" type="button" on:click={toggle} title={collapsed ? expandTitle : collapseTitle}>
  {collapsed ? '▸' : '▾'}
</button>
