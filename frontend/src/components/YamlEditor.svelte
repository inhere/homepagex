<script>
  import { createEventDispatcher, onMount, onDestroy } from 'svelte';
  import IconSearch from './IconSearch.svelte';

  const dispatch = createEventDispatcher();

  export let pagePath;

  let editorValue = '';
  let loading = true;
  let saving = false;
  let error = null;

  onMount(async () => {
    await loadYaml();
    document.addEventListener('keydown', handleKeydown);
  });

  onDestroy(() => {
    document.removeEventListener('keydown', handleKeydown);
  });

  async function loadYaml() {
    try {
      const response = await fetch(`/api/page${pagePath}?op=r`, {
        credentials: 'include',
      });
      
      const data = await response.json();
      if (data.success && data.data) {
        editorValue = data.data.content;
      } else {
        throw new Error(data.error || '加载失败');
      }
    } catch (err) {
      error = err.message;
    } finally {
      loading = false;
    }
  }

  async function handleSave() {
    if (!editorValue.trim()) {
      error = '内容不能为空';
      return;
    }

    saving = true;
    error = null;

    try {
      const response = await fetch(`/api/page${pagePath}?op=w`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json'
        },
        credentials: 'include',
        body: JSON.stringify({
          content: editorValue
        })
      });

      if (!response.ok) {
        const data = await response.json();
        throw new Error(data.error || '保存失败');
      }

      dispatch('save-success');
      handleClose();
    } catch (err) {
      error = err.message;
    } finally {
      saving = false;
    }
  }

  function handleClose() {
    dispatch('close');
  }

  function handleOverlayClick(event) {
    // 仅当点击在遮罩本身（而不是对话框内容）时才关闭
    if (event.target !== event.currentTarget) {
      return;
    }
    handleClose();
  }

  function handleKeydown(event) {
    if (event.key === 'Escape') {
      handleClose();
    }
  }

  function handleIconSelect(event) {
    const iconUrl = event.detail;
    const textarea = document.querySelector('.yaml-editor');
    const start = textarea.selectionStart;
    const end = textarea.selectionEnd;
    const text = editorValue;
    
    editorValue = text.substring(0, start) + iconUrl + text.substring(end);
    
    // Reset cursor position
    setTimeout(() => {
      textarea.focus();
      const newPosition = start + iconUrl.length;
      textarea.setSelectionRange(newPosition, newPosition);
    }, 0);
  }
</script>

<div
  class="modal-overlay"
  on:click={handleOverlayClick}
>
  <div class="modal-container" role="dialog" aria-modal="true" aria-labelledby="modal-title">
    <div class="modal-header">
      <h2 class="modal-title">
        <i class="fas fa-code"></i>
        编辑页面配置
      </h2>
      <button class="close-btn" on:click={handleClose} title="关闭">
        <i class="fas fa-times"></i>
      </button>
    </div>

    <div class="modal-body">
      <div class="icon-search-section">
        <h3 class="section-title">
          <i class="fas fa-icons"></i>
          图标搜索
        </h3>
        <IconSearch on:select={handleIconSelect} />
      </div>

      <div class="editor-section">
        <h3 class="section-title">
          <i class="fas fa-file-code"></i>
          YAML 配置
        </h3>
        {#if loading}
          <div class="loading-placeholder">
            <i class="fas fa-spinner fa-spin"></i>
            <span>加载配置中...</span>
          </div>
        {:else}
          <textarea
            class="yaml-editor"
            bind:value={editorValue}
            placeholder="YAML 配置内容..."
            spellcheck="false"
          ></textarea>
        {/if}
      </div>

      {#if error}
        <div class="error-message">
          <i class="fas fa-exclamation-circle"></i>
          <span>{error}</span>
        </div>
      {/if}
    </div>

    <div class="modal-footer">
      <button class="btn btn-cancel" on:click={handleClose} disabled={saving}>
        <i class="fas fa-times"></i>
        取消
      </button>
      <button class="btn btn-save" on:click={handleSave} disabled={saving || loading}>
        {#if saving}
          <i class="fas fa-spinner fa-spin"></i>
          保存中...
        {:else}
          <i class="fas fa-save"></i>
          保存
        {/if}
      </button>
    </div>
  </div>
</div>

<style>
  .modal-overlay {
    /* 让按钮表现得像一个全屏容器，而不是传统按钮 */
    position: fixed;
    top: 0;
    left: 0;
    right: 0;
    bottom: 0;
    background: radial-gradient(
      circle at top,
      var(--theme-secondary-rgba, rgba(255, 255, 255, 0.05)),
      rgba(0, 0, 0, 0.75)
    );
    backdrop-filter: blur(8px);
    display: flex;
    align-items: center;
    justify-content: center;
    z-index: 9999;
    padding: 20px;
    border: none;
    outline: none;
    cursor: default;
  }

  .modal-overlay:focus {
    outline: none;
  }

  .modal-container {
    background: linear-gradient(
      135deg,
      var(--theme-primary-rgba, rgba(20, 20, 40, 0.98)),
      var(--theme-secondary-rgba, rgba(10, 10, 25, 0.98))
    );
    border: 1px solid rgba(255, 255, 255, 0.1);
    border-radius: 12px;
    width: 100%;
    max-width: 1200px;
    height: 95vh;
    max-height: 95vh;
    display: flex;
    flex-direction: column;
    box-shadow: 0 20px 60px rgba(0, 0, 0, 0.5);
  }

  .modal-header {
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding: 20px 24px;
    border-bottom: 1px solid rgba(255, 255, 255, 0.1);
  }

  .modal-title {
    display: flex;
    align-items: center;
    gap: 10px;
    margin: 0;
    font-size: 1.3rem;
    font-weight: 600;
    color: #e4e4e4;
  }

  .modal-title i {
    color: var(--theme-primary, #4a9eff);
  }

  .close-btn {
    display: flex;
    align-items: center;
    justify-content: center;
    width: 36px;
    height: 36px;
    background: rgba(255, 255, 255, 0.05);
    border: 1px solid rgba(255, 255, 255, 0.1);
    border-radius: 8px;
    color: rgba(255, 255, 255, 0.7);
    cursor: pointer;
    transition: all 0.2s ease;
  }

  .close-btn:hover {
    background: rgba(255, 107, 107, 0.2);
    border-color: rgba(255, 107, 107, 0.3);
    color: #ff6b6b;
  }

  .modal-body {
    flex: 1;
    display: flex;
    flex-direction: column;
    gap: 16px;
    padding: 20px 24px;
    overflow: hidden;
  }

  .icon-search-section {
    flex-shrink: 0;
    max-height: 180px;
    display: flex;
    flex-direction: column;
  }

  .editor-section {
    flex: 1;
    display: flex;
    flex-direction: column;
    min-height: 0;
    overflow: hidden;
  }

  .section-title {
    display: flex;
    align-items: center;
    gap: 8px;
    margin: 0 0 12px 0;
    font-size: 0.95rem;
    font-weight: 500;
    color: rgba(255, 255, 255, 0.8);
  }

  .section-title i {
    color: var(--theme-primary, #4a9eff);
    font-size: 0.85rem;
  }

  .icon-search-section .section-title {
    margin-bottom: 12px;
  }

  .icon-search-section :global(.icon-search) {
    flex: 1;
    overflow: hidden;
  }

  .yaml-editor {
    flex: 1;
    width: 100%;
    min-height: 200px;
    padding: 16px;
    background: rgba(0, 0, 0, 0.3);
    border: 1px solid rgba(255, 255, 255, 0.1);
    border-radius: 8px;
    color: #e4e4e4;
    font-family: 'Consolas', 'Monaco', 'Courier New', monospace;
    font-size: 0.9rem;
    line-height: 1.6;
    resize: none;
    outline: none;
    transition: border-color 0.3s ease;
    white-space: pre-wrap;
    overflow-x: auto;
    overflow-y: auto;
  }

  .yaml-editor:focus {
    border-color: var(--theme-primary, #4a9eff);
  }

  .yaml-editor::placeholder {
    color: rgba(255, 255, 255, 0.3);
  }

  .loading-placeholder {
    flex: 1;
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    gap: 12px;
    min-height: 200px;
    background: rgba(0, 0, 0, 0.3);
    border: 1px solid rgba(255, 255, 255, 0.1);
    border-radius: 8px;
    color: rgba(255, 255, 255, 0.5);
  }

  .loading-placeholder i {
    font-size: 1.5rem;
  }

  .error-message {
    display: flex;
    align-items: center;
    gap: 8px;
    padding: 12px 16px;
    background: rgba(255, 107, 107, 0.1);
    border: 1px solid rgba(255, 107, 107, 0.3);
    border-radius: 8px;
    color: #ff6b6b;
    font-size: 0.9rem;
  }

  .modal-footer {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 12px;
    padding: 20px 24px;
    border-top: 1px solid rgba(255, 255, 255, 0.1);
  }

  .btn {
    display: flex;
    align-items: center;
    gap: 8px;
    padding: 10px 20px;
    border: 1px solid rgba(255, 255, 255, 0.2);
    border-radius: 8px;
    font-size: 0.9rem;
    font-weight: 500;
    cursor: pointer;
    transition: all 0.3s ease;
  }

  .btn:disabled {
    opacity: 0.5;
    cursor: not-allowed;
  }

  .btn-cancel {
    background: rgba(255, 255, 255, 0.05);
    color: rgba(255, 255, 255, 0.8);
  }

  .btn-cancel:hover:not(:disabled) {
    background: rgba(255, 255, 255, 0.1);
  }

  .btn-save {
    background: var(--theme-primary, #4a9eff);
    border-color: var(--theme-primary, #4a9eff);
    color: white;
  }

  .btn-save:hover:not(:disabled) {
    background: var(--theme-primary-dark, #3a8eef);
    transform: translateY(-2px);
    box-shadow: 0 4px 12px rgba(74, 158, 255, 0.3);
  }

  @media (max-width: 768px) {
    .modal-overlay {
      padding: 10px;
    }

    .modal-container {
      max-height: 98vh;
    }

    .modal-header,
    .modal-body,
    .modal-footer {
      padding: 16px;
    }

    .modal-title {
      font-size: 1.1rem;
    }

    .icon-search-section {
      max-height: 150px;
    }
  }
</style>
