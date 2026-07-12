<script lang="ts">
  // Every small line icon both apps use (1px-ish strokes, currentColor, no
  // icon-font dependency), keyed by name. One shared component so a glyph
  // used by both apps (e.g. copy, the folder) exists exactly once.
  //
  // 'load' and 'open' are aliases for the same folder glyph — the two apps
  // name the action differently (Load config vs Open) but share the visual.
  export let name:
    | 'load'
    | 'open'
    | 'refresh'
    | 'settings'
    | 'run'
    | 'run-here'
    | 'cancel'
    | 'copy'
    | 'new'
    | 'save'
    | 'save-as'
    | 'restore'
    | 'expand-all'
    | 'collapse-all'
    | 'add'
    | 'remove'
    | 'reorder'
    | 'edit'
    | 'preview'
    | 'split-v'
    | 'split-h'
</script>

{#if name === 'load' || name === 'open'}
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
{:else if name === 'settings'}
  <svg viewBox="0 0 16 16" width="14" height="14" aria-hidden="true">
    <circle cx="8" cy="8" r="3" fill="none" stroke="currentColor" stroke-width="1.1" />
    <g stroke="currentColor" stroke-width="1.1" stroke-linecap="round">
      <line x1="11" y1="8" x2="13.3" y2="8" />
      <line x1="10.12" y1="10.12" x2="11.75" y2="11.75" />
      <line x1="8" y1="11" x2="8" y2="13.3" />
      <line x1="5.88" y1="10.12" x2="4.25" y2="11.75" />
      <line x1="5" y1="8" x2="2.7" y2="8" />
      <line x1="5.88" y1="5.88" x2="4.25" y2="4.25" />
      <line x1="8" y1="5" x2="8" y2="2.7" />
      <line x1="10.12" y1="5.88" x2="11.75" y2="4.25" />
    </g>
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
{:else if name === 'copy'}
  <svg viewBox="0 0 16 16" width="14" height="14" aria-hidden="true">
    <rect x="2" y="2" width="8" height="8" rx="1" fill="none" stroke="currentColor" stroke-width="1.1" />
    <rect x="6" y="6" width="8" height="8" rx="1" fill="none" stroke="currentColor" stroke-width="1.1" />
  </svg>
{:else if name === 'new'}
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
{:else if name === 'save'}
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
{:else if name === 'save-as'}
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
{:else if name === 'restore'}
  <svg viewBox="0 0 16 16" width="14" height="14" aria-hidden="true">
    <path d="M3 8a5 5 0 1 0 1.6-3.7" fill="none" stroke="currentColor" stroke-width="1.2" stroke-linecap="round" />
    <path d="M4.9 1.9v2.8H2.1" fill="none" stroke="currentColor" stroke-width="1.2" stroke-linejoin="round" stroke-linecap="round" />
  </svg>
{:else if name === 'expand-all'}
  <svg viewBox="0 0 16 16" width="14" height="14" aria-hidden="true">
    <path d="M3 5l5 4 5-4" fill="none" stroke="currentColor" stroke-width="1.3" stroke-linecap="round" stroke-linejoin="round" />
    <path d="M3 10l5 4 5-4" fill="none" stroke="currentColor" stroke-width="1.3" stroke-linecap="round" stroke-linejoin="round" />
  </svg>
{:else if name === 'collapse-all'}
  <svg viewBox="0 0 16 16" width="14" height="14" aria-hidden="true">
    <path d="M3 6l5-4 5 4" fill="none" stroke="currentColor" stroke-width="1.3" stroke-linecap="round" stroke-linejoin="round" />
    <path d="M3 11l5-4 5 4" fill="none" stroke="currentColor" stroke-width="1.3" stroke-linecap="round" stroke-linejoin="round" />
  </svg>
{:else if name === 'add'}
  <svg viewBox="0 0 16 16" width="14" height="14" aria-hidden="true">
    <path d="M8 2.5v11M2.5 8h11" fill="none" stroke="currentColor" stroke-width="1.4" stroke-linecap="round" />
  </svg>
{:else if name === 'remove'}
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
{:else if name === 'reorder'}
  <!-- Drag-handle grip (2x3 dots) — a common, recognizable "reorder" glyph. -->
  <svg viewBox="0 0 16 16" width="14" height="14" aria-hidden="true">
    <circle cx="5" cy="3.5" r="1.1" fill="currentColor" />
    <circle cx="5" cy="8" r="1.1" fill="currentColor" />
    <circle cx="5" cy="12.5" r="1.1" fill="currentColor" />
    <circle cx="10" cy="3.5" r="1.1" fill="currentColor" />
    <circle cx="10" cy="8" r="1.1" fill="currentColor" />
    <circle cx="10" cy="12.5" r="1.1" fill="currentColor" />
  </svg>
{:else if name === 'edit'}
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
{:else if name === 'preview'}
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
{:else if name === 'split-v'}
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
