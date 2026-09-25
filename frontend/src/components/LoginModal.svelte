<script>
  import { createEventDispatcher } from 'svelte';

  const dispatch = createEventDispatcher();

  export let visible = false;

  let username = '';
  let password = '';
  let loading = false;
  let error = '';

  async function handleSubmit(event) {
    event.preventDefault();
    if (!username || !password) {
      error = '请输入用户名和密码';
      return;
    }

    loading = true;
    error = '';

    try {
      const resp = await fetch('/api/login', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        credentials: 'include',
        body: JSON.stringify({ username, password }),
      });

      const result = await resp.json().catch(() => null);

      if (!resp.ok || !result || !result.success) {
        error = (result && result.error) || '登录失败，请检查用户名和密码';
        return;
      }

      const data = result.data || {};
      // 将后端返回的用户信息整体透传给上层（包含权限等扩展字段）
      dispatch('login-success', data.username ? data : { username });
    } catch (e) {
      error = '网络错误，请稍后重试';
    } finally {
      loading = false;
    }
  }

  function close() {
    if (loading) return;
    dispatch('close');
  }
</script>

{#if visible}
  <div class="modal-backdrop">
    <div class="modal">
      <h2 class="modal-title">登录</h2>
      <p class="modal-subtitle">请输入配置中定义的用户名和密码</p>

      <form class="form" on:submit|preventDefault={handleSubmit}>
        <label class="field">
          <span>用户名</span>
          <input
            type="text"
            bind:value={username}
            placeholder="admin"
            autocomplete="username"
          />
        </label>

        <label class="field">
          <span>密码</span>
          <input
            type="password"
            bind:value={password}
            placeholder="••••••••"
            autocomplete="current-password"
          />
        </label>

        {#if error}
          <div class="error">
            <i class="fas fa-exclamation-circle"></i>
            <span>{error}</span>
          </div>
        {/if}

        <div class="actions">
          <button type="button" class="btn secondary" on:click={close} disabled={loading}>
            取消
          </button>
          <button type="submit" class="btn primary" disabled={loading}>
            {#if loading}
              <i class="fas fa-spinner fa-spin"></i>
              <span>登录中...</span>
            {:else}
              <span>登录</span>
            {/if}
          </button>
        </div>
      </form>
    </div>
  </div>
{/if}

<style>
  .modal-backdrop {
    position: fixed;
    inset: 0;
    background:
      radial-gradient(circle at top, var(--accent-soft, rgba(168, 218, 220, 0.35)), transparent 55%),
      rgba(0, 0, 0, 0.6);
    display: flex;
    align-items: center;
    justify-content: center;
    z-index: 2000;
  }

  .modal {
    width: 100%;
    max-width: 420px;
    background: rgba(12, 18, 30, 0.97);
    border-radius: 16px;
    padding: 24px 24px 20px;
    border: 1px solid var(--border, rgba(255, 255, 255, 0.12));
    box-shadow: 0 18px 45px rgba(0, 0, 0, 0.6);
    backdrop-filter: blur(16px);
    color: var(--ink, #f5f5f5);
  }

  .modal-title {
    font-size: 1.4rem;
    margin-bottom: 4px;
  }

  .modal-subtitle {
    font-size: 0.9rem;
    color: rgba(255, 255, 255, 0.7);
    margin-bottom: 18px;
  }

  .form {
    display: flex;
    flex-direction: column;
    gap: 14px;
  }

  .field {
    display: flex;
    flex-direction: column;
    gap: 6px;
    font-size: 0.9rem;
  }

  .field span {
    color: rgba(255, 255, 255, 0.85);
  }

  .field input {
    padding: 9px 10px;
    border-radius: 8px;
    border: 1px solid rgba(255, 255, 255, 0.25);
    background: rgba(5, 10, 25, 0.9);
    color: #f5f5f5;
    outline: none;
    font-size: 0.95rem;
    transition: border 0.2s, box-shadow 0.2s, background 0.2s;
  }

  .field input::placeholder {
    color: rgba(255, 255, 255, 0.4);
  }

  .field input:focus {
    border-color: var(--accent, #a8dadc);
    box-shadow: 0 0 0 1px var(--accent-soft, rgba(168, 218, 220, 0.4));
    background: rgba(5, 10, 25, 1);
  }

  .error {
    margin-top: 2px;
    padding: 8px 10px;
    border-radius: 8px;
    background: rgba(220, 53, 69, 0.12);
    color: #ffb4b8;
    display: flex;
    align-items: center;
    gap: 8px;
    font-size: 0.85rem;
  }

  .error i {
    font-size: 0.9rem;
  }

  .actions {
    margin-top: 10px;
    display: flex;
    justify-content: flex-end;
    gap: 10px;
  }

  .btn {
    min-width: 92px;
    padding: 8px 14px;
    border-radius: 999px;
    border: 1px solid transparent;
    display: inline-flex;
    align-items: center;
    justify-content: center;
    gap: 6px;
    font-size: 0.9rem;
    cursor: pointer;
    transition: all 0.2s ease;
  }

  .btn.primary {
    background: var(--accent, #a8dadc);
    border-color: var(--accent, #a8dadc);
    color: var(--accent-ink, #1a2332);
    font-weight: 600;
  }

  .btn.primary:hover:not(:disabled) {
    filter: brightness(1.1);
    transform: translateY(-1px);
  }

  .btn.secondary {
    background: transparent;
    border-color: rgba(255, 255, 255, 0.3);
    color: rgba(255, 255, 255, 0.9);
  }

  .btn.secondary:hover:not(:disabled) {
    background: rgba(255, 255, 255, 0.08);
  }

  .btn:disabled {
    opacity: 0.6;
    cursor: default;
  }

  @media (max-width: 480px) {
    .modal {
      margin: 0 16px;
      padding: 20px 18px 18px;
    }
  }
</style>
