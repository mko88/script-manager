<script lang="ts">
  // Shared by the global Actions section and an item's Custom Actions list —
  // both edit the same []Action-shaped data (internal/configedit.ActionDTO).
  import StringListEditor from './StringListEditor.svelte'

  export let action: { id: string; title: string; description: string; cmd: string; groups: string[]; noWait: boolean }
  export let showId = true
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
  <div class="field">
    <span>Groups</span>
    <StringListEditor bind:items={action.groups} placeholder="group name" />
  </div>
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
</style>
