// sidepanel.js - 侧边栏逻辑（批量采集版）
(function () {
  'use strict';

  const els = {
    configToggle: document.getElementById('configToggle'),
    configBody: document.getElementById('configBody'),
    configArrow: document.querySelector('.config-toggle .arrow'),
    apiBaseUrl: document.getElementById('apiBaseUrl'),
    apiToken: document.getElementById('apiToken'),
    userId: document.getElementById('userId'),
    defaultCategoryId: document.getElementById('defaultCategoryId'),
    saveConfigBtn: document.getElementById('saveConfigBtn'),
    configStatus: document.getElementById('configStatus'),

    batchStats: document.getElementById('batchStats'),
    totalCount: document.getElementById('totalCount'),
    submittedCount: document.getElementById('submittedCount'),
    pendingCount: document.getElementById('pendingCount'),

    batchActionsBar: document.getElementById('batchActionsBar'),
    checkAll: document.getElementById('checkAll'),
    batchSubmitBtn: document.getElementById('batchSubmitBtn'),
    clearSubmittedBtn: document.getElementById('clearSubmittedBtn'),
    clearAllBtn: document.getElementById('clearAllBtn'),

    batchList: document.getElementById('batchList'),
    emptyState: document.getElementById('emptyState'),

    editPanel: document.getElementById('editPanel'),
    closeEditPanel: document.getElementById('closeEditPanel'),
    editId: document.getElementById('editId'),
    editTitle: document.getElementById('editTitle'),
    editContent: document.getElementById('editContent'),
    editContentEn: document.getElementById('editContentEn'),
    editModel: document.getElementById('editModel'),
    editMediaType: document.getElementById('editMediaType'),
    editTags: document.getElementById('editTags'),
    editCategoryId: document.getElementById('editCategoryId'),
    editCoverImage: document.getElementById('editCoverImage'),
    editSourceUrl: document.getElementById('editSourceUrl'),
    saveEditBtn: document.getElementById('saveEditBtn'),
    cancelEditBtn: document.getElementById('cancelEditBtn'),

    globalStatus: document.getElementById('globalStatus'),
  };

  let config = {};
  let batchData = [];
  let categoriesCache = [];
  let editingId = null;

  // 初始化
  async function init() {
    const configRes = await chrome.runtime.sendMessage({ action: 'getConfig' });
    if (configRes.success) {
      config = configRes.data || {};
      els.apiBaseUrl.value = config.apiBaseUrl || '';
      els.apiToken.value = config.apiToken || '';
      if (els.userId) els.userId.value = config.userId || '';
      els.defaultCategoryId.value = config.defaultCategoryId || '';
    }

    await loadCategories();
    await loadBatchData();

    els.configToggle.addEventListener('click', toggleConfig);
    els.saveConfigBtn.addEventListener('click', saveConfig);
    els.checkAll.addEventListener('change', onCheckAll);
    els.batchSubmitBtn.addEventListener('click', batchSubmit);
    els.clearSubmittedBtn.addEventListener('click', clearSubmitted);
    els.clearAllBtn.addEventListener('click', clearAll);
    els.closeEditPanel.addEventListener('click', closeEdit);
    els.saveEditBtn.addEventListener('click', saveEdit);
    els.cancelEditBtn.addEventListener('click', closeEdit);

    chrome.runtime.onMessage.addListener((message) => {
      if (message.action === 'batchUpdated') {
        batchData = message.batch || [];
        renderBatchList();
      }
    });
  }

  // 加载分类列表
  async function loadCategories() {
    if (!config.apiBaseUrl) return;
    try {
      const res = await chrome.runtime.sendMessage({
        action: 'apiRequest',
        method: 'GET',
        path: '/prompt-category/all',
        userId: config.userId
      });
      if (res.success && res.data && res.data.success && Array.isArray(res.data.data)) {
        categoriesCache = res.data.data;
      }
    } catch (e) {}
  }

  // 根据模型名称匹配分类 ID
  function matchCategoryId(modelName) {
    if (!modelName || categoriesCache.length === 0) return 0;
    const normalized = modelName.trim().toLowerCase();
    for (const cat of categoriesCache) {
      if (cat.name && cat.name.trim().toLowerCase() === normalized) return cat.id;
    }
    for (const cat of categoriesCache) {
      const catName = (cat.name || '').trim().toLowerCase();
      if (catName.includes(normalized) || normalized.includes(catName)) return cat.id;
    }
    return 0;
  }

  // 加载批量数据
  async function loadBatchData() {
    const res = await chrome.runtime.sendMessage({ action: 'getBatchData' });
    if (res.success) {
      batchData = res.batch || [];
      renderBatchList();
    }
  }

  // 渲染批量列表
  function renderBatchList() {
    const total = batchData.length;
    const submitted = batchData.filter(i => i.submitted).length;
    const pending = total - submitted;

    els.totalCount.textContent = total;
    els.submittedCount.textContent = submitted;
    els.pendingCount.textContent = pending;

    if (total === 0) {
      els.batchStats.style.display = 'none';
      els.batchActionsBar.style.display = 'none';
      els.batchList.innerHTML = '';
      els.batchList.appendChild(els.emptyState);
      els.emptyState.style.display = 'block';
      return;
    }

    els.batchStats.style.display = 'flex';
    els.batchActionsBar.style.display = 'flex';
    els.emptyState.style.display = 'none';

    els.batchList.innerHTML = '';
    batchData.forEach(item => {
      const card = createBatchItemCard(item);
      els.batchList.appendChild(card);
    });

    updateCheckAllState();
  }

  // 创建单条卡片
  function createBatchItemCard(item) {
    const div = document.createElement('div');
    div.className = 'batch-item' + (item.submitted ? ' submitted' : '') + (item.error ? ' error' : '');
    div.dataset.id = item.id;

    const tags = parseTagsToArray(item.tags);
    const tagsHtml = tags.slice(0, 5).map(t => `<span class="tag-pill">${escapeHtml(t)}</span>`).join('');

    div.innerHTML = `
      <div class="batch-item-header">
        <input type="checkbox" class="item-check" data-id="${item.id}" ${item.submitted ? 'disabled' : ''}>
        <div class="batch-item-info">
          <div class="batch-item-title">${escapeHtml(item.title || '无标题')}</div>
          <div class="batch-item-meta">
            <span class="model-badge">${escapeHtml(item.model || '未知模型')}</span>
            ${tagsHtml}
          </div>
        </div>
        <div class="batch-item-status">
          ${item.submitted ? '<span class="status-badge success">✅ 已提交</span>' : item.error ? `<span class="status-badge error" title="${escapeHtml(item.error)}">❌ 失败</span>` : '<span class="status-badge pending">⏳ 待提交</span>'}
        </div>
      </div>
      <div class="batch-item-actions">
        <button class="btn-text" data-action="edit" data-id="${item.id}">✏️ 编辑</button>
        <button class="btn-text" data-action="submit-one" data-id="${item.id}" ${item.submitted ? 'disabled' : ''}>⬆️ 提交</button>
        <button class="btn-text danger" data-action="delete" data-id="${item.id}">🗑️ 删除</button>
      </div>
    `;

    div.querySelector('[data-action="edit"]').addEventListener('click', () => openEdit(item));
    div.querySelector('[data-action="submit-one"]').addEventListener('click', () => submitOne(item.id));
    div.querySelector('[data-action="delete"]').addEventListener('click', () => deleteOne(item.id));
    div.querySelector('.item-check').addEventListener('change', updateCheckAllState);

    return div;
  }

  // 全选/全不选
  function onCheckAll() {
    const checked = els.checkAll.checked;
    document.querySelectorAll('.item-check:not(:disabled)').forEach(cb => {
      cb.checked = checked;
    });
  }

  function updateCheckAllState() {
    const checks = document.querySelectorAll('.item-check:not(:disabled)');
    if (checks.length === 0) {
      els.checkAll.checked = false;
      els.checkAll.indeterminate = false;
      return;
    }
    const checked = document.querySelectorAll('.item-check:checked');
    els.checkAll.checked = checked.length === checks.length;
    els.checkAll.indeterminate = checked.length > 0 && checked.length < checks.length;
  }

  // 获取选中的 ID
  function getSelectedIds() {
    return Array.from(document.querySelectorAll('.item-check:checked')).map(cb => cb.dataset.id);
  }

  // 批量提交
  async function batchSubmit() {
    const ids = getSelectedIds();
    if (ids.length === 0) {
      showGlobalStatus('❌ 请先勾选要提交的提示词', 'error');
      return;
    }
    if (!config.apiBaseUrl) {
      showGlobalStatus('❌ 请先配置 API Base URL', 'error');
      toggleConfig();
      return;
    }

    els.batchSubmitBtn.disabled = true;
    els.batchSubmitBtn.textContent = '提交中...';
    showGlobalStatus(`⏳ 正在提交 ${ids.length} 条...`, 'info');

    let successCount = 0;
    let failCount = 0;

    for (const id of ids) {
      const item = batchData.find(i => i.id === id);
      if (!item || item.submitted) continue;

      const ok = await submitItem(item);
      if (ok) {
        successCount++;
        item.submitted = true;
        item.error = '';
      } else {
        failCount++;
      }
      await chrome.runtime.sendMessage({ action: 'updateBatchItem', id, updates: { submitted: item.submitted, error: item.error } });
      renderBatchList();
    }

    els.batchSubmitBtn.disabled = false;
    els.batchSubmitBtn.innerHTML = '<span class="btn-icon">⬆️</span> 批量提交';

    if (failCount === 0) {
      showGlobalStatus(`✅ 全部提交成功！${successCount} 条已入库`, 'success');
    } else {
      showGlobalStatus(`⚠️ 成功 ${successCount} 条，失败 ${failCount} 条`, 'error');
    }
  }

  // 单条提交
  async function submitOne(id) {
    const item = batchData.find(i => i.id === id);
    if (!item || item.submitted) return;
    if (!config.apiBaseUrl) {
      showGlobalStatus('❌ 请先配置 API Base URL', 'error');
      toggleConfig();
      return;
    }

    showGlobalStatus(`⏳ 正在提交「${item.title || '无标题'}」...`, 'info');
    const ok = await submitItem(item);
    if (ok) {
      item.submitted = true;
      item.error = '';
      showGlobalStatus(`✅ 「${item.title || ''}」提交成功`, 'success');
    }
    await chrome.runtime.sendMessage({ action: 'updateBatchItem', id, updates: { submitted: item.submitted, error: item.error } });
    renderBatchList();
  }

  // 执行提交
  async function submitItem(item) {
    const tagsStr = item.tags || '[]';
    const payload = {
      title: item.title || '',
      content: item.content || item.content_en || '',
      content_en: item.content_en || '',
      description: item.description || item.title || '',
      model: item.model || '',
      media_type: item.media_type || 'image',
      tags: tagsStr,
      category_id: parseInt(item.category_id) || 0,
      cover_image_url: item.cover_image_url || '',
      status: 1,
      sort_order: 0,
      is_premium: false,
      unlock_cost: 0,
      author: '采集导入',
      i18n: '{}',
      seo_i18n: '{}'
    };

    try {
      const res = await chrome.runtime.sendMessage({
        action: 'apiRequest',
        method: 'POST',
        path: '/prompt/',
        body: payload,
        userId: config.userId
      });

      if (res.success && res.data && res.data.success) {
        return true;
      } else {
        item.error = res.data?.message || res.message || '提交失败';
        return false;
      }
    } catch (err) {
      item.error = err.message || '网络错误';
      return false;
    }
  }

  // 删除单条
  async function deleteOne(id) {
    await chrome.runtime.sendMessage({ action: 'removeBatchItem', id });
    batchData = batchData.filter(i => i.id !== id);
    renderBatchList();
  }

  // 清空已提交
  async function clearSubmitted() {
    await chrome.runtime.sendMessage({ action: 'clearSubmittedBatch' });
    batchData = batchData.filter(i => !i.submitted);
    renderBatchList();
    showGlobalStatus('✅ 已清空已提交的提示词', 'success');
  }

  // 清空全部
  async function clearAll() {
    if (!confirm('确定要清空所有采集的数据吗？')) return;
    await chrome.runtime.sendMessage({ action: 'clearBatchData' });
    batchData = [];
    renderBatchList();
    showGlobalStatus('✅ 已清空全部', 'success');
  }

  // 打开编辑面板
  function openEdit(item) {
    editingId = item.id;
    els.editId.value = item.id;
    els.editTitle.value = item.title || '';
    els.editContent.value = item.content || '';
    els.editContentEn.value = item.content_en || '';
    els.editModel.value = item.model || '';
    els.editMediaType.value = item.media_type || 'image';
    els.editTags.value = parseTagsToArray(item.tags).join(', ');
    els.editCategoryId.value = item.category_id || '';
    els.editCoverImage.value = item.cover_image_url || '';
    els.editSourceUrl.value = item.source_url || '';
    els.editPanel.style.display = 'block';
  }

  // 关闭编辑面板
  function closeEdit() {
    els.editPanel.style.display = 'none';
    editingId = null;
  }

  // 保存编辑
  async function saveEdit() {
    if (!editingId) return;
    const tagsInput = els.editTags.value.trim();
    const tagsArr = tagsInput ? tagsInput.split(/[,，]/).map(t => t.trim()).filter(t => t) : [];

    const updates = {
      title: els.editTitle.value.trim(),
      content: els.editContent.value.trim(),
      content_en: els.editContentEn.value.trim(),
      model: els.editModel.value.trim(),
      media_type: els.editMediaType.value,
      tags: JSON.stringify(tagsArr),
      category_id: els.editCategoryId.value,
      cover_image_url: els.editCoverImage.value.trim(),
      source_url: els.editSourceUrl.value.trim(),
    };

    await chrome.runtime.sendMessage({ action: 'updateBatchItem', id: editingId, updates });
    const idx = batchData.findIndex(i => i.id === editingId);
    if (idx >= 0) {
      batchData[idx] = { ...batchData[idx], ...updates };
    }
    renderBatchList();
    closeEdit();
    showGlobalStatus('✅ 修改已保存', 'success');
  }

  // 工具函数
  function parseTagsToArray(tags) {
    if (!tags) return [];
    try {
      const arr = JSON.parse(tags);
      if (Array.isArray(arr)) return arr;
    } catch (e) {}
    return [];
  }

  function escapeHtml(text) {
    if (!text) return '';
    const div = document.createElement('div');
    div.textContent = text;
    return div.innerHTML;
  }

  function toggleConfig() {
    const isHidden = els.configBody.style.display === 'none';
    els.configBody.style.display = isHidden ? 'block' : 'none';
    els.configArrow.textContent = isHidden ? '▼' : '▶';
  }

  async function saveConfig() {
    const data = {
      apiBaseUrl: els.apiBaseUrl.value.trim().replace(/\/$/, ''),
      apiToken: els.apiToken.value.trim(),
      userId: els.userId ? els.userId.value.trim() : '',
      defaultCategoryId: els.defaultCategoryId.value.trim()
    };
    const res = await chrome.runtime.sendMessage({ action: 'saveConfig', data });
    if (res.success) {
      config = data;
      showStatus(els.configStatus, '✅ 配置已保存', 'success');
      await loadCategories();
    } else {
      showStatus(els.configStatus, '❌ 保存失败', 'error');
    }
  }

  function showStatus(el, text, type) {
    el.textContent = text;
    el.className = 'status ' + type;
    if (type !== 'error') {
      setTimeout(() => { el.textContent = ''; el.className = 'status'; }, 3000);
    }
  }

  function showGlobalStatus(text, type) {
    els.globalStatus.textContent = text;
    els.globalStatus.className = 'status ' + type;
    if (type !== 'error') {
      setTimeout(() => { els.globalStatus.textContent = ''; els.globalStatus.className = 'status'; }, 5000);
    }
  }

  init();
})();
