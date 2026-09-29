<script>
  import { viewStyle, themes, currentTheme, pageConfig, colorMode, COLOR_MODES, themeSwatch } from '../stores.js';

  export let onSearch = () => {};
  export let onOpenEditor = () => {};
  export let onAddService = () => {};

  let searchQuery = '';
  let showThemeDropdown = false;

  $: currentThemeName = themes.find(t => t.id === $currentTheme)?.name || '默认主题';

  // 当前生效的明暗：由 App 传入（system 已在 App 侧解析成 light/dark）
  export let mode = 'dark';

  function handleSearch(event) {
    searchQuery = event.target.value;
    onSearch(searchQuery);
  }

  function clearSearch() {
    searchQuery = '';
    onSearch('');
  }

  function toggleStyle() {
    viewStyle.update(style => style === 'cards' ? 'list' : 'cards');
  }

  function selectTheme(themeId) {
    currentTheme.set(themeId);
    showThemeDropdown = false;
  }

  function handleKeydown(event) {
    if (event.key === 'Escape') {
      showThemeDropdown = false;
      document.querySelector('.search-input').blur();
    }
  }

  function handleClickOutside(event) {
    if (!event.target.closest('.theme-selector')) {
      showThemeDropdown = false;
    }
  }

  // 是否可编辑由后端随页面数据下发（can_write），前端不再自己复刻一套权限匹配逻辑
  $: canEditCurrentRoute = !!$pageConfig?.can_write;
</script>

<svelte:window on:keydown={handleKeydown} on:click={handleClickOutside} />

<div class="toolbar">
  <div class="search-box">
    <i class="fas fa-search"></i>
    <input
      type="text"
      class="search-input"
      placeholder="搜索服务..."
      value={searchQuery}
      on:input={handleSearch}
    />
    {#if searchQuery}
      <button class="clear-btn" on:click={clearSearch} title="清除">
        <i class="fas fa-times"></i>
      </button>
    {/if}
  </div>

  <div class="toolbar-actions">
    <!-- 色彩模式：亮 / 暗 / 系统（放在主题选择前面） -->
    <div class="mode-selector" role="group" aria-label="色彩模式">
      {#each COLOR_MODES as m (m.id)}
        <button
          type="button"
          class="mode-btn"
          class:active={$colorMode === m.id}
          title={m.name}
          on:click|stopPropagation={() => colorMode.set(m.id)}
        >
          <i class={m.icon}></i>
          <span>{m.name}</span>
        </button>
      {/each}
    </div>

    <div class="theme-selector">
      <button
        class="theme-btn"
        on:click|stopPropagation={() => showThemeDropdown = !showThemeDropdown}
      >
        <div
          class="theme-indicator"
          style="background: {themeSwatch($currentTheme, mode)}"
        ></div>
        <span class="theme-name">{currentThemeName}</span>
        <i class="fas fa-chevron-down" class:open={showThemeDropdown}></i>
      </button>

      {#if showThemeDropdown}
        <div class="theme-dropdown">
          {#each themes as theme (theme.id)}
            <button
              class="theme-option"
              class:active={$currentTheme === theme.id}
              on:click={() => selectTheme(theme.id)}
            >
              <div
                class="theme-preview"
                style="background: {themeSwatch(theme.id, mode)}"
              ></div>
              <span>{theme.name}</span>
              {#if $currentTheme === theme.id}
                <i class="fas fa-check"></i>
              {/if}
            </button>
          {/each}
        </div>
      {/if}
    </div>

    <button class="style-toggle" on:click={toggleStyle} title="切换视图">
      <i class="fas {$viewStyle === 'cards' ? 'fa-list' : 'fa-th-large'}"></i>
      <span>{$viewStyle === 'cards' ? '列表' : '卡片'}</span>
    </button>

    {#if canEditCurrentRoute}
      <button class="edit-btn" type="button" on:click={onAddService} title="新增分组">
        <i class="fas fa-folder-plus"></i>
        <span>新增分组</span>
      </button>
      <button class="edit-btn" type="button" on:click={onOpenEditor} title="编辑整份页面 YAML（高级）">
        <i class="fas fa-edit"></i>
        <span>原始 YAML</span>
      </button>
    {/if}
  </div>
</div>

<style>
  .toolbar {
    display: flex;
    align-items: center;
    gap: 16px;
    margin-bottom: 20px;
  }

  .search-box {
    flex: 1;
    max-width: 400px;
    display: flex;
    align-items: center;
    gap: 10px;
    padding: 10px 16px;
    background: var(--surface-hover);
    border: 1px solid var(--border);
    border-radius: 8px;
    transition: all 0.3s ease;
  }

  .search-box:focus-within {
    background: var(--surface-hover);
    border-color: var(--accent-soft);
  }

  .search-box i {
    color: var(--ink-soft);
    font-size: 0.9rem;
  }

  .search-input {
    flex: 1;
    background: transparent;
    border: none;
    outline: none;
    color: var(--ink);
    font-size: 0.95rem;
  }

  .search-input::placeholder {
    color: var(--ink-faint);
  }

  .clear-btn {
    background: transparent;
    border: none;
    color: var(--ink-soft);
    cursor: pointer;
    padding: 4px;
    display: flex;
    align-items: center;
    justify-content: center;
    transition: color 0.2s;
  }

  .clear-btn:hover {
    color: var(--ink);
  }

  .toolbar-actions {
    display: flex;
    align-items: center;
    gap: 12px;
  }

  .mode-selector {
    display: inline-flex;
    gap: 2px;
    padding: 3px;
    background: var(--surface, rgba(255, 255, 255, 0.06));
    border: 1px solid var(--border, rgba(255, 255, 255, 0.12));
    border-radius: 10px;
  }

  .mode-btn {
    display: inline-flex;
    align-items: center;
    gap: 6px;
    padding: 6px 10px;
    border: none;
    border-radius: 8px;
    background: transparent;
    color: var(--ink-muted, rgba(255, 255, 255, 0.7));
    font-size: 0.8rem;
    cursor: pointer;
    transition: background 0.2s ease, color 0.2s ease;
  }

  .mode-btn:hover {
    color: var(--ink, #ffffff);
  }

  .mode-btn.active {
    background: var(--accent-soft, rgba(168, 218, 220, 0.16));
    color: var(--accent, #a8dadc);
  }

  .theme-selector {
    position: relative;
  }

  .theme-btn {
    display: flex;
    align-items: center;
    gap: 8px;
    padding: 10px 16px;
    background: var(--surface-hover);
    border: 1px solid var(--border);
    border-radius: 8px;
    color: var(--ink);
    cursor: pointer;
    transition: all 0.3s ease;
    font-size: 0.9rem;
  }

  .theme-btn:hover {
    background: var(--surface-hover);
  }

  .theme-indicator {
    width: 20px;
    height: 20px;
    border-radius: 4px;
    flex-shrink: 0;
  }

  .theme-name {
    white-space: nowrap;
  }

  .theme-btn i {
    font-size: 0.75rem;
    transition: transform 0.2s;
  }

  .theme-btn i.open {
    transform: rotate(180deg);
  }

  .theme-dropdown {
    position: absolute;
    top: 100%;
    right: 0;
    margin-top: 8px;
    background: var(--menu);
    border: 1px solid var(--border);
    border-radius: 12px;
    overflow: hidden;
    z-index: 1000;
    min-width: 160px;
    backdrop-filter: blur(10px);
  }

  .theme-option {
    display: flex;
    align-items: center;
    gap: 12px;
    width: 100%;
    padding: 10px 14px;
    background: transparent;
    border: none;
    cursor: pointer;
    text-align: left;
    color: var(--ink);
    font-size: 0.9rem;
    transition: background 0.2s;
  }

  .theme-option:hover {
    background: var(--surface-hover);
  }

  .theme-option.active {
    background: var(--accent-soft);
  }

  .theme-preview {
    width: 20px;
    height: 20px;
    border-radius: 4px;
    flex-shrink: 0;
  }

  .theme-option i {
    margin-left: auto;
    color: var(--accent);
    font-size: 0.85rem;
  }

  .style-toggle {
    display: flex;
    align-items: center;
    gap: 8px;
    padding: 10px 16px;
    background: var(--surface-hover);
    border: 1px solid var(--border);
    border-radius: 8px;
    color: var(--ink);
    cursor: pointer;
    transition: all 0.3s ease;
    font-size: 0.9rem;
  }

  .style-toggle:hover {
    background: var(--surface-hover);
    transform: translateY(-2px);
  }

  .edit-btn {
    display: inline-flex;
    align-items: center;
    gap: 6px;
    padding: 9px 16px;
    border-radius: 20px;
    border: 1px solid var(--border-strong);
    background: var(--surface-hover);
    color: var(--ink);
    font-size: 0.9rem;
    cursor: pointer;
    transition: all 0.2s ease;
  }

  .edit-btn i {
    font-size: 0.9rem;
  }

  .edit-btn:hover {
    background: var(--accent-soft);
    border-color: var(--accent);
    transform: translateY(-1px);
  }

  @media (max-width: 768px) {
    .mode-btn span {
      display: none;
    }

    .mode-btn {
      padding: 6px 8px;
    }

    .toolbar {
      flex-direction: column;
    }

    .search-box {
      max-width: 100%;
      width: 100%;
    }

    .toolbar-actions {
      width: 100%;
      justify-content: space-between;
      flex-wrap: wrap;
    }

    .theme-name {
      display: none;
    }

    .theme-btn {
      padding: 10px 12px;
    }

    .style-toggle span {
      display: none;
    }

    .style-toggle {
      padding: 10px 12px;
    }

    .edit-btn span {
      display: none;
    }

    .edit-btn {
      padding: 9px 10px;
    }

    .theme-dropdown {
      right: 0;
      min-width: 140px;
    }
  }
</style>
