<script lang="ts">
  export let message = ''
</script>

{#if message}
  <div class="toast">{message}</div>
{/if}
