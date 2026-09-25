<script>
  export let navs = [];
  export let currentPath = '/';
  export let onNavigate = () => {};
</script>

<nav class="navbar">
  <div class="nav-list">
    {#each navs as nav}
      <button
        class="nav-item"
        class:active={currentPath === nav.url}
        on:click={() => onNavigate(nav.url)}
      >
        <i class={nav.icon}></i>
        <span>{nav.name}</span>
      </button>
    {/each}
  </div>
</nav>

<style>
  .navbar {
    background: var(--surface, rgba(255, 255, 255, 0.05));
    border-radius: 12px;
    padding: 8px;
    margin-bottom: 20px;
  }

  .nav-list {
    display: flex;
    gap: 8px;
    flex-wrap: wrap;
  }

  .nav-item {
    display: flex;
    align-items: center;
    gap: 8px;
    padding: 10px 16px;
    color: var(--ink-muted, rgba(255, 255, 255, 0.68));
    text-decoration: none;
    border-radius: 8px;
    transition: background 0.25s ease, color 0.25s ease;
    font-size: 0.95rem;
    background: transparent;
    border: 1px solid transparent;
    cursor: pointer;
  }

  .nav-item:hover {
    background: var(--surface-hover, rgba(255, 255, 255, 0.1));
    color: var(--ink, #ffffff);
  }

  /* 选中态：强调色文字 + 强调色淡底。
     之前是 color: var(--theme-primary)（深色背景色）压在深色半透明底上，几乎不可读 */
  .nav-item.active {
    background: var(--accent-soft, rgba(168, 218, 220, 0.16));
    border-color: var(--accent-line, rgba(168, 218, 220, 0.55));
    color: var(--accent, #a8dadc);
  }

  .nav-item i {
    font-size: 1rem;
  }

  @media (max-width: 768px) {
    .nav-list {
      justify-content: center;
    }

    .nav-item span {
      display: none;
    }

    .nav-item {
      padding: 10px 14px;
    }
  }
</style>
