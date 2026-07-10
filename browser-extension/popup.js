// popup.js - 弹窗逻辑
// 展示提取的提示词内容，编辑后提交到 new-api

(function () {
  'use strict';

  // DOM 元素
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

    dataSection: document.getElementById('dataSection'),
    emptyState: document.getElementById('emptyState'),
    fieldTitle: document.getElementById('fieldTitle'),
    fieldContent: document.getElementById('fieldContent'),
    fieldContentEn: document.getElementById('fieldContentEn'),
    fieldDescription: document.getElementById('fieldDescription'),
    fieldModel: document.getElementById('fieldModel'),
    fieldSource: document.getElementById('fieldSource'),
    fieldMediaType: document.getElementById('fieldMediaType'),
    fieldTags: document.getElementById('fieldTags'),
    fieldCategoryId: document.getElementById('fieldCategoryId'),
    fieldCoverImage: document.getElementById('fieldCoverImage'),
    fieldVideoUrl: document.getElementById('fieldVideoUrl'),
    fieldSourceUrl: document.getElementById('fieldSourceUrl'),
    submitBtn: document.getElementById('submitBtn'),
    clearBtn: document.getElementById('clearBtn'),
    submitStatus: document.getElementById('submitStatus'),

    extractBtn: document.getElementById('extractBtn'),
    addToBatchBtn: document.getElementById('addToBatchBtn'),
    openBatchBtn: document.getElementById('openBatchBtn'),
  };

  let config = {};
  let extractedData = null;
  let categoriesCache = [];

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
          // 等待 SW 重启
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
    // === 第1步：同步绑定事件 ===
    els.configToggle.addEventListener('click', toggleConfig);
    els.saveConfigBtn.addEventListener('click', saveConfig);
    els.submitBtn.addEventListener('click', submitPrompt);
    els.clearBtn.addEventListener('click', clearData);
    if (els.extractBtn) els.extractBtn.addEventListener('click', extractCurrentPage);
    if (els.addToBatchBtn) els.addToBatchBtn.addEventListener('click', addToBatch);
    if (els.openBatchBtn) els.openBatchBtn.addEventListener('click', openBatchPanel);
    const fetchBtn = document.getElementById('fetchTokenBtn');
    if (fetchBtn) fetchBtn.addEventListener('click', fetchToken);

    // === 第2步：异步加载数据 ===
    try {
      const configRes = await safeMsg({ action: 'getConfig' });
      if (configRes && configRes.success) {
        config = configRes.data || {};
        els.apiBaseUrl.value = config.apiBaseUrl || '';
        els.apiToken.value = config.apiToken || '';
        if (els.userId) els.userId.value = config.userId || '';
        els.defaultCategoryId.value = config.defaultCategoryId || '';
      }
    } catch (e) {
      console.debug('[Popup] 初始化加载失败:', e.message);
    }

    await loadCategories();
    await loadExtractedData();

    // 监听后台消息（新数据到达）
    chrome.runtime.onMessage.addListener((message) => {
      if (message.action === 'dataExtracted') {
        loadExtractedData();
      }
    });
  }

  // 加载分类列表（从 new-api 获取）
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
        console.log('[Popup] 加载分类:', categoriesCache.length, '个');
      }
    } catch (e) {
      console.log('[Popup] 加载分类失败:', e.message);
    }
  }

  // 根据模型名称匹配分类 ID
  function matchCategoryId(modelName) {
    if (!modelName || categoriesCache.length === 0) return 0;
    const normalized = modelName.trim().toLowerCase();
    for (const cat of categoriesCache) {
      if (cat.name && cat.name.trim().toLowerCase() === normalized) {
        return cat.id;
      }
    }
    for (const cat of categoriesCache) {
      const catName = (cat.name || '').trim().toLowerCase();
      if (catName.includes(normalized) || normalized.includes(catName)) {
        return cat.id;
      }
    }
    return 0;
  }

  // 加载提取的数据
  async function loadExtractedData() {
    const res = await safeMsg({ action: 'getExtractedData' });
    if (res.success && res.data) {
      extractedData = res.data;
      renderData();
    } else {
      showEmpty();
    }
  }

  // 渲染提取的数据到表单
  function renderData() {
    if (!extractedData) {
      showEmpty();
      return;
    }

    els.dataSection.style.display = 'block';
    els.emptyState.style.display = 'none';
    if (els.addToBatchBtn) els.addToBatchBtn.disabled = false;

    els.fieldTitle.value = extractedData.title || '';
    els.fieldContent.value = extractedData.content || '';
    els.fieldContentEn.value = extractedData.content_en || '';
    els.fieldDescription.value = extractedData.description || '';
    els.fieldModel.value = extractedData.model || '';
    els.fieldSource.value = extractedData.source || extractedData.author || '';
    els.fieldMediaType.value = extractedData.media_type || 'image';
    els.fieldTags.value = parseTags(extractedData.tags);

    // 自动匹配模型 → 分类
    const matchedCategoryId = matchCategoryId(extractedData.model);
    els.fieldCategoryId.value = matchedCategoryId || config.defaultCategoryId || '';

    els.fieldCoverImage.value = extractedData.cover_image_url || '';
    els.fieldVideoUrl.value = extractedData.video_url || '';
    els.fieldSourceUrl.value = extractedData.source_url || '';
  }

  // 解析标签
  function parseTags(tags) {
    if (!tags) return '';
    try {
      const arr = JSON.parse(tags);
      if (Array.isArray(arr)) return arr.join(', ');
    } catch (e) {
      return tags;
    }
    return '';
  }

  // 显示空状态
  function showEmpty() {
    els.dataSection.style.display = 'none';
    els.emptyState.style.display = 'block';
    if (els.addToBatchBtn) els.addToBatchBtn.disabled = true;
  }

  // 从当前活动标签页提取数据（通用）
  async function extractCurrentPage() {
    if (!els.extractBtn) return;
    const originalText = els.extractBtn.textContent;
    els.extractBtn.disabled = true;
    els.extractBtn.textContent = '提取中...';
    showStatus(els.submitStatus, '⏳ 正在抓取页面...（最多 8 秒）', 'info');
    try {
      const res = await safeMsg({ action: 'extractFromActiveTab' });
      console.log('[Popup] extractFromActiveTab 响应:', res);
      if (res && res.success && res.data) {
        extractedData = res.data;
        await safeMsg({ action: 'saveExtractedData', data: extractedData });
        renderData();
        const tip = res.reason === 'NO_PROMPT_DETECTED' ? '⚠️ 已抓取但未识别到提示词正文（可手动填）' : '✅ 提取成功';
        showStatus(els.submitStatus, tip, res.reason === 'NO_PROMPT_DETECTED' ? 'error' : 'success');
      } else {
        const detail = res?.message || res?.data?.message || '未能提取到提示词';
        showStatus(els.submitStatus, `❌ ${detail}`, 'error');
      }
    } catch (err) {
      showStatus(els.submitStatus, `❌ 提取失败: ${err.message}`, 'error');
    } finally {
      els.extractBtn.disabled = false;
      els.extractBtn.textContent = originalText;
    }
  }

  // 收集当前表单数据
  function gatherFormData() {
    const tagsInput = els.fieldTags.value.trim();
    let tags = [];
    if (tagsInput) {
      tags = tagsInput.split(/[,，]/).map(t => t.trim()).filter(t => t);
    }
    return {
      ...extractedData,
      title: els.fieldTitle.value.trim(),
      content: els.fieldContent.value.trim(),
      content_en: els.fieldContentEn.value.trim(),
      description: els.fieldDescription.value.trim(),
      model: els.fieldModel.value.trim(),
      source: els.fieldSource.value.trim(),
      author: els.fieldSource.value.trim(),
      media_type: els.fieldMediaType.value,
      category_id: parseInt(els.fieldCategoryId.value, 10) || 0,
      tags: JSON.stringify(tags),
      cover_image_url: els.fieldCoverImage.value.trim(),
      video_url: els.fieldVideoUrl ? els.fieldVideoUrl.value.trim() : '',
      source_url: els.fieldSourceUrl.value.trim()
    };
  }

  // 加入批量库
  async function addToBatch() {
    if (!extractedData) return;
    const data = gatherFormData();
    if (!data.title || (!data.content && !data.content_en)) {
      showStatus(els.submitStatus, '❌ 标题和 Prompt 内容不能为空', 'error');
      return;
    }
    try {
      const res = await safeMsg({ action: 'appendBatchData', data });
      if (res && res.success) {
        showStatus(els.submitStatus, '✅ 已加入批量库', 'success');
      } else {
        showStatus(els.submitStatus, `❌ ${res?.message || '加入失败'}`, 'error');
      }
    } catch (err) {
      showStatus(els.submitStatus, `❌ 加入失败: ${err.message}`, 'error');
    }
  }

  // 打开批量库侧边栏
  async function openBatchPanel() {
    try {
      await safeMsg({ action: 'openSidePanel' });
    } catch (err) {
      console.error('[Popup] 打开侧边栏失败:', err);
    }
  }

  // 切换配置面板
  function toggleConfig() {
    const isHidden = els.configBody.style.display === 'none';
    els.configBody.style.display = isHidden ? 'block' : 'none';
    els.configArrow.textContent = isHidden ? '▼' : '▶';
  }

  // 保存配置
  async function saveConfig() {
    const data = {
      apiBaseUrl: els.apiBaseUrl.value.trim().replace(/\/$/, ''),
      apiToken: els.apiToken.value.trim(),
      userId: els.userId ? els.userId.value.trim() : '',
      defaultCategoryId: els.defaultCategoryId.value.trim()
    };

    const res = await safeMsg({ action: 'saveConfig', data });
    if (res.success) {
      config = data;
      showStatus(els.configStatus, '✅ 配置已保存', 'success');
      // 配置更新后重新加载分类列表
      await loadCategories();
    } else {
      showStatus(els.configStatus, '❌ 保存失败', 'error');
    }
  }

  // 提交提示词
  async function submitPrompt() {
    const title = els.fieldTitle.value.trim();
    const content = els.fieldContent.value.trim();
    const contentEn = els.fieldContentEn.value.trim();

    if (!title) {
      showStatus(els.submitStatus, '❌ 标题不能为空', 'error');
      return;
    }
    if (!content && !contentEn) {
      showStatus(els.submitStatus, '❌ 中文或英文 Prompt 至少填一个', 'error');
      return;
    }
    if (!config.apiBaseUrl) {
      showStatus(els.submitStatus, '❌ 请先配置 API Base URL', 'error');
      toggleConfig();
      return;
    }

    // 组装标签为 JSON 数组字符串
    let tagsStr = '[]';
    const tagsInput = els.fieldTags.value.trim();
    if (tagsInput) {
      const tagsArr = tagsInput.split(/[,，]/).map(t => t.trim()).filter(t => t);
      tagsStr = JSON.stringify(tagsArr);
    }

    const payload = {
      title,
      content: content || contentEn,
      content_en: contentEn,
      description: els.fieldDescription.value.trim() || title,
      model: els.fieldModel.value.trim(),
      source: els.fieldSource.value.trim(),
      media_type: els.fieldMediaType.value,
      tags: tagsStr,
      category_id: parseInt(els.fieldCategoryId.value) || 0,
      cover_image_url: els.fieldCoverImage.value.trim(),
      video_url: els.fieldVideoUrl ? els.fieldVideoUrl.value.trim() : '',
      source_url: els.fieldSourceUrl ? els.fieldSourceUrl.value.trim() : '',
      status: 1,
      sort_order: 0,
      is_premium: false,
      unlock_cost: 0,
      author: '采集导入',
      i18n: '{}',
      seo_i18n: '{}'
    };

    els.submitBtn.disabled = true;
    els.submitBtn.textContent = '提交中...';
    showStatus(els.submitStatus, '⏳ 正在提交...', 'info');

    try {
      const res = await safeMsg({
        action: 'apiRequest',
        method: 'POST',
        path: '/prompt/',
        body: payload,
        userId: config.userId
      });

      if (res.success && res.data && (res.data.success === true || res.data.code === 0 || (res.data.code !== undefined && res.data.code === 0) || (res.data.message === '' && res.data.data))) {
        showStatus(els.submitStatus, '✅ 提交成功！已入库', 'success');
        // 清空提取数据
        await safeMsg({ action: 'clearExtractedData' });
        extractedData = null;
        setTimeout(() => {
          showEmpty();
          els.submitBtn.disabled = false;
          els.submitBtn.innerHTML = '<span class="btn-icon">⬆️</span> 提交到提示词库';
        }, 1500);
      } else {
        const detail = res?.data?.message
          || res?.data?.data?.message
          || res?.data?.detail
          || res?.message
          || res?.error
          || '未知错误';
        const msg = typeof detail === 'string' ? detail : JSON.stringify(detail);
        showStatus(els.submitStatus, `❌ 提交失败: ${msg}`, 'error');
        els.submitBtn.disabled = false;
        els.submitBtn.innerHTML = '<span class="btn-icon">⬆️</span> 提交到提示词库';
      }
    } catch (err) {
      showStatus(els.submitStatus, `❌ 网络错误: ${err.message}`, 'error');
      els.submitBtn.disabled = false;
      els.submitBtn.innerHTML = '<span class="btn-icon">⬆️</span> 提交到提示词库';
    }
  }

  // 清空数据
  async function clearData() {
    await safeMsg({ action: 'clearExtractedData' });
    extractedData = null;
    showEmpty();
    showStatus(els.submitStatus, '', 'info');
  }

  // 显示状态消息
  function showStatus(el, text, type) {
    el.textContent = text;
    el.className = 'status ' + type;
    if (type !== 'error') {
      setTimeout(() => {
        el.textContent = '';
        el.className = 'status';
      }, 3000);
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
        // 如果是 access_token，同时设置用户 ID
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

  init();
})();
