<script lang="ts">
  import { createEventDispatcher } from 'svelte'

  // Shared by the Terminal section's mode picker and ActionForm's Cmd/Script
  // picker. value is bindable (bind:value) for a plain two-way-bound field
  // like cfg.terminal.mode; callers that need to run side effects on an
  // explicit user pick (not a programmatic reset) instead pass value
  // one-way and listen for on:change.
  export let options: { value: string; label: string }[]
  export let value: string

  const dispatch = createEventDispatcher<{ change: string }>()

  function select(v: string) {
    if (v === value) return
    value = v
    dispatch('change', v)
  }
</script>

<div class="radio-group">
  {#each options as opt (opt.value)}
    <label><input type="radio" checked={value === opt.value} on:change={() => select(opt.value)} /> {opt.label}</label>
  {/each}
</div>

<style>
  .radio-group {
    display: flex;
    gap: 16px;
    font-size: 0.85rem;
    margin-bottom: 10px;
  }
  .radio-group input[type="radio"] {
    outline: none;
  }
</style>
