<script>
  export let tags = [];
  export let selectedTag = '';
  export let onSelectTag = () => {};

  function getTagCount(tag) {
    return tag.count || 0;
  }
</script>

<div class="tag-filter">
  <div class="tag-header">
    <i class="fas fa-tags"></i>
    <span>标签筛选</span>
  </div>
  <div class="tag-list">
    <button
      class="tag-btn"
      class:active={selectedTag === ''}
      on:click={() => onSelectTag('')}
    >
      <span class="tag-name">全部</span>
      <span class="tag-count"></span>
    </button>
    {#each tags as tag}
      <button
        class="tag-btn"
        class:active={selectedTag === tag.name}
        on:click={() => onSelectTag(selectedTag === tag.name ? '' : tag.name)}
      >
        <span class="tag-name">{tag.name}</span>
        <span class="tag-count">{tag.count}</span>
      </button>
    {/each}
  </div>
</div>

<style>
  .tag-filter {
    display: flex;
    flex-direction: column;
    gap: 12px;
    padding: 16px;
    background: var(--surface, rgba(255, 255, 255, 0.05));
    border-radius: 12px;
    border: 1px solid var(--border, rgba(255, 255, 255, 0.1));
  }

  .tag-header {
    display: flex;
    align-items: center;
    gap: 8px;
    font-size: 0.9rem;
    font-weight: 600;
    color: var(--ink-muted, rgba(255, 255, 255, 0.8));
  }

  .tag-header i {
    color: var(--accent, #a8dadc);
  }

  .tag-list {
    display: flex;
    flex-direction: column;
    gap: 4px;
  }

  .tag-btn {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 8px;
    padding: 8px 12px;
    background: transparent;
    border: 1px solid transparent;
    border-radius: 8px;
    color: var(--ink-muted, rgba(255, 255, 255, 0.7));
    cursor: pointer;
    transition: background 0.2s ease, color 0.2s ease, border-color 0.2s ease;
    font-size: 0.85rem;
    white-space: nowrap;
  }

  .tag-btn:hover {
    background: var(--surface-hover, rgba(255, 255, 255, 0.08));
    color: var(--ink, #ffffff);
  }

  /* 选中态用强调色，避免「深色字压深色底」不可读 */
  .tag-btn.active {
    background: var(--accent-soft, rgba(168, 218, 220, 0.16));
    border-color: var(--accent-line, rgba(168, 218, 220, 0.55));
    color: var(--accent, #a8dadc);
  }

  .tag-name {
    flex: 1;
    text-align: left;
    text-transform: uppercase;
    letter-spacing: 0.5px;
    font-weight: 500;
  }

  .tag-count {
    font-size: 0.75rem;
    padding: 2px 8px;
    background: var(--surface-hover, rgba(255, 255, 255, 0.1));
    border-radius: 10px;
    color: var(--ink-muted, rgba(255, 255, 255, 0.6));
    font-weight: 500;
    min-width: 24px;
    text-align: center;
  }

  .tag-btn.active .tag-count {
    background: var(--accent, #a8dadc);
    color: var(--accent-ink, #1a2332);
  }

  /* 移动端：不再整体隐藏，改成横向可滚动，功能得以保留 */
  @media (max-width: 768px) {
    .tag-filter {
      padding: 12px;
    }

    .tag-list {
      flex-direction: row;
      overflow-x: auto;
      gap: 8px;
      padding-bottom: 4px;
    }

    .tag-btn {
      flex: 0 0 auto;
    }

    .tag-name {
      flex: 0 0 auto;
    }
  }
</style>
