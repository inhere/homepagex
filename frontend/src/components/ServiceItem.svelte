<script>
  export let item = { name: '', url: '', logo: '', subtitle: '', tags: [] };
  export let style = 'cards';

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
</script>

{#if style === 'cards'}
  <div class="service-item card">
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
    transition: all 0.3s ease;
  }

  /* Card style */
  .service-item.card {
    display: flex;
    align-items: flex-start;
    gap: 16px;
    padding: 16px;
    background: rgba(255, 255, 255, 0.05);
    border-radius: 12px;
    border: 1px solid rgba(255, 255, 255, 0.05);
  }

  .service-item.card:hover {
    background: rgba(255, 255, 255, 0.1);
    border-color: var(--theme-primary-rgba);
  }

  /* List style */
  .service-item.list {
    display: flex;
    align-items: center;
    gap: 12px;
    padding: 12px 16px;
    background: rgba(255, 255, 255, 0.03);
    border-radius: 8px;
    border: 1px solid rgba(255, 255, 255, 0.05);
  }

  .service-item.list:hover {
    background: rgba(255, 255, 255, 0.08);
    border-color: var(--theme-primary-rgba);
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
    outline: 2px solid var(--theme-primary);
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
    background: linear-gradient(135deg, var(--theme-primary), var(--theme-secondary));
    border-radius: 8px;
    color: white;
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
    color: #ffffff;
    cursor: pointer;
    text-align: left;
    transition: color 0.2s;
  }

  .item-title:hover {
    color: var(--theme-primary);
  }

  .item-title:focus {
    outline: 2px solid var(--theme-primary);
    outline-offset: 2px;
  }

  .list .item-title {
    font-size: 1rem;
  }

  .subtitle {
    font-size: 0.85rem;
    color: rgba(255, 255, 255, 0.6);
    margin: 0;
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }

  .item-subtitle {
    font-size: 0.8rem;
    color: rgba(255, 255, 255, 0.6);
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
    color: var(--theme-primary);
    background: var(--theme-primary-rgba);
  }

  .list-actions {
    position: relative;
    display: flex;
    align-items: center;
    gap: 4px;
  }

  .url-popover {
    position: absolute;
    /* top: 100%; */
    right: 0;
    /* margin-top: 8px; */
    display: flex;
    align-items: center;
    gap: 8px;
    background: rgba(30, 30, 50, 0.95);
    padding: 8px 12px;
    border-radius: 8px;
    border: 1px solid rgba(255, 255, 255, 0.1);
    z-index: 100;
    min-width: 200px;
    backdrop-filter: blur(10px);
  }

  .tag {
    display: inline-block;
    padding: 2px 8px;
    background: var(--theme-primary-rgba);
    color: var(--theme-primary);
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
    .url-popover {
      min-width: 160px;
      right: -50%;
    }
  }
</style>
