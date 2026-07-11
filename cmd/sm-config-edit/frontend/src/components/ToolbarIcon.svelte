<script lang="ts">
  // Icons for the top toolbar's New/Open/Save/Save As — same style as
  // ViewModeIcon/ListActionIcon (1px strokes, currentColor).
  export let mode: 'new' | 'open' | 'save' | 'save-as' | 'restore' | 'expand-all' | 'collapse-all' | 'theme-dark' | 'theme-light'
</script>

{#if mode === 'new'}
  <svg viewBox="0 0 16 16" width="14" height="14" aria-hidden="true">
    <path
      d="M4 1.5h5l3 3v10a.5.5 0 0 1-.5.5h-7a.5.5 0 0 1-.5-.5v-12a.5.5 0 0 1 .5-.5z"
      fill="none"
      stroke="currentColor"
      stroke-width="1.2"
      stroke-linejoin="round"
    />
    <path d="M9 1.5v3h3" fill="none" stroke="currentColor" stroke-width="1.2" stroke-linejoin="round" />
    <path d="M8 8.2v3.6M6.2 10h3.6" stroke="currentColor" stroke-width="1.2" stroke-linecap="round" />
  </svg>
{:else if mode === 'open'}
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
{:else if mode === 'save'}
  <svg viewBox="0 0 16 16" width="14" height="14" aria-hidden="true">
    <path
      d="M3 2.5h7l2.5 2.5V13a.5.5 0 0 1-.5.5H3a.5.5 0 0 1-.5-.5V3a.5.5 0 0 1 .5-.5z"
      fill="none"
      stroke="currentColor"
      stroke-width="1.2"
      stroke-linejoin="round"
    />
    <path d="M5 2.5v3.3h4.6V2.5" fill="none" stroke="currentColor" stroke-width="1.2" stroke-linejoin="round" />
    <rect x="4.6" y="9" width="6.3" height="4" fill="none" stroke="currentColor" stroke-width="1.2" />
  </svg>
{:else if mode === 'save-as'}
  <svg viewBox="0 0 16 16" width="14" height="14" aria-hidden="true">
    <path
      d="M2.5 2h6l2.2 2.2V11a.5.5 0 0 1-.5.5H2.5A.5.5 0 0 1 2 11V2.5a.5.5 0 0 1 .5-.5z"
      fill="none"
      stroke="currentColor"
      stroke-width="1.1"
      stroke-linejoin="round"
    />
    <path d="M4.2 2v2.8h3.9V2" fill="none" stroke="currentColor" stroke-width="1.1" stroke-linejoin="round" />
    <rect x="3.8" y="7.3" width="5" height="3.2" fill="none" stroke="currentColor" stroke-width="1.1" />
    <path
      d="M11 11.2l2.6 2.6m0 0v-1.8m0 1.8h-1.8"
      fill="none"
      stroke="currentColor"
      stroke-width="1.1"
      stroke-linecap="round"
      stroke-linejoin="round"
    />
  </svg>
{:else if mode === 'restore'}
  <svg viewBox="0 0 16 16" width="14" height="14" aria-hidden="true">
    <path d="M3 8a5 5 0 1 0 1.6-3.7" fill="none" stroke="currentColor" stroke-width="1.2" stroke-linecap="round" />
    <path d="M4.9 1.9v2.8H2.1" fill="none" stroke="currentColor" stroke-width="1.2" stroke-linejoin="round" stroke-linecap="round" />
  </svg>
{:else if mode === 'expand-all'}
  <svg viewBox="0 0 16 16" width="14" height="14" aria-hidden="true">
    <path d="M3 5l5 4 5-4" fill="none" stroke="currentColor" stroke-width="1.3" stroke-linecap="round" stroke-linejoin="round" />
    <path d="M3 10l5 4 5-4" fill="none" stroke="currentColor" stroke-width="1.3" stroke-linecap="round" stroke-linejoin="round" />
  </svg>
{:else if mode === 'collapse-all'}
  <svg viewBox="0 0 16 16" width="14" height="14" aria-hidden="true">
    <path d="M3 6l5-4 5 4" fill="none" stroke="currentColor" stroke-width="1.3" stroke-linecap="round" stroke-linejoin="round" />
    <path d="M3 11l5-4 5 4" fill="none" stroke="currentColor" stroke-width="1.3" stroke-linecap="round" stroke-linejoin="round" />
  </svg>
{:else if mode === 'theme-dark'}
  <!-- Sun — shown while the dark theme is active; depicts the mode a click
       switches TO, matching the toggle's usual convention. -->
  <svg viewBox="0 0 16 16" width="14" height="14" aria-hidden="true">
    <circle cx="8" cy="8" r="3" fill="none" stroke="currentColor" stroke-width="1.2" />
    <path
      d="M8 1.5v1.6M8 12.9v1.6M1.5 8h1.6M12.9 8h1.6M3.3 3.3l1.1 1.1M11.6 11.6l1.1 1.1M3.3 12.7l1.1-1.1M11.6 4.4l1.1-1.1"
      stroke="currentColor"
      stroke-width="1.2"
      stroke-linecap="round"
    />
  </svg>
{:else}
  <!-- Moon — shown while the light theme is active. -->
  <svg viewBox="0 0 16 16" width="14" height="14" aria-hidden="true">
    <path d="M13 9.5A5.5 5.5 0 1 1 6.5 3a4.3 4.3 0 1 0 6.5 6.5z" fill="currentColor" />
  </svg>
{/if}

<style>
  svg {
    display: block;
  }
</style>
