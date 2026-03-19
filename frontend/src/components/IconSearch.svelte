<script>
  import { createEventDispatcher, onMount } from 'svelte';

  const dispatch = createEventDispatcher();

  let searchQuery = '';
  let icons = [];
  let filteredIcons = [];
  let loading = true;
  let error = null;

  let activeSourceId = 'dashboard';
  let sourceCache = {};

  const ICON_SOURCES = [
    {
      id: 'dashboard',
      label: 'Dashboard Icons',
      metaUrl: 'https://cdn.jsdelivr.net/gh/homarr-labs/dashboard-icons@master/metadata.json',
      type: 'dashboard'
    },
    {
      id: 'selfhst',
      label: 'selfh.st Icons',
      metaUrl: 'https://cdn.jsdelivr.net/gh/selfhst/icons@main/index.json',
      type: 'selfhst'
    }
  ];

  function getActiveSource() {
    return ICON_SOURCES.find((s) => s.id === activeSourceId) || ICON_SOURCES[0];
  }

  function buildIconUrl(source, icon) {
    if (source.type === 'dashboard') {
      return `https://cdn.jsdelivr.net/gh/homarr-labs/dashboard-icons@master/png/${icon.name}.png`;
    }

    // selfhst/icons - 使用 reference 作为文件名
    if (source.type === 'selfhst') {
      return `https://cdn.jsdelivr.net/gh/selfhst/icons@main/png/${icon.name}.png`;
    }

    return icon.url || '';
  }

  function normalizeMetadata(source, metadata) {
    const icons = [];

    // dashboard-icons: metadata 是对象 { name: { aliases: [...] } }
    if (source.type === 'dashboard' && metadata && typeof metadata === 'object' && !Array.isArray(metadata)) {
      Object.entries(metadata).forEach(([name, data]) => {
        icons.push({
          name,
          displayName: name,
          aliases: (data && data.aliases) || []
        });
      });
      return icons;
    }

    // selfhst/icons: index.json 可能是数组或对象，这里做兼容处理
    if (source.type === 'selfhst') {
      // 某些版本可能包在 Icons 字段里
      if (metadata && Array.isArray(metadata.Icons)) {
        metadata = metadata.Icons;
      }

      // 数组形式
      if (Array.isArray(metadata)) {
        metadata.forEach((item) => {
          if (!item) return;
          const ref =
            item.Reference ||
            item.reference ||
            item.Name ||
            item.name ||
            item.slug;
          if (!ref) return;
          const aliases = [];
          const rawAliases =
            item.aliases ||
            item.Aliases ||
            item.tags ||
            item.Tags ||
            item.keywords ||
            item.Keywords ||
            item.Category ||
            item.category;

          if (Array.isArray(rawAliases)) {
            aliases.push(...rawAliases);
          } else if (typeof rawAliases === 'string' && rawAliases.trim()) {
            rawAliases
              .split(/[,\s]+/)
              .map((s) => s.trim())
              .filter(Boolean)
              .forEach((s) => aliases.push(s));
          }

          const displayName =
            item.Name ||
            item.name ||
            item.title ||
            item.label ||
            item.Reference ||
            item.reference ||
            ref;
          icons.push({
            name: ref,
            displayName,
            aliases
          });
        });
        return icons;
      }

      // 对象形式 { immich: { ... }, ... }
      if (metadata && typeof metadata === 'object') {
        Object.entries(metadata).forEach(([ref, item]) => {
          if (!item) return;
          const aliases = [];
          const rawAliases =
            item.aliases ||
            item.Aliases ||
            item.tags ||
            item.Tags ||
            item.keywords ||
            item.Keywords ||
            item.Category ||
            item.category;

          if (Array.isArray(rawAliases)) {
            aliases.push(...rawAliases);
          } else if (typeof rawAliases === 'string' && rawAliases.trim()) {
            rawAliases
              .split(/[,\s]+/)
              .map((s) => s.trim())
              .filter(Boolean)
              .forEach((s) => aliases.push(s));
          }

          const displayName =
            item.Name ||
            item.name ||
            item.title ||
            item.label ||
            item.Reference ||
            item.reference ||
            ref;
          icons.push({
            name: ref,
            displayName,
            aliases
          });
        });
        return icons;
      }
    }

    return icons;
  }

  async function loadIconsForSource(source) {
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
          url: buildIconUrl(source, icon)
        }));

        sourceCache = {
          ...sourceCache,
          [source.id]: normalized
        };

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
    await loadIconsForSource(source);
  });

  async function handleSourceChange(id) {
    if (id === activeSourceId) return;
    activeSourceId = id;
    searchQuery = '';
    const source = getActiveSource();
    await loadIconsForSource(source);
  }

  function handleSearch(event) {
    searchQuery = event.target.value.toLowerCase();

    if (!searchQuery) {
      filteredIcons = icons.slice(0, 20);
      return;
    }

    const results = icons.filter((icon) => {
      const nameMatch = (icon.name || '').toLowerCase().includes(searchQuery);
      const displayNameMatch = (icon.displayName || '').toLowerCase().includes(
        searchQuery
      );
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

    <div class="source-toggle" aria-label="图标来源切换">
      {#each ICON_SOURCES as source}
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
            title={icon.name}
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
    background: var(--theme-primary, #4a9eff);
    color: #ffffff;
  }

  .search-input-wrapper:focus-within {
    background: rgba(255, 255, 255, 0.1);
    border-color: var(--theme-primary, #4a9eff);
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
    border-color: var(--theme-primary, #4a9eff);
    transform: translateY(-2px);
  }

  .icon-image {
    width: 48px;
    height: 48px;
    object-fit: contain;
    filter: brightness(0) invert(1);
    opacity: 0.9;
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
    width: 6px;
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
