<script lang="ts">
  export let title: string
  export let aria: string | undefined = undefined
  export let active = false
  export let disabled = false
  let className = 'btn icon-btn'
  export { className as class }
</script>

<button class={className} class:active type="button" {title} aria-label={aria ?? title} {disabled} on:click>
  <slot />
</button>
