<script>
  import { createEventDispatcher, onMount } from 'svelte';
  import yaml from 'js-yaml';
  import IconSearch from './IconSearch.svelte';

  const dispatch = createEventDispatcher();

  // kind: item | service ; mode: edit | insert
  export let pagePath = '/';
  export let kind = 'item';
  export let mode = 'edit';
  export let serviceIndex = 0;
  export let index = 0;

  let loading = true;
  let saving = false;
  let error = '';
  let view = 'form';
  let touched = false;

  // originalText 服务端返回的块原始文本，用于「只改必要行」的保存
  let originalText = '';
  let sourceText = '';

  let itemForm = { name: '', url: '', logo: '', subtitle: '', tags: '', target: '_blank' };
  let groupForm = { name: '', icon: '' };

  $: isItem = kind === 'item';
  $: modalTitle = `${mode === 'insert' ? '新增' : '编辑'}${isItem ? '站点' : '分组'}`;

  let overlayEl = null;

  onMount(loadBlock);

  async function loadBlock() {
    if (mode === 'insert') {
      loading = false;
      return;
    }

    try {
      const resp = await fetch(`/api/page${pagePath}?op=blocks`, { credentials: 'include' });
      const payload = await resp.json().catch(() => null);
      if (!resp.ok || !payload || !payload.success) {
        throw new Error((payload && payload.error) || '加载配置失败');
      }

      const blocks = (payload.data && payload.data.blocks) || [];
      const target = blocks.find(
        (b) =>
          b.kind === kind &&
          b.index === index &&
          (kind !== 'item' || b.service_index === serviceIndex)
      );

      if (!target) {
        throw new Error('未找到对应的配置块');
      }

      originalText = target.yaml || '';
      sourceText = originalText;

      if (isItem) {
        const v = target.value || {};
        itemForm = {
          name: v.name || '',
          url: v.url || '',
          logo: v.logo || '',
          subtitle: v.subtitle || '',
          tags: (v.tags || []).join(', '),
          target: v.target || '_blank',
        };
      } else {
        groupForm = { name: target.name || '', icon: target.icon || '' };
      }
    } catch (e) {
      error = e.message;
    } finally {
      loading = false;
    }
  }

  function parseTags() {
    return itemForm.tags
      .split(/[,，\s]+/)
      .map((t) => t.trim())
      .filter(Boolean);
  }

  // yamlScalar 输出可直接放进 YAML 的标量（必要时加引号）
  function yamlScalar(value) {
    const s = String(value == null ? '' : value);
    if (s === '') {
      return '""';
    }
    if (
      /^[^\s#:[\]{},&*!|>'"%@`]+$/.test(s) &&
      !/^(true|false|null|~|[-+]?(\d+|\d*\.\d+))$/i.test(s)
    ) {
      return s;
    }
    return JSON.stringify(s);
  }

  // splitComment 拆分出行尾注释（忽略引号内的 #）
  function splitComment(raw) {
    let inSingle = false;
    let inDouble = false;
    for (let i = 0; i < raw.length; i++) {
      const c = raw[i];
      if (c === "'" && !inDouble) {
        inSingle = !inSingle;
      } else if (c === '"' && !inSingle) {
        inDouble = !inDouble;
      } else if (c === '#' && !inSingle && !inDouble && i > 0 && /\s/.test(raw[i - 1])) {
        return [raw.slice(0, i), raw.slice(i - 1)];
      }
    }
    return [raw, ''];
  }

  // patchBlockText 只在块的原始文本上替换涉及的行，其余行与注释原样保留
  function patchBlockText(base, updates) {
    const seen = new Set();
    const kept = [];

    for (const line of base.split('\n')) {
      const m = line.match(/^(\s*)([A-Za-z_][\w-]*):(\s*)(.*)$/);
      if (!m || !Object.prototype.hasOwnProperty.call(updates, m[2])) {
        kept.push(line);
        continue;
      }

      const key = m[2];
      const value = updates[key];
      seen.add(key);

      if (value === null || value === undefined || value === '') {
        continue; // 字段被清空 → 删除该行
      }
      const [, comment] = splitComment(m[4]);
      kept.push(`${m[1]}${key}: ${value}${comment}`);
    }

    let result = kept.join('\n');

    // 原本没有的字段：插到首行之后，按块自身的子字段缩进对齐
    const missing = Object.keys(updates).filter(
      (k) => !seen.has(k) && updates[k] !== null && updates[k] !== undefined && updates[k] !== ''
    );
    if (missing.length) {
      const lines = result.split('\n');
      const marker = lines[0].match(/^(\s*)-/);
      const pad = ' '.repeat((marker ? marker[1].length : 0) + 2);
      lines.splice(1, 0, ...missing.map((k) => `${pad}${k}: ${updates[k]}`));
      result = lines.join('\n');
    }

    return result;
  }

  function buildItemBlock() {
    const lines = [`- name: ${yamlScalar(itemForm.name)}`];
    if (itemForm.logo) {
      lines.push(`  logo: ${yamlScalar(itemForm.logo)}`);
    }
    if (itemForm.subtitle) {
      lines.push(`  subtitle: ${yamlScalar(itemForm.subtitle)}`);
    }
    const tags = parseTags();
    if (tags.length) {
      lines.push(`  tags: [${tags.map(yamlScalar).join(', ')}]`);
    }
    if (itemForm.url) {
      lines.push(`  url: ${yamlScalar(itemForm.url)}`);
    }
    if (itemForm.target) {
      lines.push(`  target: ${yamlScalar(itemForm.target)}`);
    }
    return lines.join('\n');
  }

  function buildGroupBlock() {
    const lines = [`- name: ${yamlScalar(groupForm.name)}`];
    if (groupForm.icon) {
      lines.push(`  icon: ${yamlScalar(groupForm.icon)}`);
    }
    lines.push('  items: []');
    return lines.join('\n');
  }

  function itemUpdates() {
    const tags = parseTags();
    return {
      name: yamlScalar(itemForm.name),
      logo: itemForm.logo ? yamlScalar(itemForm.logo) : '',
      subtitle: itemForm.subtitle ? yamlScalar(itemForm.subtitle) : '',
      tags: tags.length ? `[${tags.map(yamlScalar).join(', ')}]` : '',
      url: itemForm.url ? yamlScalar(itemForm.url) : '',
      target: itemForm.target ? yamlScalar(itemForm.target) : '',
    };
  }

  function groupUpdates() {
    return {
      name: yamlScalar(groupForm.name),
      icon: groupForm.icon ? yamlScalar(groupForm.icon) : '',
    };
  }

  // 表单视图下「当前会写成什么」，切到源码时用它预览
  function currentBlockYaml() {
    if (isItem) {
      return mode === 'insert'
        ? buildItemBlock()
        : patchBlockText(originalText, itemUpdates());
    }
    return mode === 'insert' ? buildGroupBlock() : patchBlockText(originalText, groupUpdates());
  }

  function switchToSource() {
    sourceText = currentBlockYaml();
    view = 'source';
    error = '';
  }

  function switchToForm() {
    try {
      const loaded = yaml.load(sourceText);
      const obj = Array.isArray(loaded) ? loaded[0] : loaded;
      if (!obj || typeof obj !== 'object') {
        throw new Error('源码必须是一个映射（key: value）');
      }

      if (isItem) {
        itemForm = {
          name: obj.name || '',
          url: obj.url || '',
          logo: obj.logo || '',
          subtitle: obj.subtitle || '',
          tags: Array.isArray(obj.tags) ? obj.tags.join(', ') : obj.tags || '',
          target: obj.target || '_blank',
        };
      } else {
        groupForm = { name: obj.name || '', icon: obj.icon || '' };
      }
      view = 'form';
      error = '';
    } catch (e) {
      error = '源码解析失败：' + e.message;
    }
  }

  function toggleView() {
    if (view === 'form') {
      switchToSource();
    } else {
      switchToForm();
    }
  }

  function validate() {
    if (isItem) {
      if (!itemForm.name.trim()) {
        return '站点名称不能为空';
      }
      if (mode === 'insert' || itemForm.url.trim()) {
        if (!/^(https?:\/\/|\/)/.test(itemForm.url.trim())) {
          return 'URL 需要以 http(s):// 或 / 开头';
        }
      }
    } else if (!groupForm.name.trim()) {
      return '分组名称不能为空';
    }
    if (view === 'source' && !sourceText.trim()) {
      return '内容不能为空';
    }
    return '';
  }

  async function handleSave() {
    if (view === 'form' || view === 'source') {
      const msg = validate();
      if (msg) {
        error = msg;
        return;
      }
    }

    saving = true;
    error = '';

    try {
      const body = {
        kind,
        action: mode === 'insert' ? 'insert' : 'update',
        service_index: Number(serviceIndex),
        index: Number(index),
        yaml: view === 'source' ? sourceText : currentBlockYaml(),
      };

      const resp = await fetch(`/api/page${pagePath}?op=block`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        credentials: 'include',
        body: JSON.stringify(body),
      });
      const payload = await resp.json().catch(() => null);
      if (!resp.ok || !payload || !payload.success) {
        throw new Error((payload && payload.error) || '保存失败');
      }

      touched = false;
      dispatch('saved');
      dispatch('close');
    } catch (e) {
      error = e.message;
    } finally {
      saving = false;
    }
  }

  async function handleDelete() {
    if (!confirm('确定删除这个' + (isItem ? '站点' : '分组') + '吗？')) {
      return;
    }

    saving = true;
    error = '';

    try {
      const resp = await fetch(`/api/page${pagePath}?op=block`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        credentials: 'include',
        body: JSON.stringify({
          kind,
          action: 'delete',
          service_index: Number(serviceIndex),
          index: Number(index),
        }),
      });
      const payload = await resp.json().catch(() => null);
      if (!resp.ok || !payload || !payload.success) {
        throw new Error((payload && payload.error) || '删除失败');
      }

      touched = false;
      dispatch('saved');
      dispatch('close');
    } catch (e) {
      error = e.message;
    } finally {
      saving = false;
    }
  }

  function markTouched() {
    touched = true;
  }

  function close() {
    if (saving) {
      return;
    }
    if (touched && !confirm('有未保存的修改，确定关闭吗？')) {
      return;
    }
    dispatch('close');
  }

  function handleWindowClick(event) {
    // 仅点击遮罩自身（而不是对话框内容）时关闭
    if (event.target === overlayEl) {
      close();
    }
  }

  function handleKeydown(event) {
    if (event.key === 'Escape') {
      close();
    } else if ((event.ctrlKey || event.metaKey) && event.key === 's') {
      event.preventDefault();
      handleSave();
    }
  }

  function handleIconSelect(event) {
    itemForm.logo = event.detail;
    markTouched();
  }
</script>

<svelte:window on:click={handleWindowClick} on:keydown={handleKeydown} />

<div class="modal-overlay" bind:this={overlayEl}>
  <div class="modal-box" role="dialog" aria-modal="true" aria-label={modalTitle}>
    <div class="modal-head">
      <h2>
        <i class="fas {isItem ? 'fa-link' : 'fa-folder'}"></i>
        {modalTitle}
      </h2>
      <div class="head-actions">
        <button class="view-toggle" type="button" on:click={toggleView} disabled={loading || saving}>
          <i class="fas {view === 'form' ? 'fa-code' : 'fa-list-alt'}"></i>
          {view === 'form' ? '源码' : '表单'}
        </button>
        <button class="icon-btn" type="button" on:click={close} title="关闭">
          <i class="fas fa-times"></i>
        </button>
      </div>
    </div>

    <div class="modal-body">
      {#if loading}
        <div class="hint"><i class="fas fa-spinner fa-spin"></i> 加载中...</div>
      {:else if view === 'form'}
        {#if isItem}
          <div class="grid">
            <label class="field">
              <span>名称 *</span>
              <input type="text" bind:value={itemForm.name} on:input={markTouched} placeholder="Plex" />
            </label>
            <label class="field">
              <span>URL *</span>
              <input type="text" bind:value={itemForm.url} on:input={markTouched} placeholder="https://plex.example.com" />
            </label>
            <label class="field">
              <span>副标题</span>
              <input type="text" bind:value={itemForm.subtitle} on:input={markTouched} placeholder="Media server" />
            </label>
            <label class="field">
              <span>标签（逗号或空格分隔）</span>
              <input type="text" bind:value={itemForm.tags} on:input={markTouched} placeholder="app, media" />
            </label>
            <label class="field">
              <span>打开方式</span>
              <select bind:value={itemForm.target} on:change={markTouched}>
                <option value="_blank">新窗口 (_blank)</option>
                <option value="_self">当前窗口 (_self)</option>
              </select>
            </label>
            <label class="field wide">
              <span>图标</span>
              <input type="text" bind:value={itemForm.logo} on:input={markTouched} placeholder="icons-local/dashboard-icons/png/plex.png" />
            </label>
          </div>

          <div class="icon-picker">
            <IconSearch on:select={handleIconSelect} />
          </div>
        {:else}
          <div class="grid">
            <label class="field">
              <span>分组名称 *</span>
              <input type="text" bind:value={groupForm.name} on:input={markTouched} placeholder="Media" />
            </label>
            <label class="field">
              <span>分组图标（FontAwesome，如 fas fa-play-circle）</span>
              <input type="text" bind:value={groupForm.icon} on:input={markTouched} placeholder="fas fa-play-circle" />
            </label>
          </div>
          {#if mode === 'insert'}
            <p class="tip">新建分组后，可以在分组标题处点「+」添加站点。</p>
          {/if}
        {/if}
      {:else}
        <textarea
          class="source-editor"
          bind:value={sourceText}
          on:input={markTouched}
          spellcheck="false"
          placeholder="YAML 片段..."
        ></textarea>
        <p class="tip">
          只改动这一个块；其余内容（注释、空行、其它条目）不会被触碰。
        </p>
      {/if}

      {#if error}
        <div class="error"><i class="fas fa-exclamation-circle"></i> {error}</div>
      {/if}
    </div>

    <div class="modal-foot">
      {#if mode === 'edit'}
        <button class="btn danger" type="button" on:click={handleDelete} disabled={saving}>
          <i class="fas fa-trash"></i> 删除
        </button>
      {:else}
        <span></span>
      {/if}
      <div class="foot-right">
        <button class="btn" type="button" on:click={close} disabled={saving}>取消</button>
        <button class="btn primary" type="button" on:click={handleSave} disabled={saving || loading}>
          {#if saving}
            <i class="fas fa-spinner fa-spin"></i> 保存中...
          {:else}
            <i class="fas fa-save"></i> 保存
          {/if}
        </button>
      </div>
    </div>
  </div>
</div>

<style>
  .modal-overlay {
    position: fixed;
    inset: 0;
    z-index: 2000;
    display: flex;
    align-items: center;
    justify-content: center;
    padding: 20px;
    background: radial-gradient(circle at top, var(--accent-soft, rgba(45, 139, 139, 0.4)), rgba(0, 0, 0, 0.72));
    backdrop-filter: blur(6px);
  }

  .modal-box {
    width: 100%;
    max-width: 720px;
    max-height: 92vh;
    display: flex;
    flex-direction: column;
    border-radius: 14px;
    border: 1px solid rgba(255, 255, 255, 0.12);
    background: linear-gradient(150deg, rgba(16, 22, 36, 0.98), rgba(8, 12, 22, 0.98));
    box-shadow: 0 24px 60px rgba(0, 0, 0, 0.55);
    color: #f1f5f9;
  }

  .modal-head {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 12px;
    padding: 16px 20px;
    border-bottom: 1px solid rgba(255, 255, 255, 0.08);
  }

  .modal-head h2 {
    display: flex;
    align-items: center;
    gap: 10px;
    margin: 0;
    font-size: 1.1rem;
    font-weight: 600;
  }

  .modal-head h2 i {
    color: var(--accent, #a8dadc);
  }

  .head-actions {
    display: flex;
    align-items: center;
    gap: 8px;
  }

  .view-toggle,
  .icon-btn {
    display: inline-flex;
    align-items: center;
    gap: 6px;
    padding: 7px 12px;
    border-radius: 8px;
    border: 1px solid rgba(255, 255, 255, 0.18);
    background: rgba(255, 255, 255, 0.06);
    color: #e2e8f0;
    font-size: 0.85rem;
    cursor: pointer;
    transition: all 0.2s ease;
  }

  .view-toggle:hover:not(:disabled),
  .icon-btn:hover {
    background: rgba(255, 255, 255, 0.14);
  }

  .view-toggle:disabled {
    opacity: 0.5;
    cursor: default;
  }

  .modal-body {
    flex: 1;
    min-height: 0;
    overflow-y: auto;
    padding: 18px 20px;
    display: flex;
    flex-direction: column;
    gap: 14px;
  }

  .grid {
    display: grid;
    grid-template-columns: repeat(2, minmax(0, 1fr));
    gap: 12px 16px;
  }

  .field {
    display: flex;
    flex-direction: column;
    gap: 6px;
    font-size: 0.85rem;
    color: rgba(255, 255, 255, 0.75);
  }

  .field.wide {
    grid-column: 1 / -1;
  }

  .field input,
  .field select {
    padding: 9px 10px;
    border-radius: 8px;
    border: 1px solid rgba(255, 255, 255, 0.2);
    background: rgba(4, 8, 18, 0.85);
    color: #f1f5f9;
    font-size: 0.9rem;
    outline: none;
    transition: border 0.2s, box-shadow 0.2s;
  }

  .field input:focus,
  .field select:focus {
    border-color: var(--accent, #a8dadc);
    box-shadow: 0 0 0 1px var(--accent-soft, rgba(168, 218, 220, 0.4));
  }

  .icon-picker {
    border-top: 1px solid rgba(255, 255, 255, 0.08);
    padding-top: 12px;
    max-height: 210px;
    display: flex;
    flex-direction: column;
  }

  .source-editor {
    width: 100%;
    min-height: 260px;
    flex: 1;
    padding: 12px 14px;
    border-radius: 10px;
    border: 1px solid rgba(255, 255, 255, 0.14);
    background: rgba(0, 0, 0, 0.35);
    color: #e2e8f0;
    font-family: 'Consolas', 'Monaco', 'Courier New', monospace;
    font-size: 0.85rem;
    line-height: 1.55;
    resize: vertical;
    outline: none;
  }

  .source-editor:focus {
    border-color: var(--accent, #a8dadc);
  }

  .tip,
  .hint {
    margin: 0;
    font-size: 0.8rem;
    color: rgba(255, 255, 255, 0.55);
  }

  .hint {
    display: flex;
    align-items: center;
    gap: 8px;
    padding: 20px 0;
    font-size: 0.9rem;
  }

  .error {
    display: flex;
    align-items: center;
    gap: 8px;
    padding: 10px 12px;
    border-radius: 8px;
    background: rgba(220, 53, 69, 0.14);
    border: 1px solid rgba(220, 53, 69, 0.35);
    color: #ffb4b8;
    font-size: 0.85rem;
  }

  .modal-foot {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 12px;
    padding: 14px 20px;
    border-top: 1px solid rgba(255, 255, 255, 0.08);
  }

  .foot-right {
    display: flex;
    align-items: center;
    gap: 10px;
  }

  .btn {
    display: inline-flex;
    align-items: center;
    gap: 6px;
    padding: 9px 16px;
    border-radius: 8px;
    border: 1px solid rgba(255, 255, 255, 0.2);
    background: rgba(255, 255, 255, 0.06);
    color: #e2e8f0;
    font-size: 0.88rem;
    cursor: pointer;
    transition: all 0.2s ease;
  }

  .btn:hover:not(:disabled) {
    background: rgba(255, 255, 255, 0.14);
  }

  .btn:disabled {
    opacity: 0.55;
    cursor: default;
  }

  .btn.primary {
    background: var(--accent, #2d8b8b);
    border-color: var(--accent, #2d8b8b);
    color: #08131f;
    font-weight: 600;
  }

  .btn.danger {
    border-color: rgba(220, 53, 69, 0.5);
    color: #ff9aa2;
    background: transparent;
  }

  .btn.danger:hover:not(:disabled) {
    background: rgba(220, 53, 69, 0.16);
  }

  @media (max-width: 640px) {
    .grid {
      grid-template-columns: 1fr;
    }
  }
</style>
