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
    autoSubmit: document.getElementById('autoSubmit'),
    categoryMapping: document.getElementById('categoryMapping'),
    saveConfigBtn: document.getElementById('saveConfigBtn'),
    testConnBtn: document.getElementById('testConnBtn'),
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
    editSource: document.getElementById('editSource'),
    editMediaType: document.getElementById('editMediaType'),
    editTags: document.getElementById('editTags'),
    editCategoryId: document.getElementById('editCategoryId'),
    editCoverImage: document.getElementById('editCoverImage'),
    editVideoUrl: document.getElementById('editVideoUrl'),
    editSourceUrl: document.getElementById('editSourceUrl'),
    saveEditBtn: document.getElementById('saveEditBtn'),
    cancelEditBtn: document.getElementById('cancelEditBtn'),

    globalStatus: document.getElementById('globalStatus'),
  };

  let config = {};
  let batchData = [];
  let categoriesCache = [];
  let editingId = null;

  // 安全消息发送：自动重试，处理 SW 失活 / context invalidated
  async function safeMsg(msg) {
    for (let i = 0; i < 3; i++) {
      try {
        return await chrome.runtime.sendMessage(msg);
      } catch (e) {
        const isContextErr = e.message?.includes('context invalidated') ||
                            e.message?.includes('Extension context') ||
                            e.message?.includes('Receiving end does not exist');
        if (i < 2 && isContextErr) {
          await new Promise(r => setTimeout(r, 400 * (i + 1)));
          continue;
        }
        return { success: false, message: e.message, errorType: 'CONTEXT_INVALIDATED' };
      }
    }
    return null;
  }

  // 初始化
  async function init() {
    // === 第1步：同步绑定所有事件（无论后台是否就绪，点击必须立即响应）===
    els.configToggle.addEventListener('click', toggleConfig);
    els.saveConfigBtn.addEventListener('click', saveConfig);
    els.testConnBtn.addEventListener('click', testConnection);
    els.checkAll.addEventListener('change', onCheckAll);
    els.batchSubmitBtn.addEventListener('click', batchSubmit);
    els.clearSubmittedBtn.addEventListener('click', clearSubmitted);
    els.clearAllBtn.addEventListener('click', clearAll);
    els.closeEditPanel.addEventListener('click', closeEdit);
    els.saveEditBtn.addEventListener('click', saveEdit);
    els.cancelEditBtn.addEventListener('click', closeEdit);

    // Token 获取面板事件
    const tokenToggle = document.getElementById('tokenFetchToggle');
    const fetchBtn2 = document.getElementById('fetchTokenBtn2');
    if (tokenToggle) {
      tokenToggle.addEventListener('click', () => {
        const body = document.getElementById('tokenFetchBody');
        const arrow = tokenToggle.querySelector('.arrow');
        if (!body) return;
        const isHidden = body.style.display === 'none';
        body.style.display = isHidden ? 'block' : 'none';
        if (arrow) arrow.textContent = isHidden ? '▼' : '▶';
      });
    }
    if (fetchBtn2) fetchBtn2.addEventListener('click', fetchToken2);

    // === 第2步：异步加载数据（失败不影响交互）===
    try {
      const configRes = await safeMsg({ action: 'getConfig' });
      if (configRes && configRes.success) {
        config = configRes.data || {};
        els.apiBaseUrl.value = config.apiBaseUrl || '';
        els.apiToken.value = config.apiToken || '';
        if (els.userId) els.userId.value = config.userId || '';
        els.defaultCategoryId.value = config.defaultCategoryId || '';
        if (els.autoSubmit) els.autoSubmit.checked = !!config.autoSubmit;
        if (els.categoryMapping) els.categoryMapping.value = config.categoryMapping || '';
      }
    } catch (e) {
      console.debug('[Sidepanel] 初始化加载失败:', e.message);
    }

    await loadCategories();
    await loadBatchData();

    // 监听后台消息（新数据到达）
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
      const res = await safeMsg({
        action: 'apiRequest',
        method: 'GET',
        path: '/prompt-category/all',
        userId: config.userId
      });
      if (res.success && res.data && res.data.success && Array.isArray(res.data.data)) {
        categoriesCache = res.data.data;
      }
    } catch (e) {
      console.debug('[Sidepanel] 加载分类列表失败:', e.message);
    }
  }

  // 解析映射配置
  function parseCategoryMapping() {
    const mapping = {};
    const text = config.categoryMapping || '';
    if (!text) return mapping;
    text.split('\n').forEach(line => {
      const [key, val] = line.split('=');
      if (key && val) {
        mapping[key.trim().toLowerCase()] = parseInt(val.trim()) || 0;
      }
    });
    return mapping;
  }

  // 根据模型名称匹配分类 ID（优先用映射，其次自动匹配）
  function matchCategoryId(modelName) {
    if (!modelName) return 0;
    const normalized = modelName.trim().toLowerCase();

    // 优先用自定义映射
    const mapping = parseCategoryMapping();
    if (mapping[normalized]) return mapping[normalized];

    // 其次自动匹配分类列表
    if (categoriesCache.length === 0) return 0;
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
    try {
      const res = await safeMsg({ action: 'getBatchData' });
      if (res && res.success) {
        batchData = res.batch || [];
        renderBatchList();
      }
    } catch (e) {
      console.debug('[Sidepanel] 加载批量数据失败:', e.message);
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

    const errorSummary = item.error
      ? escapeHtml(item.error.length > 80 ? item.error.slice(0, 80) + '…' : item.error)
      : '';

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
      ${item.error ? `<div class="batch-item-error" title="${escapeHtml(item.error)}">${errorSummary}</div>` : ''}
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
      await safeMsg({ action: 'updateBatchItem', id, updates: { submitted: item.submitted, error: item.error } });
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
    await safeMsg({ action: 'updateBatchItem', id, updates: { submitted: item.submitted, error: item.error } });
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
      source: item.source || '',
      media_type: item.media_type || 'image',
      tags: tagsStr,
      category_id: parseInt(item.category_id) || 0,
      cover_image_url: item.cover_image_url || '',
      video_url: item.video_url || '',
      status: 1,
      sort_order: 0,
      is_premium: false,
      unlock_cost: 0,
      author: item.author || item.source || '采集导入',
      i18n: '{}',
      seo_i18n: '{}'
    };

    try {
      const res = await safeMsg({
        action: 'apiRequest',
        method: 'POST',
        path: '/prompt/',
        body: payload,
        userId: config.userId
      });

      if (res.success && res.data && res.data.success) {
        return true;
      } else {
        item.error = res.data?.message || res.message || `提交失败 (HTTP ${res.status || '?'})`;
        console.error('[Sidepanel] 提交失败:', { title: item.title, error: item.error, res });
        return false;
      }
    } catch (err) {
      item.error = err.message || '网络错误';
      console.error('[Sidepanel] 提交异常:', err);
      return false;
    }
  }

  // 删除单条
  async function deleteOne(id) {
    await safeMsg({ action: 'removeBatchItem', id });
    batchData = batchData.filter(i => i.id !== id);
    renderBatchList();
  }

  // 清空已提交
  async function clearSubmitted() {
    await safeMsg({ action: 'clearSubmittedBatch' });
    batchData = batchData.filter(i => !i.submitted);
    renderBatchList();
    showGlobalStatus('✅ 已清空已提交的提示词', 'success');
  }

  // 清空全部
  async function clearAll() {
    if (!confirm('确定要清空所有采集的数据吗？')) return;
    await safeMsg({ action: 'clearBatchData' });
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
    els.editSource.value = item.source || '';
    els.editMediaType.value = item.media_type || 'image';
    els.editTags.value = parseTagsToArray(item.tags).join(', ');
    els.editCategoryId.value = item.category_id || '';
    els.editCoverImage.value = item.cover_image_url || '';
    els.editVideoUrl.value = item.video_url || '';
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
      source: els.editSource.value.trim(),
      media_type: els.editMediaType.value,
      tags: JSON.stringify(tagsArr),
      category_id: els.editCategoryId.value,
      cover_image_url: els.editCoverImage.value.trim(),
      video_url: els.editVideoUrl.value.trim(),
      source_url: els.editSourceUrl.value.trim(),
    };

    await safeMsg({ action: 'updateBatchItem', id: editingId, updates });
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
      defaultCategoryId: els.defaultCategoryId.value.trim(),
      autoSubmit: els.autoSubmit ? els.autoSubmit.checked : false,
      categoryMapping: els.categoryMapping ? els.categoryMapping.value.trim() : ''
    };
    const res = await safeMsg({ action: 'saveConfig', data });
    if (res.success) {
      config = data;
      showStatus(els.configStatus, '✅ 配置已保存', 'success');
      await loadCategories();
    } else {
      showStatus(els.configStatus, '❌ 保存失败', 'error');
    }
  }

  // 测试 API 连通性
  async function testConnection() {
    els.testConnBtn.disabled = true;
    els.testConnBtn.textContent = '测试中...';
    showStatus(els.configStatus, '⏳ 正在测试 API 连通性...', 'info');
    try {
      const res = await safeMsg({ action: 'testConnection' });
      if (res.success) {
        const d = res;
        if (d.ok) {
          showStatus(els.configStatus, `✅ API 连通正常（${d.latencyMs}ms, HTTP ${d.status}）`, 'success');
        } else if (d.status === 401) {
          showStatus(els.configStatus, `⚠️ API 可达但需认证（HTTP 401, ${d.latencyMs}ms）— 请检查 Token`, 'error');
        } else {
          showStatus(els.configStatus, `⚠️ HTTP ${d.status}（${d.latencyMs}ms）`, 'error');
        }
      } else {
        showStatus(els.configStatus, `${d.tip || res.message}`, 'error');
      }
    } catch (e) {
      showStatus(els.configStatus, `❌ 测试失败: ${e.message}`, 'error');
    } finally {
      els.testConnBtn.disabled = false;
      els.testConnBtn.textContent = '🔌 测试连接';
    }
  }

  // 一键获取 Token
  async function fetchToken() {
    const btn = document.getElementById('fetchTokenBtn');
    if (!btn) return;
    btn.disabled = true;
    btn.textContent = '获取中...';
    showStatus(els.configStatus, '⏳ 正在从管理后台获取 Token...', 'info');
    try {
      const res = await safeMsg({ action: 'fetchTokenFromTab' });
      if (res.success && res.token) {
        els.apiToken.value = res.isAccessToken ? `Bearer ${res.token}` : res.token;
        showStatus(els.configStatus, '✅ Token 已获取，请保存配置', 'success');
        if (res.isAccessToken && els.userId && !els.userId.value) {
          els.userId.value = '1';
        }
      } else {
        showStatus(els.configStatus, '❌ ' + (res.message || '获取失败'), 'error');
      }
    } catch (err) {
      showStatus(els.configStatus, '❌ 获取失败: ' + err.message, 'error');
    } finally {
      btn.disabled = false;
      btn.textContent = '🔄 获取';
    }
  }

  // 通用折叠面板切换
  function togglePanel(bodyId, arrowEl) {
    const body = document.getElementById(bodyId);
    if (!body) return;
    const isHidden = body.style.display === 'none';
    body.style.display = isHidden ? 'block' : 'none';
    if (arrowEl) arrowEl.textContent = isHidden ? '▼' : '▶';
  }

  // 从后台页面一键获取 Token
  async function fetchToken2() {
    const statusEl = document.getElementById('tokenFetchStatus');
    const btn = document.getElementById('fetchTokenBtn2');
    if (!btn) return;

    btn.disabled = true;
    btn.textContent = '获取中...';
    showStatus(statusEl, '⏳ 正在从管理后台获取 Token...', 'info');

    try {
      // 先保存当前配置（确保 API URL 已存在）
      await safeMsg({
        action: 'saveConfig',
        data: {
          apiBaseUrl: els.apiBaseUrl.value.trim().replace(/\/$/, ''),
          apiToken: els.apiToken.value.trim(),
          userId: els.userId ? els.userId.value.trim() : '1',
          defaultCategoryId: els.defaultCategoryId ? els.defaultCategoryId.value.trim() : '',
          autoSubmit: els.autoSubmit ? els.autoSubmit.checked : false,
          categoryMapping: els.categoryMapping ? els.categoryMapping.value.trim() : ''
        }
      });

      const res = await safeMsg({ action: 'fetchTokenFromTab' });
      if (res.success && res.token) {
        els.apiToken.value = res.isAccessToken ? `Bearer ${res.token}` : res.token;
        if (res.isAccessToken && els.userId && !els.userId.value) {
          els.userId.value = '1';
        }
        showStatus(statusEl, '✅ Token 已获取，请保存配置', 'success');
      } else {
        showStatus(statusEl, '❌ ' + (res.message || '获取失败') + '（需先打开 https://heharse.cloud 并登录）', 'error');
      }
    } catch (err) {
      showStatus(statusEl, '❌ 获取失败: ' + err.message, 'error');
    } finally {
      btn.disabled = false;
      btn.textContent = '🔄 一键获取 Token';
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
