<script lang="ts">
  // The standalone ▾/▸ toggle button used by both apps' collapsible
  // panels/sections — shared so a future tweak (a new state, a different
  // glyph) touches one file instead of every panel header that repeats it.
  // collapsed is bindable for the plain case; onToggle additionally fires
  // for callers that need a side effect on every flip (e.g. persisting
  // layout state), so they don't need to reimplement the flip themselves.
  // A plain callback prop, not a dispatched event: frontend-shared sits
  // outside both apps' node_modules, so a bare `svelte` import (needed for
  // createEventDispatcher) doesn't resolve from here — see toast.ts for the
  // same constraint. This also matches how the rest of the codebase already
  // wires parent callbacks into child components (e.g. ItemsEditor's
  // previewItem/validateField props).
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
