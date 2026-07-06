<script lang="ts">
  // Shared by the global Actions section and an item's Custom Actions list —
  // both edit the same []Action-shaped data (internal/configedit.ActionDTO).
  export let action: { id: string; title: string; description: string; cmd: string; groups: string[]; noWait: boolean }
  export let showId = true
  // The Action Groups catalog's IDs — Groups is a picker against this list
  // (like Items' Actions/Action groups pickers), not free text, so an
  // action can only belong to a group that's actually been catalogued.
  export let allActionGroups: string[] = []

  function toggleGroup(id: string) {
    action.groups = action.groups.includes(id) ? action.groups.filter((g) => g !== id) : [...action.groups, id]
  }
</script>

<div class="action-form">
  {#if showId}
    <label class="field">
      <span>ID</span>
      <input type="text" bind:value={action.id} placeholder="unique id, referenced by items' Actions list" />
    </label>
  {/if}
  <label class="field">
    <span>Title</span>
    <input type="text" bind:value={action.title} />
  </label>
  <label class="field">
    <span>Description</span>
    <input type="text" bind:value={action.description} />
  </label>
  <label class="field">
    <span>Command</span>
    <textarea rows="3" bind:value={action.cmd}></textarea>
  </label>
  {#if allActionGroups.length > 0}
    <div class="field">
      <span>Groups</span>
      <div class="checkbox-list">
        {#each allActionGroups as g}
          <label class="checkbox-chip">
            <input type="checkbox" checked={action.groups.includes(g)} on:change={() => toggleGroup(g)} />
            {g}
          </label>
        {/each}
      </div>
    </div>
  {/if}
  <label class="field-checkbox">
    <input type="checkbox" bind:checked={action.noWait} />
    <span>No wait (close the terminal immediately instead of pausing after the command finishes)</span>
  </label>
</div>

<style>
  .action-form {
    display: flex;
    flex-direction: column;
    gap: 8px;
  }
  .field {
    display: flex;
    flex-direction: column;
    gap: 2px;
    font-size: 0.8rem;
    color: var(--sm-text-muted);
  }
  .field input,
  .field textarea {
    background: var(--sm-bg-deep);
    color: var(--sm-text);
    border: 1px solid var(--sm-border);
    border-radius: 4px;
    padding: 5px 7px;
    font-family: inherit;
    font-size: 0.85rem;
  }
  .field textarea {
    font-family: "SF Mono", Consolas, monospace;
  }
  .field-checkbox {
    display: flex;
    align-items: center;
    gap: 6px;
    font-size: 0.8rem;
    color: var(--sm-text-muted);
  }
  .checkbox-list {
    display: flex;
    flex-wrap: wrap;
    gap: 6px;
  }
  .checkbox-chip {
    display: flex;
    align-items: center;
    gap: 4px;
    background: var(--sm-bg-deep);
    border: 1px solid var(--sm-border);
    border-radius: 999px;
    padding: 2px 9px;
    font-size: 0.75rem;
    cursor: pointer;
  }
</style>
