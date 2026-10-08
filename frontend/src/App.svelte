  <script>
  import { onMount } from 'svelte';
  import Header from './components/Header.svelte';
  import Navbar from './components/Navbar.svelte';
  import Toolbar from './components/Toolbar.svelte';
  import TagFilter from './components/TagFilter.svelte';
  import ServiceGroup from './components/ServiceGroup.svelte';
  import YamlEditor from './components/YamlEditor.svelte';
  import BlockEditor from './components/BlockEditor.svelte';
  import LoginModal from './components/LoginModal.svelte';
  import { pageConfig, currentRoute, viewStyle, currentTheme, getThemeTokens, userInfo, colorMode, searchQuery } from './stores.js';

  // loading: 是否正在请求；booted: 是否已完成首次加载。
  // 只有首次加载才显示整页 Loading —— 切换页面时若把 Header/Navbar/Toolbar 一起
  // 换掉再重建，视觉上就是每次点菜单都「闪一下」。
  let loading = true;
  let booted = false;
  // loadSeq: 请求序号，避免快速切换时「先发的慢响应」覆盖后发的页面
  let loadSeq = 0;
  let error = null;
  let selectedTag = '';
  // 初值由下面的响应式块统一赋值，这里不需要再给一次
  let filteredServices;
  let allTags;
  let showYamlEditor = false;
  let showLoginModal = false;
  let iframeNav = null;
  // 单块编辑（表单 / 源码）：{ kind, mode, serviceIndex, index }
  let blockEditor = null;

  function openItemEditor(serviceIndex, itemIndex) {
    blockEditor = { kind: 'item', mode: 'edit', serviceIndex, index: itemIndex };
  }

  function openNewItem(serviceIndex) {
    blockEditor = { kind: 'item', mode: 'insert', serviceIndex, index: 0 };
  }

  function openServiceEditor(serviceIndex) {
    blockEditor = { kind: 'service', mode: 'edit', serviceIndex, index: serviceIndex };
  }

  function openNewService() {
    blockEditor = { kind: 'service', mode: 'insert', serviceIndex: 0, index: 0 };
  }

  function closeBlockEditor() {
    blockEditor = null;
  }

  function handleBlockSaved() {
    blockEditor = null;
    loadConfig(getRoute());
  }

  async function deleteItem(serviceIndex, itemIndex) {
    if (!confirm('确定删除这个站点吗？')) {
      return;
    }

    try {
      const resp = await fetch(`/api/page${getRoute()}?op=block`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        credentials: 'include',
        body: JSON.stringify({
          kind: 'item',
          action: 'delete',
          service_index: serviceIndex,
          index: itemIndex,
        }),
      });
      const payload = await resp.json().catch(() => null);
      if (!resp.ok || !payload || !payload.success) {
        throw new Error((payload && payload.error) || '删除失败');
      }
      await loadConfig(getRoute());
    } catch (e) {
      error = e.message;
    }
  }

  // 色彩模式为 system 时，实际采用系统偏好
  let systemPrefersDark = true;

  $: effectiveMode = $colorMode === 'system'
    ? (systemPrefersDark ? 'dark' : 'light')
    : $colorMode;

  // 主题只提供色值映射，组件一律使用语义 token（--ink / --accent / --surface ...）
  $: themeVars = Object.entries(getThemeTokens($currentTheme, effectiveMode))
    .map(([k, v]) => `${k}: ${v};`)
    .join('');

  // token 写到 documentElement（而非只挂在 .theme-wrapper 上）：
  // body 在 wrapper 之外，只挂 wrapper 的话 body 拿不到主题色，
  // 亮色模式下 overscroll 会露出 :root 里的深色兜底
  $: applyThemeTokens(getThemeTokens($currentTheme, effectiveMode), effectiveMode);

  function applyThemeTokens(tokens, mode) {
    if (typeof document === 'undefined' || !document.documentElement) {
      return;
    }

    const root = document.documentElement;
    // 让原生控件（滚动条、select 下拉、日期选择器等）跟随明暗
    root.style.colorScheme = mode;

    for (const [key, value] of Object.entries(tokens)) {
      root.style.setProperty(key, value);
    }
  }

  function matchesAllKeywords(text, keywords) {
    if (!keywords.length) return true;
    const lowerText = text.toLowerCase();
    return keywords.every(keyword => lowerText.includes(keyword));
  }

  function collectTags(services) {
    const tagCount = {};
    (services || []).forEach(service => {
      (service.items || []).forEach(item => {
        (item.tags || []).forEach(tag => {
          tagCount[tag] = (tagCount[tag] || 0) + 1;
        });
      });
    });
    return Object.entries(tagCount)
      .map(([name, count]) => ({ name, count }))
      .sort((a, b) => b.count - a.count);
  }

  function filterByTag(services, tag) {
    if (!tag) return services;
    return services.map(service => ({
      ...service,
      items: (service.items || []).filter(item =>
        (item.tags || []).includes(tag)
      )
    })).filter(service => service.items.length > 0);
  }

  $: {
    // 保留 _serviceIndex / _itemIndex：过滤后仍能定位到配置文件里的原始位置
    const baseServices = ($pageConfig.services || []).map((service, si) => ({
      ...service,
      _serviceIndex: si,
      items: (service.items || []).map((item, ii) => ({ ...item, _itemIndex: ii })),
    }));
    allTags = collectTags(baseServices);

    let result = baseServices;

    if ($searchQuery.length > 1) {
      const keywords = $searchQuery.split(/\s+/).filter(k => k.length > 0);
      result = result.map(service => ({
        ...service,
        items: (service.items || []).filter(item => {
          const searchFields = [
            item.name,
            item.subtitle,
            ...(item.tags || []),
            service.name
          ].filter(Boolean).join(' ');
          return matchesAllKeywords(searchFields, keywords);
        })
      })).filter(service => service.items.length > 0);
    }

    if (selectedTag) {
      result = filterByTag(result, selectedTag);
    }

    filteredServices = result;
  }

  // 获取当前路由
  function getRoute() {
    const path = window.location.pathname;
    return path === '/' ? '/' : path;
  }

  // 检查并处理 URL 中的 refresh 参数
  function getRefreshParam() {
    const urlParams = new URLSearchParams(window.location.search);
    const refresh = urlParams.get('refresh');
    if (refresh === 'true') {
      // 移除 URL 中的 refresh 参数
      urlParams.delete('refresh');
      const newUrl = window.location.pathname + (urlParams.toString() ? '?' + urlParams.toString() : '');
      history.replaceState({}, '', newUrl);
      return true;
    }
    return false;
  }

  // URL 中只恢复当前权限允许的 iframe 菜单。
  function restoreIframe() {
    const view = new URLSearchParams(window.location.search).get('view');
    iframeNav = ($pageConfig.navs || []).find(nav => nav.mode === 'iframe' && nav.url === view) || null;
    currentRoute.set(iframeNav ? iframeNav.url : getRoute());
  }

  async function loadConfig(route) {
    const currentRoutePath = route || getRoute();
    const seq = ++loadSeq;

    loading = true;
    error = null;
    // 这里先不清空搜索/标签：旧页面在请求期间保持原样，
    // 否则过滤条件一没，旧内容会先「展开跳一下」。新数据到了再重置。

    const shouldRefresh = getRefreshParam();
    const apiPath = shouldRefresh
      ? `/api/page${currentRoutePath === '/' ? '' : currentRoutePath}?refresh=true`
      : `/api/page${currentRoutePath === '/' ? '' : currentRoutePath}`;

    try {
      // 选中态立即更新，不等请求回来
      restoreIframe();

      const fetchOptions = {
        credentials: 'include',
      };
      const response = await fetch(apiPath, fetchOptions);

      // 期间用户又切了页面：丢弃本次结果，避免旧数据覆盖新页面
      if (seq !== loadSeq) {
        return;
      }

      if (response.status === 401) {
        // 需要认证：弹出登录对话框，由用户输入用户名和密码
        showLoginModal = true;
        return;
      }

      const result = await response.json();

      if (seq !== loadSeq) {
        return;
      }

      if (!result.success) {
        throw new Error(result.error || 'Failed to load config');
      }

      pageConfig.set(result.data);
      restoreIframe();
      userInfo.set(result.data.user_info || null);
      // 新页面数据已就绪，这时再重置过滤条件
      // （搜索框与这里共用同一个 store，所以会一起清空）
      searchQuery.set('');
      selectedTag = '';
      if (!localStorage.getItem('viewStyle')) {
        viewStyle.set(result.data.style || 'cards');
      }
    } catch (err) {
      if (seq !== loadSeq) {
        return;
      }
      error = err.message;
    } finally {
      if (seq === loadSeq) {
        loading = false;
        booted = true;
      }
    }
  }

  // 处理导航（单页应用）
  function handleNavigate(nav, event) {
    // 独立页面及浏览器的新标签页操作保留原生链接行为。
    if (nav.mode === 'page' || (nav.target && nav.target !== '_self') || event.button !== 0 ||
        event.ctrlKey || event.metaKey || event.shiftKey || event.altKey) {
      return;
    }
    event.preventDefault();
    if (nav.mode === 'iframe') {
      error = null;
      const url = new URL(window.location.href);
      url.searchParams.set('view', nav.url);
      // 保留 YAML 路由，刷新后仍能加载骨架；HTML 的 URL 只用于 iframe src。
      if (url.href !== window.location.href) history.pushState({}, '', url);
      iframeNav = nav;
      currentRoute.set(nav.url);
    } else if (nav.url !== getRoute() || iframeNav) {
      iframeNav = null;
      history.pushState({}, '', nav.url);
      loadConfig(nav.url);
    }
  }

  function handlePopstate() {
    loadConfig();
  }

  onMount(() => {
    // 跟随系统明暗
    const mq = window.matchMedia('(prefers-color-scheme: dark)');
    systemPrefersDark = mq.matches;
    const onSchemeChange = (e) => {
      systemPrefersDark = e.matches;
    };
    mq.addEventListener('change', onSchemeChange);

    loadConfig();

    // 监听浏览器前进后退
    window.addEventListener('popstate', handlePopstate);

    return () => {
      mq.removeEventListener('change', onSchemeChange);
      window.removeEventListener('popstate', handlePopstate);
    };
  });

  function handleSelectTag(tag) {
    selectedTag = tag;
  }

  function openYamlEditor() {
    showYamlEditor = true;
  }

  function closeYamlEditor() {
    showYamlEditor = false;
  }

  function handleSaveSuccess() {
    showYamlEditor = false;
    // 刷新页面数据
    loadConfig(getRoute());
  }

  function openLoginModal() {
    showLoginModal = true;
  }

  function handleLoginSuccess(event) {
    const info = event.detail;
    if (info && info.username) {
      // 后端可能返回包含权限等扩展字段的用户信息，这里整体保存
      userInfo.set(info);
      showLoginModal = false;
      // 登录成功后重新加载当前页面数据
      loadConfig(getRoute());
    }
  }

  function handleLogout() {
    userInfo.set(null);
    // 退出后跳转到首页并按游客身份重新加载数据
    if (getRoute() !== '/' || iframeNav) {
      history.pushState({}, '', '/');
    }
    loadConfig('/');
  }
</script>

<svelte:head>
  <title>{$pageConfig?.title || 'Home Dashboard'}</title>
</svelte:head>

<div class="theme-wrapper" style={themeVars}>
  <main class="app {$viewStyle} theme-{$currentTheme}">
    {#if !booted && loading}
      <div class="loading">
        <i class="fas fa-spinner fa-spin"></i>
        <span>Loading...</span>
      </div>
    {:else if !booted && error}
      <div class="error">
        <i class="fas fa-exclamation-circle"></i>
        <span>{error}</span>
      </div>
    {:else}
      <Header
        title={$pageConfig.title}
        subtitle={$pageConfig.subtitle}
        logo={$pageConfig.logo}
        on:open-editor={openYamlEditor}
        on:open-login={openLoginModal}
        on:logged-out={handleLogout}
      />

      {#if $pageConfig.navs && $pageConfig.navs.length > 0}
        <Navbar navs={$pageConfig.navs} currentPath={$currentRoute} currentMode={iframeNav ? 'iframe' : 'yaml'} onNavigate={handleNavigate} />
      {/if}

      {#if iframeNav && !error}
        {#key iframeNav.url}
          <iframe
            title={iframeNav.name || '嵌入页面'}
            src={iframeNav.url}
            style="width: 100%; min-height: 65vh; border: 0; border-radius: 12px; background: var(--surface);"
          ></iframe>
        {/key}
      {:else}
      <Toolbar
        onOpenEditor={openYamlEditor}
        onAddService={openNewService}
        mode={effectiveMode}
      />

      <div class="main-content">
        {#if allTags.length > 0}
          <aside class="sidebar">
            <TagFilter
              tags={allTags}
              {selectedTag}
              onSelectTag={handleSelectTag}
            />
          </aside>
        {/if}

        <div class="services-wrapper" aria-busy={loading}>
          {#if error}
            <!-- 首次加载之后的错误就地在内容区展示，不再整页替换（否则又是「闪一下」） -->
            <div class="no-results inline-error">
              <i class="fas fa-exclamation-circle"></i>
              <p>{error}</p>
            </div>
          {:else if filteredServices.length === 0}
            <div class="no-results">
              <i class="fas fa-search"></i>
              <p>没有找到匹配的服务</p>
            </div>
          {:else}
            <div class="services-container" style="--columns: {$pageConfig.columns || '3'}">
              {#each filteredServices as service (service._serviceIndex)}
                <ServiceGroup
                  {service}
                  style={$viewStyle}
                  canEdit={!!$pageConfig.can_write}
                  onEditItem={(itemIndex) => openItemEditor(service._serviceIndex, itemIndex)}
                  onDeleteItem={(itemIndex) => deleteItem(service._serviceIndex, itemIndex)}
                  onAddItem={() => openNewItem(service._serviceIndex)}
                  onEditService={() => openServiceEditor(service._serviceIndex)}
                />
              {/each}
            </div>
          {/if}
        </div>
      </div>

      {/if}

      {#if $pageConfig.footer}
        <footer class="footer">
          {$pageConfig.footer}
        </footer>
      {/if}
    {/if}
  </main>

  <!-- YAML Editor Modal（整文件，高级入口） -->
  {#if showYamlEditor}
    <YamlEditor
      pagePath={$currentRoute}
      on:close={closeYamlEditor}
      on:save-success={handleSaveSuccess}
    />
  {/if}

  <!-- 单块编辑：只改一个站点 / 分组，表单与源码可切换 -->
  {#if blockEditor}
    <BlockEditor
      pagePath={$currentRoute}
      kind={blockEditor.kind}
      mode={blockEditor.mode}
      serviceIndex={blockEditor.serviceIndex}
      index={blockEditor.index}
      on:close={closeBlockEditor}
      on:saved={handleBlockSaved}
    />
  {/if}

  <!-- Login Modal -->
  <LoginModal
    visible={showLoginModal}
    on:close={() => (showLoginModal = false)}
    on:login-success={handleLoginSuccess}
  />
</div>

<style>
  :global(*) {
    margin: 0;
    padding: 0;
    box-sizing: border-box;
  }

  :global(body) {
    font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, Oxygen, Ubuntu, Cantarell, sans-serif;
    background: var(--bg-deep, #1a2332);
    min-height: 100vh;
    color: var(--ink, #ffffff);
    transition: background 0.4s ease, color 0.4s ease;
  }

  /* 语义 token 的兜底值，保证在任何主题变量注入前也不会掉样式 */
  :global(:root) {
    --bg-deep: #1a2332;
    --bg-mid: #12293a;
    --bg-grad: linear-gradient(165deg, #1a2332 0%, #12293a 100%);
    --surface: rgba(255, 255, 255, 0.06);
    --surface-hover: rgba(255, 255, 255, 0.12);
    --border: rgba(255, 255, 255, 0.12);
    --border-strong: rgba(255, 255, 255, 0.24);
    --ink: #ffffff;
    --ink-muted: rgba(255, 255, 255, 0.68);
    --ink-soft: rgba(255, 255, 255, 0.45);
    --accent: #a8dadc;
    --accent-soft: rgba(168, 218, 220, 0.16);
    --accent-line: rgba(168, 218, 220, 0.55);
    --accent-ink: #1a2332;
    --ink-faint: rgba(255, 255, 255, 0.38);
    --ok-ink: #b9f6ca;
    --warn-ink: #ffe082;
    --danger-ink: #ff9aa2;
    --panel: rgba(12, 18, 30, 0.97);
    --panel-inset: rgba(0, 0, 0, 0.35);
    --menu: rgba(24, 30, 46, 0.96);
    --overlay-strong: rgba(10, 16, 28, 0.75);
    --code-bg: rgba(0, 0, 0, 0.3);
    --scrim: rgba(0, 0, 0, 0.6);
    --shadow: rgba(0, 0, 0, 0.35);
    --shadow-strong: rgba(0, 0, 0, 0.5);
  }

  .theme-wrapper {
    min-height: 100vh;
    /* 只用两段深色渐变：之前是三段并在右下角铺到浅色，白字对比度随位置剧烈变化 */
    background: var(--bg-grad, linear-gradient(165deg, #1a2332 0%, #12293a 100%));
    background-attachment: fixed;
    transition: background 0.4s ease;
  }

  .app {
    max-width: 1600px;
    margin: 0 auto;
    padding: 20px;
    min-height: 100vh;
  }

  .loading, .error {
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    min-height: 100vh;
    gap: 16px;
    font-size: 1.2rem;
  }

  .loading i, .error i {
    font-size: 3rem;
  }

  .error {
    color: var(--danger-ink);
  }

  .main-content {
    display: flex;
    gap: 20px;
  }

  .sidebar {
    width: 200px;
    flex-shrink: 0;
  }

  .services-wrapper {
    flex: 1;
    min-width: 0;
  }

  .no-results {
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    padding: 60px 20px;
    color: var(--ink-soft, rgba(255, 255, 255, 0.45));
    gap: 16px;
  }

  .no-results i {
    font-size: 3rem;
    opacity: 0.5;
  }

  .no-results p {
    font-size: 1.1rem;
  }

  .no-results.inline-error {
    color: var(--danger-ink, #ff9aa2);
  }

  .services-container {
    display: grid;
    /* 用 --columns 作为「最大列数」参与计算：每列最小宽度 =
       max(280px, 扣除间距后均分给 columns 列的宽度)。
       容器够宽时正好 columns 列；变窄后回落到 280px 起自动换列。
       之前是固定 repeat(columns, ...) + 媒体查询硬覆盖成 2 列，
       导致 columns: "4" 在 1366 宽度的笔记本上只显示 2 列。 */
    grid-template-columns: repeat(
      auto-fit,
      minmax(
        max(280px, calc((100% - (var(--columns, 3) - 1) * 24px) / var(--columns, 3))),
        1fr
      )
    );
    gap: 24px;
  }

  .services-container > :global(*) {
    min-width: 0;
  }

  /* Responsive */
  @media (max-width: 900px) {
    .main-content {
      flex-direction: column;
    }

    .sidebar {
      width: 100%;
    }
  }

  @media (max-width: 768px) {
    .services-container {
      grid-template-columns: 1fr;
    }
  }

  /* List view styles */
  .list .services-container {
    display: flex;
    flex-direction: column;
    gap: 16px;
  }

  .footer {
    text-align: center;
    padding: 40px 20px;
    color: var(--ink-soft, rgba(255, 255, 255, 0.45));
    font-size: 0.9rem;
  }

  /* 尊重系统「减少动态效果」设置 */
  @media (prefers-reduced-motion: reduce) {
    :global(*),
    .theme-wrapper {
      transition: none !important;
      animation: none !important;
    }
  }
</style>
