<script>
  import { createEventDispatcher, onMount } from 'svelte';
  import { pageConfig } from '../stores.js';

  const dispatch = createEventDispatcher();

  // 已知图标源的元数据地址；实际可选来源由后端下发的 icon_cdn_keys 决定
  const SOURCE_META = {
    'dashboard-icons': {
      label: 'Dashboard Icons',
      metaUrl: 'https://cdn.jsdelivr.net/gh/homarr-labs/dashboard-icons@master/metadata.json',
      kind: 'dashboard',
      format: 'png',
    },
    'selfhst-icons': {
      label: 'selfh.st Icons',
      metaUrl: 'https://cdn.jsdelivr.net/gh/selfhst/icons@main/index.json',
      kind: 'selfhst',
      format: 'png',
    },
  };

  let searchQuery = '';
  let icons = [];
  let filteredIcons = [];
  let loading = false;
  let error = null;

  let activeSourceId = '';
  let sourceCache = {};

  // 只展示「已配置 + 有元数据地址」的来源
  $: sources = ($pageConfig.icon_cdn_keys || [])
    .map((key) => {
      const meta = SOURCE_META[key] || {};
      return {
        id: key,
        label: meta.label || key,
        metaUrl: meta.metaUrl || '',
        kind: meta.kind || 'plain',
        format: meta.format || 'png',
      };
    })
    .filter((s) => s.metaUrl);

  function getActiveSource() {
    return sources.find((s) => s.id === activeSourceId) || sources[0] || null;
  }

  // 接入后端的 icons-local 缓存：首次访问时由服务端下载并缓存，
  // 这样页面配置里存的是本地路径，不再直连 CDN。
  function buildIconUrl(source, icon) {
    return `icons-local/${source.id}/${source.format}/${icon.name}.${source.format}`;
  }

  function collectAliases(item) {
    const aliases = [];
    const raw =
      item.aliases || item.Aliases || item.tags || item.Tags ||
      item.keywords || item.Keywords || item.Category || item.category;

    if (Array.isArray(raw)) {
      aliases.push(...raw);
    } else if (typeof raw === 'string' && raw.trim()) {
      raw
        .split(/[,\s]+/)
        .map((s) => s.trim())
        .filter(Boolean)
        .forEach((s) => aliases.push(s));
    }
    return aliases;
  }

  function normalizeMetadata(source, metadata) {
    const result = [];

    // dashboard-icons: { name: { aliases: [...] } }
    if (source.kind === 'dashboard' && metadata && typeof metadata === 'object' && !Array.isArray(metadata)) {
      Object.entries(metadata).forEach(([name, data]) => {
        result.push({ name, displayName: name, aliases: (data && data.aliases) || [] });
      });
      return result;
    }

    if (source.kind !== 'selfhst') {
      return result;
    }

    if (metadata && Array.isArray(metadata.Icons)) {
      metadata = metadata.Icons;
    }

    const push = (ref, item) => {
      if (!ref) return;
      result.push({
        name: ref,
        displayName: item.Name || item.name || item.title || item.label || item.Reference || item.reference || ref,
        aliases: collectAliases(item),
      });
    };

    if (Array.isArray(metadata)) {
      metadata.forEach((item) => {
        if (!item) return;
        push(item.Reference || item.reference || item.Name || item.name || item.slug, item);
      });
      return result;
    }

    if (metadata && typeof metadata === 'object') {
      Object.entries(metadata).forEach(([ref, item]) => {
        if (!item) return;
        push(ref, item);
      });
    }
    return result;
  }

  async function loadIconsForSource(source) {
    if (!source) {
      return;
    }

    loading = true;
    error = null;

    try {
      if (sourceCache[source.id]) {
        icons = sourceCache[source.id];
      } else {
        const response = await fetch(source.metaUrl);
        if (!response.ok) throw new Error('Failed to load icons');

        const metadata = await response.json();
        const normalized = normalizeMetadata(source, metadata).map((icon) => ({
          ...icon,
          url: buildIconUrl(source, icon),
        }));

        sourceCache = { ...sourceCache, [source.id]: normalized };
        icons = normalized;
      }

      filteredIcons = icons.slice(0, 20);
    } catch (err) {
      error = err.message;
    } finally {
      loading = false;
    }
  }

  onMount(async () => {
    const source = getActiveSource();
    if (!source) {
      error = '未配置图标 CDN（config.yaml 的 icons_cdn）';
      return;
    }
    activeSourceId = source.id;
    await loadIconsForSource(source);
  });

  async function handleSourceChange(id) {
    if (id === activeSourceId) return;
    activeSourceId = id;
    searchQuery = '';
    await loadIconsForSource(getActiveSource());
  }

  function handleSearch(event) {
    searchQuery = event.target.value.toLowerCase();

    if (!searchQuery) {
      filteredIcons = icons.slice(0, 20);
      return;
    }

    const results = icons.filter((icon) => {
      const nameMatch = (icon.name || '').toLowerCase().includes(searchQuery);
      const displayNameMatch = (icon.displayName || '').toLowerCase().includes(searchQuery);
      const aliasMatch = (icon.aliases || []).some((alias) =>
        alias.toLowerCase().includes(searchQuery)
      );
      return nameMatch || displayNameMatch || aliasMatch;
    });

    filteredIcons = results.slice(0, 50);
  }

  function selectIcon(icon) {
    dispatch('select', icon.url);
  }
</script>

<div class="icon-search">
  <div class="search-header">
    <div class="search-input-wrapper">
      <i class="fas fa-search"></i>
      <input
        type="text"
        class="search-input"
        placeholder="搜索图标..."
        value={searchQuery}
        on:input={handleSearch}
      />
    </div>

    {#if sources.length > 1}
      <div class="source-toggle" aria-label="图标来源切换">
        {#each sources as source}
          <button
            type="button"
            class="source-btn"
            class:active={source.id === activeSourceId}
            on:click={() => handleSourceChange(source.id)}
          >
            {source.label}
          </button>
        {/each}
      </div>
    {/if}
  </div>

  <div class="icon-grid-container">
    {#if loading}
      <div class="loading-state">
        <i class="fas fa-spinner fa-spin"></i>
        <span>加载图标中...</span>
      </div>
    {:else if error}
      <div class="error-state">
        <i class="fas fa-exclamation-triangle"></i>
        <span>{error}</span>
      </div>
    {:else if filteredIcons.length === 0}
      <div class="empty-state">
        <i class="fas fa-search"></i>
        <span>未找到匹配的图标</span>
      </div>
    {:else}
      <div class="icon-grid">
        {#each filteredIcons as icon}
          <button
            class="icon-card"
            on:click={() => selectIcon(icon)}
            title={`${icon.name}（点击插入 icons-local 路径）`}
          >
            <img src={icon.url} alt={icon.name} class="icon-image" />
            <span class="icon-name">{icon.name}</span>
          </button>
        {/each}
      </div>
    {/if}
  </div>
</div>

<style>
  .icon-search {
    display: flex;
    flex-direction: column;
    gap: 12px;
    height: 100%;
  }

  .search-header {
    flex-shrink: 0;
    display: flex;
    align-items: center;
    gap: 12px;
  }

  .search-input-wrapper {
    flex: 1;
    display: flex;
    align-items: center;
    gap: 10px;
    padding: 10px 14px;
    background: rgba(255, 255, 255, 0.05);
    border: 1px solid rgba(255, 255, 255, 0.15);
    border-radius: 8px;
    transition: all 0.3s ease;
  }

  .source-toggle {
    display: inline-flex;
    padding: 2px;
    background: rgba(255, 255, 255, 0.06);
    border-radius: 999px;
    border: 1px solid rgba(255, 255, 255, 0.1);
  }

  .source-btn {
    border: none;
    background: transparent;
    color: rgba(255, 255, 255, 0.7);
    font-size: 0.75rem;
    padding: 6px 10px;
    border-radius: 999px;
    cursor: pointer;
    transition: all 0.2s ease;
    white-space: nowrap;
  }

  .source-btn:hover {
    background: rgba(255, 255, 255, 0.1);
  }

  .source-btn.active {
    background: var(--accent, #4a9eff);
    color: #08131f;
  }

  .search-input-wrapper:focus-within {
    background: rgba(255, 255, 255, 0.1);
    border-color: var(--accent, #4a9eff);
  }

  .search-input-wrapper i {
    color: rgba(255, 255, 255, 0.5);
    font-size: 0.9rem;
  }

  .search-input {
    flex: 1;
    background: transparent;
    border: none;
    outline: none;
    color: #e4e4e4;
    font-size: 0.9rem;
  }

  .search-input::placeholder {
    color: rgba(255, 255, 255, 0.4);
  }

  .icon-grid-container {
    flex: 1;
    overflow-x: auto;
    overflow-y: hidden;
    min-height: 0;
  }

  .icon-grid {
    display: flex;
    gap: 12px;
    padding: 4px;
    white-space: nowrap;
  }

  .icon-card {
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: 6px;
    padding: 10px 12px;
    background: rgba(255, 255, 255, 0.05);
    border: 1px solid rgba(255, 255, 255, 0.1);
    border-radius: 8px;
    cursor: pointer;
    transition: all 0.2s ease;
    flex-shrink: 0;
    min-width: 80px;
  }

  .icon-card:hover {
    background: rgba(255, 255, 255, 0.1);
    border-color: var(--accent, #4a9eff);
    transform: translateY(-2px);
  }

  /* 保留图标原色：之前用 brightness(0) invert(1) 把彩色 logo 变成白色剪影，
     预览与页面上实际效果不一致 */
  .icon-image {
    width: 48px;
    height: 48px;
    object-fit: contain;
  }

  .icon-name {
    font-size: 0.7rem;
    color: rgba(255, 255, 255, 0.7);
    text-align: center;
    word-break: break-word;
    line-height: 1.2;
    max-width: 100%;
    overflow: hidden;
    text-overflow: ellipsis;
    display: -webkit-box;
    -webkit-line-clamp: 2;
    -webkit-box-orient: vertical;
  }

  .loading-state,
  .error-state,
  .empty-state {
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    gap: 12px;
    padding: 40px 20px;
    color: rgba(255, 255, 255, 0.5);
  }

  .loading-state i,
  .error-state i,
  .empty-state i {
    font-size: 2rem;
  }

  .error-state {
    color: #ff6b6b;
  }

  .icon-grid-container::-webkit-scrollbar {
    height: 6px;
  }

  .icon-grid-container::-webkit-scrollbar-track {
    background: rgba(255, 255, 255, 0.05);
    border-radius: 3px;
  }

  .icon-grid-container::-webkit-scrollbar-thumb {
    background: rgba(255, 255, 255, 0.2);
    border-radius: 3px;
  }

  .icon-grid-container::-webkit-scrollbar-thumb:hover {
    background: rgba(255, 255, 255, 0.3);
  }
</style>
