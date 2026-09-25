<script>
  export let item = { name: '', url: '', logo: '', subtitle: '', tags: [] };
  export let style = 'cards';
  // canEdit / itemIndex / onEdit / onDelete 由上层注入，用于单块编辑
  export let canEdit = false;
  export let itemIndex = 0;
  export let onEdit = () => {};
  export let onDelete = () => {};

  function getInitials(name) {
    if (!name) return '';
    return name
      .split(/\s+/)
      .filter(Boolean)
      .map((word) => word[0].toUpperCase())
      .join('')
      .slice(0, 3);
  }

  function handleClick() {
    if (item.target === '_blank') {
      window.open(item.url, '_blank');
    } else {
      window.location.href = item.url;
    }
  }

  function copyUrl() {
    navigator.clipboard.writeText(item.url);
  }

  function handleKeydown(event) {
    if (event.key === 'Enter' || event.key === ' ') {
      event.preventDefault();
      handleClick();
    }
  }

  function handleEdit(event) {
    event.stopPropagation();
    onEdit(itemIndex);
  }

  function handleDelete(event) {
    event.stopPropagation();
    onDelete(itemIndex);
  }
</script>

{#if style === 'cards'}
  <div class="service-item card">
    {#if canEdit}
      <div class="item-actions">
        <button class="act-btn" on:click={handleEdit} title="编辑" aria-label="编辑站点">
          <i class="fas fa-pen"></i>
        </button>
        <button class="act-btn danger" on:click={handleDelete} title="删除" aria-label="删除站点">
          <i class="fas fa-trash"></i>
        </button>
      </div>
    {/if}
    <button class="item-logo" on:click={handleClick} on:keydown={handleKeydown}>
      {#if item.logo}
        <img src={item.logo} alt={item.name} />
      {:else}
        <div class="logo-fallback">
          {#if item.name}
            {getInitials(item.name)}
          {:else}
            <i class="fas fa-link"></i>
          {/if}
        </div>
      {/if}
    </button>
    <div class="item-content">
      <div class="item-header">
        <button class="item-title" on:click={handleClick} on:keydown={handleKeydown}>
          {item.name}
        </button>
        {#if item.tags && item.tags.length > 0}
          {#each item.tags as tag}
            <span class="tag">{tag}</span>
          {/each}
        {/if}
      </div>
      {#if item.subtitle}
        <p class="subtitle">{item.subtitle}</p>
      {/if}
      <div class="url-section">
          <div class="url-display">
            <span class="url-text">{item.url}</span>
            <button class="copy-btn" on:click={copyUrl} title="复制链接">
              <i class="fas fa-copy"></i>
            </button>
          </div>
      </div>
    </div>
  </div>
{:else}
  <div class="service-item list">
    {#if canEdit}
      <div class="item-actions">
        <button class="act-btn" on:click={handleEdit} title="编辑" aria-label="编辑站点">
          <i class="fas fa-pen"></i>
        </button>
        <button class="act-btn danger" on:click={handleDelete} title="删除" aria-label="删除站点">
          <i class="fas fa-trash"></i>
        </button>
      </div>
    {/if}
    <button class="item-logo" on:click={handleClick} on:keydown={handleKeydown}>
      {#if item.logo}
        <img src={item.logo} alt={item.name} />
      {:else}
        <div class="logo-fallback">
          {#if item.name}
            {getInitials(item.name)}
          {:else}
            <i class="fas fa-link"></i>
          {/if}
        </div>
      {/if}
    </button>
    <div class="item-content">
      <button class="item-title" on:click={handleClick} on:keydown={handleKeydown}>
        {item.name}
      </button>
      {#if item.subtitle}
        <p class="item-subtitle">{item.subtitle}</p>
      {/if}
    </div>
    <div class="list-actions">
        <div class="url-popover">
          <span class="url-text">{item.url}</span>
          <button class="copy-btn" on:click={copyUrl} title="复制链接">
            <i class="fas fa-copy"></i>
          </button>
        </div>
    </div>
    {#if item.tags && item.tags.length > 0}
      {#each item.tags as tag}
        <span class="tag">{tag}</span>
      {/each}
    {/if}
  </div>
{/if}

<style>
  .service-item {
    position: relative;
    transition: all 0.3s ease;
  }

  /* 单块编辑入口：hover / focus-within 时出现 */
  .item-actions {
    position: absolute;
    top: 8px;
    right: 8px;
    display: flex;
    gap: 4px;
    opacity: 0;
    transition: opacity 0.2s ease;
    z-index: 2;
  }

  .service-item:hover .item-actions,
  .service-item:focus-within .item-actions {
    opacity: 1;
  }

  .act-btn {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    width: 28px;
    height: 28px;
    border-radius: 8px;
    border: 1px solid rgba(255, 255, 255, 0.18);
    background: rgba(10, 16, 28, 0.75);
    color: rgba(255, 255, 255, 0.85);
    font-size: 0.75rem;
    cursor: pointer;
    transition: all 0.2s ease;
  }

  .act-btn:hover {
    background: var(--accent-soft, rgba(45, 139, 139, 0.35));
    color: var(--accent, #a8dadc);
    border-color: var(--accent, #a8dadc);
  }

  .act-btn.danger:hover {
    background: rgba(220, 53, 69, 0.22);
    color: #ff9aa2;
    border-color: rgba(220, 53, 69, 0.6);
  }

  /* Card style */
  .service-item.card {
    display: flex;
    align-items: flex-start;
    gap: 16px;
    padding: 16px;
    background: var(--surface, rgba(255, 255, 255, 0.05));
    border-radius: 12px;
    border: 1px solid var(--border, rgba(255, 255, 255, 0.05));
  }

  .service-item.card:hover {
    background: var(--surface-hover, rgba(255, 255, 255, 0.1));
    border-color: var(--accent-line, rgba(168, 218, 220, 0.55));
  }

  /* List style */
  .service-item.list {
    display: flex;
    align-items: center;
    gap: 12px;
    padding: 12px 16px;
    background: var(--surface, rgba(255, 255, 255, 0.03));
    border-radius: 8px;
    border: 1px solid var(--border, rgba(255, 255, 255, 0.05));
  }

  .service-item.list:hover {
    background: var(--surface-hover, rgba(255, 255, 255, 0.08));
    border-color: var(--accent-line, rgba(168, 218, 220, 0.55));
  }

  .item-logo {
    flex-shrink: 0;
    background: transparent;
    border: none;
    padding: 0;
    cursor: pointer;
    transition: transform 0.2s;
  }

  .item-logo:hover {
    transform: scale(1.05);
  }

  .item-logo:focus {
    outline: 2px solid var(--accent, #a8dadc);
    outline-offset: 2px;
    border-radius: 8px;
  }

  .item-logo img {
    width: 48px;
    height: 48px;
    object-fit: contain;
    border-radius: 8px;
  }

  .logo-fallback {
    width: 48px;
    height: 48px;
    display: flex;
    align-items: center;
    justify-content: center;
    /* 原来是「深色背景色 → 深色」的渐变配白字，主题一换就脏；改成中性面 + 强调色文字 */
    background: var(--surface-hover, rgba(255, 255, 255, 0.12));
    border: 1px solid var(--border, rgba(255, 255, 255, 0.12));
    border-radius: 8px;
    color: var(--accent, #a8dadc);
    font-size: 1.2rem;
  }

  .list .item-logo img,
  .list .logo-fallback {
    width: 36px;
    height: 36px;
  }

  .list .logo-fallback {
    font-size: 1rem;
  }

  .item-content {
    flex: 1;
    min-width: 0;
    display: flex;
    flex-direction: column;
    gap: 4px;
  }

  .item-header {
    display: flex;
    align-items: center;
    gap: 8px;
    flex-wrap: wrap;
  }

  .item-title {
    background: transparent;
    border: none;
    padding: 0;
    font-size: 1.1rem;
    font-weight: 600;
    color: var(--ink, #ffffff);
    cursor: pointer;
    text-align: left;
    transition: color 0.2s;
  }

  .item-title:hover {
    color: var(--accent, #a8dadc);
  }

  .item-title:focus {
    outline: 2px solid var(--accent, #a8dadc);
    outline-offset: 2px;
  }

  .list .item-title {
    font-size: 1rem;
  }

  .subtitle {
    font-size: 0.85rem;
    color: var(--ink-muted, rgba(255, 255, 255, 0.6));
    margin: 0;
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }

  .item-subtitle {
    font-size: 0.8rem;
    color: var(--ink-muted, rgba(255, 255, 255, 0.6));
    margin: 0;
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }

  .url-section {
    display: flex;
    align-items: center;
    gap: 8px;
    margin-top: 4px;
  }

  .url-display {
    display: flex;
    align-items: center;
    gap: 8px;
    flex: 1;
    background: rgba(0, 0, 0, 0.2);
    padding: 6px 10px;
    border-radius: 6px;
  }

  .url-text {
    flex: 1;
    font-size: 0.75rem;
    color: rgba(255, 255, 255, 0.5);
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }

  .copy-btn {
    background: transparent;
    border: none;
    color: rgba(255, 255, 255, 0.5);
    cursor: pointer;
    padding: 4px;
    border-radius: 4px;
    transition: all 0.2s;
    font-size: 0.8rem;
  }

  .copy-btn:hover {
    color: var(--accent, #a8dadc);
    background: var(--accent-soft, rgba(168, 218, 220, 0.16));
  }

  .list-actions {
    display: flex;
    align-items: center;
    gap: 8px;
    flex-shrink: 0;
    margin-left: auto;
    /* 给右上角的编辑 / 删除按钮留出位置，避免遮挡 */
    padding-right: 62px;
  }

  /* 列表视图的 URL：常态内联显示、超长省略。
     原来是绝对定位且 top/margin-top 被注释掉，导致它永久浮在行上盖住标题。 */
  .url-popover {
    display: flex;
    align-items: center;
    gap: 8px;
    max-width: 260px;
    padding: 6px 10px;
    border-radius: 8px;
    background: rgba(0, 0, 0, 0.22);
    border: 1px solid rgba(255, 255, 255, 0.08);
  }

  .tag {
    display: inline-block;
    padding: 2px 8px;
    background: var(--accent-soft, rgba(168, 218, 220, 0.16));
    border: 1px solid var(--accent-line, rgba(168, 218, 220, 0.55));
    color: var(--accent, #a8dadc);
    font-size: 0.7rem;
    font-weight: 500;
    border-radius: 4px;
    text-transform: uppercase;
    letter-spacing: 0.5px;
  }

  .list .tag {
    flex-shrink: 0;
  }

  @media (max-width: 768px) {
    .list-actions {
      padding-right: 0;
      flex-direction: column;
      align-items: flex-end;
      gap: 4px;
    }

    .url-popover {
      max-width: 160px;
    }
  }
</style>
