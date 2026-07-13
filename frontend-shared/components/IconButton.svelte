<script lang="ts">
  // The "<button class='btn icon-btn' title=... aria-label=... on:click=...>"
  // shape, repeated 35+ times across both apps. aria defaults to title —
  // true at most call sites — but some buttons deliberately diverge (a
  // title with a keyboard-shortcut hint, e.g. "Refresh config (F5)", vs a
  // plain aria-label for screen readers, "Refresh config"), so it's a
  // separate overridable prop, not folded into title. class is a full
  // override, not an append: call sites vary their base class (btn
  // icon-btn, btn btn-primary icon-btn, run-cmd-btn icon-btn, copy-cmd-btn
  // icon-btn).
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
