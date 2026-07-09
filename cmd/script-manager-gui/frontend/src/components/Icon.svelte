<script lang="ts">
  // Icons for the new toolbar (load/refresh/config-edit) and the Command
  // pane's action buttons (run/run-here/cancel/copy) — same style as
  // sm-config-edit's ToolbarIcon.svelte (1px-ish strokes, currentColor), not
  // shared with it directly since the two frontends don't share a
  // components/ folder.
  export let name: 'load' | 'refresh' | 'config-edit' | 'run' | 'run-here' | 'cancel' | 'copy'
</script>

{#if name === 'load'}
  <svg viewBox="0 0 16 16" width="14" height="14" aria-hidden="true">
    <path
      d="M1.5 4.2a.7.7 0 0 1 .7-.7h3.3l1.2 1.5h6.1a.7.7 0 0 1 .7.7v.8H2.2z"
      fill="none"
      stroke="currentColor"
      stroke-width="1.1"
      stroke-linejoin="round"
    />
    <path
      d="M1.9 6.5h11.8a.6.6 0 0 1 .58.75l-1.1 4.4a.9.9 0 0 1-.87.65H3.29a.9.9 0 0 1-.87-.65l-1.1-4.4a.6.6 0 0 1 .58-.75z"
      fill="none"
      stroke="currentColor"
      stroke-width="1.1"
      stroke-linejoin="round"
    />
  </svg>
{:else if name === 'refresh'}
  <svg viewBox="0 0 16 16" width="14" height="14" aria-hidden="true">
    <path d="M2.5 8a5.5 5.5 0 0 1 9.3-4" fill="none" stroke="currentColor" stroke-width="1.2" stroke-linecap="round" />
    <path
      d="M11.8 4v-2.3M11.8 4H9.5"
      fill="none"
      stroke="currentColor"
      stroke-width="1.2"
      stroke-linecap="round"
      stroke-linejoin="round"
    />
    <path d="M13.5 8a5.5 5.5 0 0 1-9.3 4" fill="none" stroke="currentColor" stroke-width="1.2" stroke-linecap="round" />
    <path
      d="M4.2 12v2.3M4.2 12H6.5"
      fill="none"
      stroke="currentColor"
      stroke-width="1.2"
      stroke-linecap="round"
      stroke-linejoin="round"
    />
  </svg>
{:else if name === 'config-edit'}
  <svg viewBox="0 0 16 16" width="14" height="14" aria-hidden="true">
    <path
      d="M10.5 2.5l3 3-8 8-3.5 1 1-3.5z"
      fill="none"
      stroke="currentColor"
      stroke-width="1.1"
      stroke-linejoin="round"
      stroke-linecap="round"
    />
    <path d="M9.5 3.5l3 3" stroke="currentColor" stroke-width="1.1" />
  </svg>
{:else if name === 'run'}
  <svg viewBox="0 0 16 16" width="14" height="14" aria-hidden="true">
    <path d="M4 2.5v11l9-5.5z" fill="currentColor" />
  </svg>
{:else if name === 'run-here'}
  <svg viewBox="0 0 16 16" width="14" height="14" aria-hidden="true">
    <rect x="1.5" y="1.5" width="13" height="13" rx="2" fill="none" stroke="currentColor" stroke-width="1.1" />
    <path d="M6 5v6l5-3z" fill="currentColor" />
  </svg>
{:else if name === 'cancel'}
  <svg viewBox="0 0 16 16" width="14" height="14" aria-hidden="true">
    <path d="M3.5 3.5l9 9M12.5 3.5l-9 9" stroke="currentColor" stroke-width="1.3" stroke-linecap="round" />
  </svg>
{:else}
  <svg viewBox="0 0 16 16" width="14" height="14" aria-hidden="true">
    <rect x="4.5" y="3" width="7" height="10.5" rx="1" fill="none" stroke="currentColor" stroke-width="1.1" />
    <path
      d="M6.3 3V1.8a.6.6 0 0 1 .6-.6h2.2a.6.6 0 0 1 .6.6V3"
      fill="none"
      stroke="currentColor"
      stroke-width="1.1"
    />
  </svg>
{/if}

<style>
  svg {
    display: block;
  }
</style>
