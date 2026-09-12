<script lang="ts">
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
