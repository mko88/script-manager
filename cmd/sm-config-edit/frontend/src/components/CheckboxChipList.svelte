<script lang="ts">
  export let options: string[]
  export let selected: string[]

  function toggle(v: string) {
    selected = selected.includes(v) ? selected.filter((x) => x !== v) : [...selected, v]
  }
</script>

<div class="checkbox-list">
  {#each options as opt}
    <label class="checkbox-chip">
      <input type="checkbox" checked={selected.includes(opt)} on:change={() => toggle(opt)} />
      {opt}
    </label>
  {/each}
</div>
