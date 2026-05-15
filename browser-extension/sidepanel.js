// sidepanel.js - 侧边栏逻辑
// 固定在浏览器右侧，实时显示采集内容

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

    dataSection: document.getElementById('dataSection'),
    emptyState: document.getElementById('emptyState'),
    dataBadge: document.getElementById('dataBadge'),
    coverPreview: document.getElementById('coverPreview'),
    coverImg: document.getElementById('coverImg'),
    fieldTitle: document.getElementById('fieldTitle'),
    fieldContent: document.getElementById('fieldContent'),
    fieldContentEn: document.getElementById('fieldContentEn'),
    fieldDescription: document.getElementById('fieldDescription'),
    fieldModel: document.getElementById('fieldModel'),
    fieldMediaType: document.getElementById('fieldMediaType'),
    fieldTags: document.getElementById('fieldTags'),
    fieldCategoryId: document.getElementById('fieldCategoryId'),
    fieldCoverImage: document.getElementById('fieldCoverImage'),
    fieldSourceUrl: document.getElementById('fieldSourceUrl'),
    submitBtn: document.getElementById('submitBtn'),
    clearBtn: document.getElementById('clearBtn'),
    submitStatus: document.getElementById('submitStatus'),
  };

  let config = {};
  let extractedData = null;
  let categoriesCache = []; // 缓存分类列表

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

    // 预加载分类列表
    await loadCategories();

    await loadExtractedData();

    els.configToggle.addEventListener('click', toggleConfig);
    els.saveConfigBtn.addEventListener('click', saveConfig);
    els.submitBtn.addEventListener('click', submitPrompt);
    els.clearBtn.addEventListener('click', clearData);

    // 封面图 URL 变化时更新预览
    els.fieldCoverImage.addEventListener('input', updateCoverPreview);

    // 监听后台消息（新数据到达）
    chrome.runtime.onMessage.addListener((message) => {
      if (message.action === 'dataExtracted') {
        extractedData = message.data;
        renderData();
        flashBadge();
      }
    });
  }

  // 加载分类列表（从 new-api 获取）
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
        console.log('[SidePanel] 加载分类:', categoriesCache.length, '个');
      }
    } catch (e) {
      console.log('[SidePanel] 加载分类失败:', e.message);
    }
  }

  // 根据模型名称匹配分类 ID
  function matchCategoryId(modelName) {
    if (!modelName || categoriesCache.length === 0) return 0;
    const normalized = modelName.trim().toLowerCase();
    // 精确匹配
    for (const cat of categoriesCache) {
      if (cat.name && cat.name.trim().toLowerCase() === normalized) {
        return cat.id;
      }
    }
    // 包含匹配
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
    const res = await chrome.runtime.sendMessage({ action: 'getExtractedData' });
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

    els.fieldTitle.value = extractedData.title || '';
    els.fieldContent.value = extractedData.content || '';
    els.fieldContentEn.value = extractedData.content_en || '';
    els.fieldDescription.value = extractedData.description || '';
    els.fieldModel.value = extractedData.model || '';
    els.fieldMediaType.value = extractedData.media_type || 'image';
    els.fieldTags.value = parseTags(extractedData.tags);

    // 自动匹配模型 → 分类
    const matchedCategoryId = matchCategoryId(extractedData.model);
    els.fieldCategoryId.value = matchedCategoryId || config.defaultCategoryId || '';
    if (matchedCategoryId) {
      console.log('[SidePanel] 模型 "' + extractedData.model + '" 匹配到分类 ID:', matchedCategoryId);
    }

    els.fieldCoverImage.value = extractedData.cover_image_url || '';
    els.fieldSourceUrl.value = extractedData.source_url || '';

    updateCoverPreview();
  }

  // 更新封面图预览
  function updateCoverPreview() {
    const url = els.fieldCoverImage.value.trim();
    if (url) {
      els.coverPreview.style.display = 'block';
      els.coverImg.src = url;
      els.coverImg.onerror = () => {
        els.coverPreview.style.display = 'none';
      };
    } else {
      els.coverPreview.style.display = 'none';
      els.coverImg.src = '';
    }
  }

  // 新数据闪烁提示
  function flashBadge() {
    els.dataBadge.style.display = 'inline-block';
    setTimeout(() => {
      els.dataBadge.style.display = 'none';
    }, 3000);
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

    const res = await chrome.runtime.sendMessage({ action: 'saveConfig', data });
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
      media_type: els.fieldMediaType.value,
      tags: tagsStr,
      category_id: parseInt(els.fieldCategoryId.value) || 0,
      cover_image_url: els.fieldCoverImage.value.trim(),
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
      const res = await chrome.runtime.sendMessage({
        action: 'apiRequest',
        method: 'POST',
        path: '/prompt/',
        body: payload,
        userId: config.userId
      });

      if (res.success && res.data && res.data.success) {
        showStatus(els.submitStatus, '✅ 提交成功！已入库', 'success');
        await chrome.runtime.sendMessage({ action: 'clearExtractedData' });
        extractedData = null;
        setTimeout(() => {
          showEmpty();
          els.submitBtn.disabled = false;
          els.submitBtn.innerHTML = '<span class="btn-icon">⬆️</span> 提交到提示词库';
        }, 1500);
      } else {
        const msg = res.data?.message || res.message || '未知错误';
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
    await chrome.runtime.sendMessage({ action: 'clearExtractedData' });
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

  init();
})();
