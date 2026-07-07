<script lang="ts">
  // Small line icons for the Items/Action Groups/Actions list toolbars —
  // same style as ViewModeIcon (1px strokes, currentColor, no icon-font
  // dependency).
  export let mode: 'add' | 'remove'
</script>

{#if mode === 'add'}
  <svg viewBox="0 0 16 16" width="14" height="14" aria-hidden="true">
    <path d="M8 2.5v11M2.5 8h11" fill="none" stroke="currentColor" stroke-width="1.4" stroke-linecap="round" />
  </svg>
{:else}
  <svg viewBox="0 0 16 16" width="14" height="14" aria-hidden="true">
    <path
      d="M3.5 4.5h9M6.5 4.5V3a1 1 0 0 1 1-1h1a1 1 0 0 1 1 1v1.5M5 4.5l.6 8.4a1 1 0 0 0 1 .9h2.8a1 1 0 0 0 1-.9l.6-8.4"
      fill="none"
      stroke="currentColor"
      stroke-width="1.2"
      stroke-linejoin="round"
      stroke-linecap="round"
    />
  </svg>
{/if}

<style>
  svg {
    display: block;
  }
</style>
