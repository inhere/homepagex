<script>
  import { createEventDispatcher } from 'svelte';
  import { userInfo } from '../stores.js';

  const dispatch = createEventDispatcher();

  export let title = 'Home Dashboard';
  export let subtitle = '';
  export let logo = '';

  // 是否展示权限下拉详情
  let showPermDropdown = false;

  async function logout() {
    try {
      await fetch('/api/logout', {
        method: 'POST',
        credentials: 'include',
      });
    } catch (e) {
      // ignore network errors on logout
    }
    userInfo.set(null);
    dispatch('logged-out');
  }

  function handleLoginClick() {
    dispatch('open-login');
  }

  function togglePermDropdown(event) {
    event.stopPropagation();
    showPermDropdown = !showPermDropdown;
  }

  function formatPermLabel(perm) {
    if (!perm) return '未知';
    switch (perm) {
      case 'rw':
        return '读写权限 (rw)';
      case 'ro':
        return '只读权限 (ro)';
      case 'no':
        return '禁止访问 (no)';
      default:
        return perm;
    }
  }

  function permIconClass(perm) {
    switch (perm) {
      case 'rw':
        return 'fas fa-unlock-alt';
      case 'ro':
        return 'fas fa-lock-open';
      case 'no':
        return 'fas fa-ban';
      default:
        return 'fas fa-circle';
    }
  }

  function permBadgeClass(perm) {
    switch (perm) {
      case 'rw':
        return 'perm-badge perm-rw';
      case 'ro':
        return 'perm-badge perm-ro';
      case 'no':
        return 'perm-badge perm-no';
      default:
        return 'perm-badge';
    }
  }

  // 根据后端透传的权限信息生成简单摘要，用于 Hover 显示
  $: permSummary = $userInfo && $userInfo.permissions && $userInfo.permissions.length
    ? $userInfo.permissions.map(p => `${p.path || '*' }:${p.perm}`).join(', ')
    : '';

  // 点击页面其他区域时关闭权限下拉
  function handleWindowClick(event) {
    const target = event.target;
    if (!(target instanceof HTMLElement)) {
      showPermDropdown = false;
      return;
    }
    if (!target.closest('.user-info-wrapper')) {
      showPermDropdown = false;
    }
  }
</script>

<svelte:window on:click={handleWindowClick} />

<header class="header">
  <div class="header-content">
    {#if logo}
      <img src={logo} alt="Logo" class="logo" />
    {:else}
      <div class="logo-placeholder">
        <i class="fas fa-home"></i>
      </div>
    {/if}

    <div class="title-section">
      <h1 class="title">{title}</h1>
      {#if subtitle}
        <p class="subtitle">{subtitle}</p>
      {/if}
    </div>

    <div class="user-section">
      {#if $userInfo}
        <div class="user-info-wrapper">
          <button
            type="button"
            class="user-info"
            title={permSummary ? `权限: ${permSummary}` : '已登录'}
            on:click={togglePermDropdown}
          >
            <i class="fas fa-user-circle"></i>
            <span class="username">{$userInfo.username}</span>
            {#if $userInfo.permissions && $userInfo.permissions.length}
              <i class="fas fa-chevron-down caret" class:open={showPermDropdown}></i>
            {/if}
          </button>

          {#if showPermDropdown && $userInfo.permissions && $userInfo.permissions.length}
            <div class="perm-dropdown">
              <div class="perm-header">
                <span class="perm-title">权限明细</span>
                <span class="perm-count">{$userInfo.permissions.length} 条规则</span>
              </div>
              <div class="perm-list">
                {#each $userInfo.permissions as perm (perm.path + perm.perm)}
                  <div class="perm-item">
                    <div class="perm-icon-wrapper">
                      <i class={permIconClass(perm.perm)}></i>
                    </div>
                    <div class="perm-content">
                      <div class="perm-path">
                        {perm.path || '*'}
                      </div>
                      <div class="perm-meta">
                        <span class={permBadgeClass(perm.perm)}>
                          {formatPermLabel(perm.perm)}
                        </span>
                      </div>
                    </div>
                  </div>
                {/each}
              </div>
              <div class="perm-footer">
                <span>路径规则与 config.yaml 中 auths 配置保持一致</span>
              </div>
            </div>
          {/if}
        </div>
        <button class="btn-auth" on:click={logout} title="退出登录">
          <i class="fas fa-sign-out-alt"></i>
          <span>退出</span>
        </button>
      {:else}
        <button class="btn-auth login" type="button" on:click={handleLoginClick} title="登录">
          <i class="fas fa-sign-in-alt"></i>
          <span>登录</span>
        </button>
      {/if}
    </div>
  </div>
</header>

<style>
  .header {
    margin-bottom: 30px;
    position: relative;
    z-index: 1200;
  }

  .header-content {
    display: flex;
    align-items: center;
    gap: 20px;
    padding: 20px 30px;
    backdrop-filter: blur(10px);
  }

  .logo {
    width: 64px;
    height: 64px;
    object-fit: contain;
    border-radius: 12px;
    flex-shrink: 0;
  }

  .logo-placeholder {
    width: 64px;
    height: 64px;
    display: flex;
    align-items: center;
    justify-content: center;
    /* 原来是「深色背景色」渐变配白字，换成中性面 + 强调色，随主题变化 */
    background: var(--surface-hover, rgba(255, 255, 255, 0.12));
    border: 1px solid var(--border, rgba(255, 255, 255, 0.12));
    border-radius: 12px;
    font-size: 2rem;
    color: var(--accent, #a8dadc);
    flex-shrink: 0;
  }

  .title-section {
    flex: 1;
    min-width: 0;
  }

  .title {
    font-size: 2.5rem;
    font-weight: 700;
    color: var(--ink, #ffffff);
    margin: 0;
    text-shadow: 0 2px 4px rgba(0, 0, 0, 0.3);
  }

  .subtitle {
    font-size: 1.1rem;
    color: rgba(255, 255, 255, 0.7);
    margin: 8px 0 0 0;
  }

  /* 用户信息区域 */
  .user-section {
    display: flex;
    align-items: center;
    gap: 10px;
    flex-shrink: 0;
  }

  .user-info-wrapper {
    position: relative;
  }

  .user-info {
    display: inline-flex;
    align-items: center;
    gap: 8px;
    padding: 8px 14px;
    background: rgba(255, 255, 255, 0.08);
    border-radius: 20px;
    color: rgba(255, 255, 255, 0.9);
    font-size: 0.9rem;
    border: 1px solid rgba(255, 255, 255, 0.15);
    cursor: pointer;
    transition: all 0.2s ease;
    outline: none;
  }

  .user-info:hover {
    background: rgba(255, 255, 255, 0.15);
    border-color: rgba(255, 255, 255, 0.35);
  }

  .user-info i {
    font-size: 1.1rem;
    color: var(--accent, #a8dadc);
  }

  .user-info .caret {
    font-size: 0.75rem;
    color: rgba(255, 255, 255, 0.7);
    transition: transform 0.2s ease;
  }

  .user-info .caret.open {
    transform: rotate(180deg);
  }

  .perm-dropdown {
    position: absolute;
    right: 0;
    top: calc(100% + 10px);
    min-width: 260px;
    max-height: 320px;
    padding: 10px 12px;
    border-radius: 14px;
    /* 下拉面板用固定的深色玻璃面，不再拿主题的「深色背景色 + 透明度」拼渐变 */
    background: rgba(12, 18, 30, 0.96);
    border: 1px solid var(--border, rgba(255, 255, 255, 0.18));
    box-shadow:
      0 18px 45px rgba(0, 0, 0, 0.45),
      0 0 0 1px rgba(0, 0, 0, 0.3);
    backdrop-filter: blur(18px);
    z-index: 1500;
    overflow: hidden;
    color: var(--ink, #f5f5f5);
  }

  .perm-header {
    display: flex;
    align-items: baseline;
    justify-content: space-between;
    margin-bottom: 8px;
  }

  .perm-title {
    font-size: 0.9rem;
    font-weight: 600;
    color: var(--ink, #ffffff);
  }

  .perm-count {
    font-size: 0.75rem;
    color: rgba(255, 255, 255, 0.65);
  }

  .perm-list {
    max-height: 220px;
    padding-right: 4px;
    margin: 0 -4px;
    overflow-y: auto;
  }

  .perm-item {
    display: flex;
    align-items: flex-start;
    gap: 10px;
    padding: 8px 6px;
    border-radius: 10px;
    transition: background 0.15s ease, transform 0.1s ease;
  }

  .perm-item:hover {
    background: rgba(255, 255, 255, 0.04);
    transform: translateX(1px);
  }

  .perm-icon-wrapper {
    width: 26px;
    height: 26px;
    border-radius: 999px;
    display: flex;
    align-items: center;
    justify-content: center;
    background: rgba(255, 255, 255, 0.12);
    flex-shrink: 0;
  }

  .perm-icon-wrapper i {
    font-size: 0.9rem;
    color: var(--accent, #a8dadc);
  }

  .perm-content {
    flex: 1;
    min-width: 0;
  }

  .perm-path {
    font-size: 0.85rem;
    color: rgba(255, 255, 255, 0.95);
    word-break: break-all;
  }

  .perm-meta {
    margin-top: 4px;
  }

  .perm-badge {
    display: inline-flex;
    align-items: center;
    padding: 2px 8px;
    border-radius: 999px;
    font-size: 0.7rem;
    letter-spacing: 0.01em;
    border: 1px solid rgba(255, 255, 255, 0.18);
    color: rgba(255, 255, 255, 0.9);
  }

  /* 权限徽标用固定语义色（绿/黄/红），不再借用主题背景色 —— 否则边框和文字几乎看不见 */
  .perm-badge.perm-rw {
    background: rgba(76, 175, 80, 0.18);
    border-color: rgba(76, 175, 80, 0.7);
    color: #b9f6ca;
  }

  .perm-badge.perm-ro {
    background: rgba(255, 193, 7, 0.16);
    border-color: rgba(255, 193, 7, 0.7);
    color: #ffe082;
  }

  .perm-badge.perm-no {
    background: rgba(244, 67, 54, 0.18);
    border-color: rgba(244, 67, 54, 0.7);
    color: #ffb4b8;
  }

  .perm-footer {
    margin-top: 8px;
    padding-top: 6px;
    border-top: 1px solid rgba(255, 255, 255, 0.08);
  }

  .perm-footer span {
    font-size: 0.7rem;
    color: rgba(255, 255, 255, 0.6);
  }

  .btn-auth {
    display: inline-flex;
    align-items: center;
    gap: 6px;
    padding: 8px 16px;
    border-radius: 20px;
    font-size: 0.9rem;
    cursor: pointer;
    transition: all 0.2s ease;
    text-decoration: none;
    border: 1px solid rgba(255, 255, 255, 0.2);
    background: rgba(255, 255, 255, 0.08);
    color: rgba(255, 255, 255, 0.85);
  }

  .btn-auth:hover {
    background: rgba(255, 255, 255, 0.15);
    color: var(--ink, #ffffff);
    border-color: rgba(255, 255, 255, 0.4);
  }

  .btn-auth.login {
    border-color: var(--accent-line, rgba(168, 218, 220, 0.55));
    color: var(--accent, #a8dadc);
  }

  .btn-auth.login:hover {
    background: var(--accent-soft, rgba(168, 218, 220, 0.16));
  }

  @media (max-width: 768px) {
    .header-content {
      flex-wrap: wrap;
      padding: 16px 20px;
      gap: 12px;
    }

    .title-section {
      flex: 1 1 auto;
    }

    .title {
      font-size: 1.8rem;
    }

    .logo, .logo-placeholder {
      width: 48px;
      height: 48px;
    }

    .user-info-wrapper {
      order: 2;
    }

    .user-section {
      width: 100%;
      justify-content: flex-end;
      position: relative;
    }

    .btn-auth span {
      display: none;
    }

    .btn-auth {
      padding: 8px 12px;
    }

    .perm-dropdown {
      right: 0;
      left: auto;
      top: calc(100% + 8px);
      max-width: min(320px, 90vw);
    }
  }
</style>
