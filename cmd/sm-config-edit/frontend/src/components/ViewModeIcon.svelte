<script lang="ts">
  // Small line icons for the Displays view-mode toggle, styled after VS
  // Code's editor/preview and split-editor icons (simple 1px strokes,
  // currentColor so they follow the button's normal/active/hover color
  // without any extra styling).
  export let mode: 'edit' | 'preview' | 'split-v' | 'split-h'
</script>

{#if mode === 'edit'}
  <svg viewBox="0 0 16 16" width="14" height="14" aria-hidden="true">
    <path
      d="M11.4 1.6a1.2 1.2 0 0 1 1.7 0l1.3 1.3a1.2 1.2 0 0 1 0 1.7L5.6 13.4l-3.4.8.8-3.4z"
      fill="none"
      stroke="currentColor"
      stroke-width="1.2"
      stroke-linejoin="round"
      stroke-linecap="round"
    />
    <path d="M9.5 3.5l3 3" fill="none" stroke="currentColor" stroke-width="1.2" stroke-linecap="round" />
  </svg>
{:else if mode === 'preview'}
  <svg viewBox="0 0 16 16" width="14" height="14" aria-hidden="true">
    <path
      d="M1 8s2.7-4.5 7-4.5S15 8 15 8s-2.7 4.5-7 4.5S1 8 1 8z"
      fill="none"
      stroke="currentColor"
      stroke-width="1.2"
      stroke-linejoin="round"
      stroke-linecap="round"
    />
    <circle cx="8" cy="8" r="1.8" fill="none" stroke="currentColor" stroke-width="1.2" />
  </svg>
{:else if mode === 'split-v'}
  <svg viewBox="0 0 16 16" width="14" height="14" aria-hidden="true">
    <rect x="1.5" y="2.5" width="5.3" height="11" rx="1" fill="none" stroke="currentColor" stroke-width="1.2" />
    <rect x="9.2" y="2.5" width="5.3" height="11" rx="1" fill="none" stroke="currentColor" stroke-width="1.2" />
  </svg>
{:else}
  <svg viewBox="0 0 16 16" width="14" height="14" aria-hidden="true">
    <rect x="1.5" y="1.5" width="13" height="5.3" rx="1" fill="none" stroke="currentColor" stroke-width="1.2" />
    <rect x="1.5" y="9.2" width="13" height="5.3" rx="1" fill="none" stroke="currentColor" stroke-width="1.2" />
  </svg>
{/if}

<style>
  svg {
    display: block;
  }
</style>
