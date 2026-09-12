<script lang="ts">
  import { createEventDispatcher } from 'svelte'

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
    font-size: var(--sm-type-base);
    margin-bottom: 10px;
  }
  .radio-group input[type="radio"] {
    outline: none;
  }
</style>
