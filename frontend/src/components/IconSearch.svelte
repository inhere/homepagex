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

  // 元数据都在 CDN 上：离线/内网环境可能一直不返回，必须加超时，
  // 否则编辑表单会一直卡在「加载图标中」（请求挂起、loading 不落地）。
  const META_TIMEOUT_MS = 8000;
  const MANUAL_HINT =
    '可手动填写图标路径，例如 icons-local/dashboard-icons/png/plex.png';

  // 服务端是否允许访问 CDN（config.yaml 的 icons_remote）。
  // false 时后端不会下载图标，前端也就不必去拉 CDN 元数据了。
  $: remoteEnabled = $pageConfig.icons_remote !== false;

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

  // fetchWithTimeout 给元数据请求加超时，避免不可达的 CDN 把请求挂死
  async function fetchWithTimeout(url) {
    const controller = new AbortController();
    const timer = setTimeout(() => controller.abort(), META_TIMEOUT_MS);
    try {
      return await fetch(url, { signal: controller.signal });
    } finally {
      clearTimeout(timer);
    }
  }

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
        const response = await fetchWithTimeout(source.metaUrl);
        if (!response.ok) throw new Error(`HTTP ${response.status}`);

        const metadata = await response.json();
        const normalized = normalizeMetadata(source, metadata).map((icon) => ({
          ...icon,
          url: buildIconUrl(source, icon),
        }));

        // 按名字去重：既避免重复图标，也保证下面 keyed each 的 key 唯一
        // （远端元数据由第三方维护，不能假定 name 一定不重复）
        const unique = [...new Map(normalized.map((icon) => [icon.name, icon])).values()];

        sourceCache = { ...sourceCache, [source.id]: unique };
        icons = unique;
      }

      filteredIcons = icons.slice(0, 20);
    } catch (err) {
      // 离线/内网时这里几乎必然失败：给出可操作的提示，而不是把表单卡在加载中
      error =
        err.name === 'AbortError'
          ? `图标元数据加载超时（${META_TIMEOUT_MS / 1000}s），网络不可达。${MANUAL_HINT}`
          : `图标元数据加载失败：${err.message}。${MANUAL_HINT}`;
    } finally {
      loading = false;
    }
  }

  onMount(async () => {
    // 后端已关闭远程图标：不必（也不应该）去请求 CDN
    if (!remoteEnabled) {
      error = `服务端已关闭远程图标（icons_remote: false），无法在线搜索图标。${MANUAL_HINT}`;
      return;
    }

    const source = getActiveSource();
    if (!source) {
      error = '未配置图标 CDN（config.yaml 的 icons_cdn）';
      return;
    }
    activeSourceId = source.id;
    await loadIconsForSource(source);
  });

  async function handleSourceChange(id) {
    if (id === activeSourceId || !remoteEnabled) return;
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
        {#each sources as source (source.id)}
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
        {#each filteredIcons as icon (icon.name)}
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
    background: var(--surface);
    border: 1px solid var(--border);
    border-radius: 8px;
    transition: all 0.3s ease;
  }

  .source-toggle {
    display: inline-flex;
    padding: 2px;
    background: var(--surface);
    border-radius: 999px;
    border: 1px solid var(--border);
  }

  .source-btn {
    border: none;
    background: transparent;
    color: var(--ink-muted);
    font-size: 0.75rem;
    padding: 6px 10px;
    border-radius: 999px;
    cursor: pointer;
    transition: all 0.2s ease;
    white-space: nowrap;
  }

  .source-btn:hover {
    background: var(--surface-hover);
  }

  .source-btn.active {
    background: var(--accent, #4a9eff);
    color: var(--accent-ink);
  }

  .search-input-wrapper:focus-within {
    background: var(--surface-hover);
    border-color: var(--accent, #4a9eff);
  }

  .search-input-wrapper i {
    color: var(--ink-soft);
    font-size: 0.9rem;
  }

  .search-input {
    flex: 1;
    background: transparent;
    border: none;
    outline: none;
    color: var(--ink);
    font-size: 0.9rem;
  }

  .search-input::placeholder {
    color: var(--ink-faint);
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
    background: var(--surface);
    border: 1px solid var(--border);
    border-radius: 8px;
    cursor: pointer;
    transition: all 0.2s ease;
    flex-shrink: 0;
    min-width: 80px;
  }

  .icon-card:hover {
    background: var(--surface-hover);
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
    color: var(--ink-muted);
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
    color: var(--ink-soft);
  }

  .loading-state i,
  .error-state i,
  .empty-state i {
    font-size: 2rem;
  }

  .error-state {
    color: var(--danger-ink);
    text-align: center;
    line-height: 1.6;
  }

  .icon-grid-container::-webkit-scrollbar {
    height: 6px;
  }

  .icon-grid-container::-webkit-scrollbar-track {
    background: var(--surface);
    border-radius: 3px;
  }

  .icon-grid-container::-webkit-scrollbar-thumb {
    background: var(--surface-hover);
    border-radius: 3px;
  }

  .icon-grid-container::-webkit-scrollbar-thumb:hover {
    background: var(--surface-hover);
  }
</style>
