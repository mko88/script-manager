<script lang="ts">
  // Shared by script-manager-gui's Command pane (both a cmd: action's
  // command text and a script: action's file content) and sm-config-edit's
  // Action editor (a script: action's file content preview) — one
  // line-numbered source display so both apps show code the same way.
  // The default slot is an optional corner overlay (script-manager-gui puts
  // its copy button there); sm-config-edit's simpler preview passes none.
  export let content: string
</script>

{#if content}
  <div class="script-source">
    <slot />
    {#each content.replace(/\n+$/, '').split('\n') as line, i (i)}
      <div class="script-source-row">
        <span class="script-source-no">{i + 1}</span>
        <span class="script-source-text">{line}</span>
      </div>
    {/each}
  </div>
{/if}

<style>
  .script-source {
    position: relative;
    background: var(--sm-bg-deep);
    border-radius: 4px;
    padding: 8px 0;
    margin: 0 0 8px;
    color: var(--sm-text);
    font-family: "SF Mono", Consolas, monospace;
    font-size: 0.85rem;
  }

  .script-source-row {
    display: flex;
    gap: 10px;
    padding: 0 8px;
  }

  .script-source-no {
    flex: none;
    width: 1.6em;
    text-align: right;
    color: var(--sm-line-number);
    user-select: none;
  }

  .script-source-text {
    flex: 1;
    min-width: 0;
    white-space: pre-wrap;
    word-break: break-word;
  }
</style>
