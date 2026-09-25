<script>
  import ServiceItem from './ServiceItem.svelte';

  export let service = { name: '', icon: '', items: [] };
  export let style = 'cards';
  // 单块编辑相关：各回调由上层注入，item 的原始下标由 item._itemIndex 携带
  export let canEdit = false;
  export let onEditItem = () => {};
  export let onDeleteItem = () => {};
  export let onAddItem = () => {};
  export let onEditService = () => {};
</script>

<div class="service-group {style}">
  <div class="group-header">
    <i class="{service.icon || 'fas fa-folder'}"></i>
    <h2>{service.name}</h2>

    {#if canEdit}
      <div class="group-actions">
        <button class="act-btn" on:click={onEditService} title="编辑分组" aria-label="编辑分组">
          <i class="fas fa-pen"></i>
        </button>
        <button class="act-btn" on:click={onAddItem} title="新增站点" aria-label="新增站点">
          <i class="fas fa-plus"></i>
        </button>
      </div>
    {/if}
  </div>

  <div class="items-container">
    {#each service.items || [] as item, idx}
      <ServiceItem
        {item}
        {style}
        {canEdit}
        itemIndex={item._itemIndex != null ? item._itemIndex : idx}
        onEdit={onEditItem}
        onDelete={onDeleteItem}
      />
    {/each}
  </div>
</div>

<style>
  .service-group {
    background: var(--surface, rgba(255, 255, 255, 0.05));
    border-radius: 16px;
    padding: 20px;
    backdrop-filter: blur(10px);
    border: 1px solid var(--border, rgba(255, 255, 255, 0.1));
    transition: transform 0.3s ease, box-shadow 0.3s ease;
  }

  .service-group:hover {
    transform: translateY(-4px);
    box-shadow: 0 20px 40px rgba(0, 0, 0, 0.3);
  }

  .group-header {
    display: flex;
    align-items: center;
    gap: 12px;
    margin-bottom: 16px;
    padding-bottom: 12px;
    border-bottom: 1px solid var(--border, rgba(255, 255, 255, 0.1));
  }

  .group-header i {
    font-size: 1.5rem;
    color: var(--accent, #a8dadc);
    width: 32px;
    text-align: center;
  }

  .group-header h2 {
    font-size: 1.3rem;
    font-weight: 600;
    color: var(--ink, #ffffff);
    margin: 0;
  }

  .group-actions {
    display: flex;
    align-items: center;
    gap: 6px;
    margin-left: auto;
    opacity: 0;
    transition: opacity 0.2s ease;
  }

  .service-group:hover .group-actions,
  .service-group:focus-within .group-actions {
    opacity: 1;
  }

  .act-btn {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    width: 28px;
    height: 28px;
    border-radius: 8px;
    border: 1px solid rgba(255, 255, 255, 0.18);
    background: rgba(255, 255, 255, 0.06);
    color: rgba(255, 255, 255, 0.8);
    font-size: 0.75rem;
    cursor: pointer;
    transition: all 0.2s ease;
  }

  .act-btn:hover {
    background: var(--accent-soft, rgba(45, 139, 139, 0.35));
    color: var(--accent, #a8dadc);
    border-color: var(--accent, #a8dadc);
  }

  .items-container {
    display: flex;
    flex-direction: column;
    gap: 12px;
  }

  /* List view specific styles */
  .list.service-group {
    padding: 16px;
  }

  .list .group-header {
    margin-bottom: 12px;
    padding-bottom: 8px;
  }

  .list .group-header i {
    font-size: 1.2rem;
  }

  .list .group-header h2 {
    font-size: 1.1rem;
  }

  .list .items-container {
    gap: 8px;
  }
</style>
