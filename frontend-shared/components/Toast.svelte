<script lang="ts">
  import { toastMessage } from '../toast'
</script>

{#if $toastMessage}
  <div class="toast">{$toastMessage}</div>
{/if}
