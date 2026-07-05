<script lang="ts">
  // A plain <textarea> has no line-number gutter; this pairs one with a
  // synced-scroll number column. Only the scroll position needs manual
  // syncing — line count and gutter height already follow `value` reactively.
  export let value = ''

  let textareaEl: HTMLTextAreaElement
  let gutterEl: HTMLDivElement

  $: lineCount = value.split('\n').length

  function syncScroll() {
    if (gutterEl && textareaEl) gutterEl.scrollTop = textareaEl.scrollTop
  }
</script>

<div class="line-numbered-textarea">
  <div class="line-numbers" bind:this={gutterEl}>
    {#each { length: lineCount } as _, i (i)}
      <div class="line-number">{i + 1}</div>
    {/each}
  </div>
  <textarea
    bind:this={textareaEl}
    bind:value
    on:scroll={syncScroll}
    on:input={syncScroll}
    spellcheck="false"
  ></textarea>
</div>

<style>
  .line-numbered-textarea {
    display: flex;
    flex: 1 1 auto;
    min-height: 0;
    border: 1px solid var(--sm-border);
    border-radius: 4px;
    background: var(--sm-bg-deep);
    overflow: hidden;
  }

  .line-numbers {
    flex: none;
    padding: 5px 8px 5px 6px;
    text-align: right;
    color: var(--sm-text-faint);
    user-select: none;
    overflow: hidden;
    border-right: 1px solid var(--sm-border);
    font-family: "SF Mono", Consolas, monospace;
    font-size: 0.8rem;
    line-height: 1.4;
  }

  .line-number {
    white-space: nowrap;
  }

  textarea {
    flex: 1 1 auto;
    min-width: 0;
    min-height: 60px;
    resize: none;
    border: none;
    background: transparent;
    color: var(--sm-text);
    padding: 5px 7px;
    font-family: "SF Mono", Consolas, monospace;
    font-size: 0.8rem;
    line-height: 1.4;
    outline: none;
  }
</style>
